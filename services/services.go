package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"realtime-ranking/models"
	"realtime-ranking/store"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type RankingService struct {
	redisStore    *store.RedisStore
	postgresStore *store.PostgresStore
	kafkaWriter   *kafka.Writer
}

func NewRankingService(redisStore *store.RedisStore, postgresStore *store.PostgresStore, kafkaWriter *kafka.Writer) *RankingService {
	return &RankingService{redisStore: redisStore, postgresStore: postgresStore, kafkaWriter: kafkaWriter}
}

func (rs *RankingService) CreateVideo(ctx context.Context, video *models.Video) error {
	err := rs.postgresStore.CreateVideo(ctx, video)
	if err != nil {
		return fmt.Errorf("error creating video in postgres: %w", err)
	}
	if err := rs.updateVideoInRedis(ctx, video); err != nil {
		log.Printf("Error updating video in Redis: %v", err)
	}
	return nil
}

func (rs *RankingService) UpdateVideo(ctx context.Context, video *models.Video) error {
	err := rs.postgresStore.UpdateVideo(ctx, video)
	if err != nil {
		return fmt.Errorf("error updating video in postgres: %w", err)
	}
	if err := rs.updateVideoInRedis(ctx, video); err != nil {
		log.Printf("Error updating video in Redis: %v", err)
	}
	return nil
}

func (rs *RankingService) updateVideoInRedis(ctx context.Context, video *models.Video) error {
	return rs.redisStore.UpdateVideoScore(ctx, video.ID, video.Score)
}

func (rs *RankingService) GetTopVideos(ctx context.Context, start, stop int64) ([]models.Video, error) {
	redisVideos, err := rs.redisStore.GetTopVideos(ctx, start, stop)
	if err != nil {
		return nil, fmt.Errorf("error getting top videos from redis: %w", err)
	}

	videos := make([]models.Video, len(redisVideos))
	for i, rv := range redisVideos {
		video, err := rs.postgresStore.GetVideo(ctx, rv.ID)
		if err != nil {
			log.Printf("Error fetching video %s from Postgres: %v", rv.ID, err)
			continue
		}
		videos[i] = *video
	}

	return videos, nil
}

func (rs *RankingService) GetTopVideosPerUser(ctx context.Context, userID string, start, stop int64) ([]models.Video, error) {
	// 1.  Try to get user preferences from cache
	cachedPreferences, err := rs.redisStore.GetCachedUserPreferences(ctx, userID)
	if err != nil {
		log.Printf("Error getting cached user preferences: %v", err)
	}

	var userPreferences *models.UserPreference
	if cachedPreferences != nil {
		userPreferences = cachedPreferences
	} else {
		// 2.  If not in cache, get from Postgres
		prefs, err := rs.postgresStore.GetUserPreferences(ctx, userID)
		if err != nil {
			log.Printf("Error fetching user preferences from Postgres: %v", err)
			prefs = &models.UserPreference{UserID: userID} // Default to empty preferences
		}
		userPreferences = prefs

		// 3.  Cache the preferences (with a TTL)
		cacheErr := rs.redisStore.CacheUserPreferences(ctx, userID, *userPreferences, time.Hour)
		if cacheErr != nil {
			log.Printf("Error caching user preferences: %v", cacheErr)
		}
	}

	// 4.  Get user's video interaction history
	userInteractions, err := rs.postgresStore.GetUserVideoInteractions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching user video interactions: %w", err)
	}

	// 5.  Get global top videos (consider caching this too)
	globalTopVideos, err := rs.GetTopVideos(ctx, 0, 100) // Fetch a larger set to filter
	if err != nil {
		return nil, fmt.Errorf("error fetching global top videos: %w", err)
	}

	// 6.  Personalize ranking
	personalizedVideos := rs.personalizeVideoRanking(globalTopVideos, userInteractions, userPreferences)

	// 7.  Apply pagination
	if start > int64(len(personalizedVideos)) {
		return []models.Video{}, nil
	}
	if stop > int64(len(personalizedVideos)) {
		stop = int64(len(personalizedVideos)) - 1
	}
	return personalizedVideos[start : stop+1], nil
}

func (rs *RankingService) personalizeVideoRanking(videos []models.Video, userInteractions []models.UserVideoInteraction, userPreferences *models.UserPreference) []models.Video {
	interactionMap := make(map[uuid.UUID]models.UserVideoInteraction)
	for _, interaction := range userInteractions {
		interactionMap[interaction.VideoID] = interaction
	}

	videoCategoryMap := rs.getVideoCategoryMap(videos)

	for i := range videos {
		interaction, exists := interactionMap[videos[i].ID]
		if exists {
			// Apply boosts based on user interaction
			videos[i].Score += float64(interaction.Views) * 0.1
			videos[i].Score += float64(interaction.Likes) * 0.5
			videos[i].Score += float64(interaction.Comments) * 0.8
			videos[i].Score += float64(interaction.Shares) * 1.2
			videos[i].Score += float64(interaction.WatchTime) * 0.05

			// Apply a recency boost
			timeDiff := time.Since(interaction.LastViewed).Hours()
			recencyBoost := 1.0 / (1.0 + timeDiff/24.0)
			videos[i].Score *= recencyBoost
		}

		// Apply boosts based on user preferences
		if len(userPreferences.Categories) > 0 {
			videoCategories, ok := videoCategoryMap[videos[i].ID]
			if ok {
				for _, userCategory := range userPreferences.Categories {
					for _, videoCategory := range videoCategories {
						if userCategory == videoCategory {
							videos[i].Score += 3.0
							break
						}
					}
				}
			}
		}
	}

	sort.Slice(videos, func(i, j int) bool {
		return videos[i].Score > videos[j].Score
	})

	return videos
}

// Dummy implementation - replace with actual logic to fetch categories from DB or another service
func (rs *RankingService) getVideoCategoryMap(videos []models.Video) map[uuid.UUID][]string {
	categoryMap := make(map[uuid.UUID][]string)
	for _, video := range videos {
		categoryMap[video.ID] = []string{"default"} // Placeholder
	}
	return categoryMap
}

func (rs *RankingService) PublishVideoEvent(ctx context.Context, event *models.VideoEvent) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling video event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.VideoID.String()),
		Value: eventBytes,
	}

	if err := rs.kafkaWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("error writing message to kafka: %w", err)
	}
	return nil
}

func (rs *RankingService) GetVideo(ctx context.Context, videoID uuid.UUID) (*models.Video, error) {
	video, err := rs.postgresStore.GetVideo(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("error getting video from postgres: %w", err)
	}
	return video, nil
}

func (rs *RankingService) UpdateUserPreferences(ctx context.Context, preferences *models.UserPreference) error {
	if err := rs.postgresStore.UpdateUserPreferences(ctx, preferences); err != nil {
		return fmt.Errorf("error updating user preferences in postgres: %w", err)
	}

	// Invalidate cache
	err := rs.redisStore.DeleteCachedUserPreferences(ctx, preferences.UserID)
	if err != nil {
		log.Printf("Error deleting cached user preferences: %v", err) // Log, but don't fail
	}

	return nil
}

func (rs *RankingService) UpdateUserVideoInteraction(ctx context.Context, interaction *models.UserVideoInteraction) error {
	if err := rs.postgresStore.UpdateUserVideoInteraction(ctx, interaction); err != nil {
		return fmt.Errorf("error updating user video interaction in postgres: %w", err)
	}
	return nil
}