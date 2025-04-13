package handlers

import (
	"context"
	"fmt"
	"log"
	"realtime-ranking/models"
	"realtime-ranking/services"
	"time"
)

type VideoEventHandler struct {
	rankingService *services.RankingService
}

func NewVideoEventHandler(rankingService *services.RankingService) *VideoEventHandler {
	return &VideoEventHandler{rankingService: rankingService}
}

func (vh *VideoEventHandler) ProcessVideoEvent(ctx context.Context, event *models.VideoEvent) error {
	video, err := vh.rankingService.GetVideo(ctx, event.VideoID)
	if err != nil {
		return fmt.Errorf("error getting video %s: %w", event.VideoID, err)
	}

	switch event.Action {
	case models.ViewAction:
		video.Views++
		video.Score += 1
	case models.LikeAction:
		video.Likes++
		video.Score += 5
	case models.CommentAction:
		video.Comments++
		video.Score += 3
	case models.ShareAction:
		video.Shares++
		video.Score += 4
	case models.WatchTimeAction:
		watchTime, ok := event.Value.(float64)
		if !ok {
			return fmt.Errorf("invalid watch time value: %v", event.Value)
		}
		video.WatchTime += int(watchTime)
		video.Score += watchTime * 0.05 // Reduced weight for watch time
	default:
		return fmt.Errorf("unknown action: %s", event.Action)
	}

	if err := vh.rankingService.UpdateVideo(ctx, video); err != nil {
		return fmt.Errorf("error updating video %s: %w", video.ID, err)
	}

	// Update user interaction history (with error handling)
	interaction := &models.UserVideoInteraction{
		UserID:    event.UserID,
		VideoID:   event.VideoID,
		LastViewed: time.Now().UTC(),
		Views:     0,
		Likes:     0,
		Comments:  0,
		Shares:    0,
		WatchTime: 0,
	}

	switch event.Action {
	case models.ViewAction:
		interaction.Views = 1
	case models.LikeAction:
		interaction.Likes = 1
	case models.CommentAction:
		interaction.Comments = 1
	case models.ShareAction:
		interaction.Shares = 1
	case models.WatchTimeAction:
		interaction.WatchTime = int(event.Value.(float64))
	}

	if err := vh.rankingService.UpdateUserVideoInteraction(ctx, interaction); err != nil {
		log.Printf("Error updating user video interaction: %v", err)
	}

	log.Printf("Processed event for video %s: Action=%s, New Score=%.2f\n", video.ID, event.Action, video.Score)
	return nil
}