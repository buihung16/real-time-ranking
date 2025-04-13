package store

import (
	"context"
	"realtime-ranking/models"
	"time"

	"github.com/google/uuid"
)

type IStore interface {
	CreateVideo(ctx context.Context, video *models.Video) error
	UpdateVideo(ctx context.Context, video *models.Video) error
	GetVideo(ctx context.Context, videoID uuid.UUID) (*models.Video, error)
	GetUserVideoInteractions(ctx context.Context, userID string) ([]models.UserVideoInteraction, error)
	GetUserPreferences(ctx context.Context, userID string) (*models.UserPreference, error)
	UpdateUserVideoInteraction(ctx context.Context, interaction *models.UserVideoInteraction) error
	UpdateUserPreferences(ctx context.Context, preferences *models.UserPreference) error

	UpdateVideoScore(ctx context.Context, videoID uuid.UUID, score float64) error
	GetTopVideos(ctx context.Context, start, stop int64) ([]models.Video, error)
	CacheUserPreferences(ctx context.Context, userID string, preferences models.UserPreference, expiration time.Duration) error
	GetCachedUserPreferences(ctx context.Context, userID string) (*models.UserPreference, error)
	DeleteCachedUserPreferences(ctx context.Context, userID string) error
	Close() error
}