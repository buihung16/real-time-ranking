package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"realtime-ranking/models"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(redisURL string) (*RedisStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(options)

	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	return &RedisStore{client: client}, nil
}

func (rs *RedisStore) UpdateVideoScore(ctx context.Context, videoID uuid.UUID, score float64) error {
	return rs.client.ZAdd(ctx, "video_ranking", &redis.Z{
		Score:  score,
		Member: videoID.String(),
	}).Err()
}

func (rs *RedisStore) GetTopVideos(ctx context.Context, start, stop int64) ([]models.Video, error) {
	results, err := rs.client.ZRevRange(ctx, "video_ranking", start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get top videos from redis: %w", err)
	}

	videos := make([]models.Video, len(results))
	for i, videoIDStr := range results {
		videoID, _ := uuid.Parse(videoIDStr)
		score, err := rs.client.ZScore(ctx, "video_ranking", videoIDStr).Result() // `score` is already float64
		if err != nil {
			log.Printf("Error getting score for video %s from Redis: %v", videoIDStr, err)
			continue
		}

		videos[i] = models.Video{
			ID:    videoID,
			Score: score, // No need to use strconv.ParseFloat
		}
	}
	return videos, nil
}

func (rs *RedisStore) Close() error {
	return rs.client.Close()
}

func (rs *RedisStore) CacheUserPreferences(ctx context.Context, userID string, preferences models.UserPreference, expiration time.Duration) error {
	preferencesJSON, err := json.Marshal(preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal user preferences: %w", err)
	}

	return rs.client.Set(ctx, fmt.Sprintf("user:preferences:%s", userID), preferencesJSON, expiration).Err()
}

func (rs *RedisStore) GetCachedUserPreferences(ctx context.Context, userID string) (*models.UserPreference, error) {
	preferencesJSON, err := rs.client.Get(ctx, fmt.Sprintf("user:preferences:%s", userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get user preferences from redis: %w", err)
	}

	var preferences models.UserPreference
	if err := json.Unmarshal([]byte(preferencesJSON), &preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user preferences: %w", err)
	}
	return &preferences, nil
}

func (rs *RedisStore) DeleteCachedUserPreferences(ctx context.Context, userID string) error {
	_, err := rs.client.Del(ctx, fmt.Sprintf("user:preferences:%s", userID)).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to delete cached user preferences: %w", err)
	}
	return nil
}