package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v4"
	"realtime-ranking/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (ps *PostgresStore) CreateVideo(ctx context.Context, video *models.Video) error {
	video.CreatedAt = time.Now().UTC()
	video.UpdatedAt = time.Now().UTC()
	_, err := ps.pool.Exec(ctx,
		"INSERT INTO videos (id, title, data, score, views, likes, comments, shares, watch_time, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		video.ID, video.Title, video.Data, video.Score, video.Views, video.Likes, video.Comments, video.Shares, video.WatchTime, video.CreatedAt, video.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating video: %w", err)
	}
	return nil
}

func (ps *PostgresStore) UpdateVideo(ctx context.Context, video *models.Video) error {
	video.UpdatedAt = time.Now().UTC()
	_, err := ps.pool.Exec(ctx,
		"UPDATE videos SET title = $2, data = $3, score = $4, views = $5, likes = $6, comments = $7, shares = $8, watch_time = $9, updated_at = $10 WHERE id = $1",
		video.ID, video.Title, video.Data, video.Score, video.Views, video.Likes, video.Comments, video.Shares, video.WatchTime, video.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error updating video: %w", err)
	}
	return nil
}

func (ps *PostgresStore) GetVideo(ctx context.Context, videoID uuid.UUID) (*models.Video, error) {
	video := &models.Video{}
	err := ps.pool.QueryRow(ctx, "SELECT id, title, data, score, views, likes, comments, shares, watch_time, created_at, updated_at FROM videos WHERE id = $1", videoID).Scan(&video.ID, &video.Title, &video.Data, &video.Score, &video.Views, &video.Likes, &video.Comments, &video.Shares, &video.WatchTime, &video.CreatedAt, &video.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("video not found: %w", err)
		}
		return nil, fmt.Errorf("error getting video: %w", err)
	}
	return video, nil
}

func (ps *PostgresStore) Close() {
	ps.pool.Close()
}

func (ps *PostgresStore) GetUserVideoInteractions(ctx context.Context, userID string) ([]models.UserVideoInteraction, error) {
	rows, err := ps.pool.Query(ctx,
		`SELECT user_id, video_id, last_viewed, views, likes, comments, shares, watch_time
         FROM user_video_interactions
         WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying user video interactions: %w", err)
	}
	defer rows.Close()

	var interactions []models.UserVideoInteraction
	for rows.Next() {
		var interaction models.UserVideoInteraction
		if err := rows.Scan(&interaction.UserID, &interaction.VideoID, &interaction.LastViewed, &interaction.Views, &interaction.Likes, &interaction.Comments, &interaction.Shares, &interaction.WatchTime); err != nil {
			return nil, fmt.Errorf("error scanning user video interaction row: %w", err)
		}
		interactions = append(interactions, interaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over user video interaction rows: %w", err)
	}

	return interactions, nil
}

func (ps *PostgresStore) GetUserPreferences(ctx context.Context, userID string) (*models.UserPreference, error) {
	row := ps.pool.QueryRow(ctx,
		`SELECT user_id, categories, updated_at
         FROM user_preferences
         WHERE user_id = $1`, userID)

	var preferences models.UserPreference
	var categoriesJSON string
	if err := row.Scan(&preferences.UserID, &categoriesJSON, &preferences.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return &models.UserPreference{UserID: userID}, nil
		}
		return nil, fmt.Errorf("error scanning user preferences row: %w", err)
	}

	if err := json.Unmarshal([]byte(categoriesJSON), &preferences.Categories); err != nil {
		return nil, fmt.Errorf("error unmarshaling user preferences categories: %w", err)
	}

	return &preferences, nil
}

func (ps *PostgresStore) UpdateUserVideoInteraction(ctx context.Context, interaction *models.UserVideoInteraction) error {
	_, err := ps.pool.Exec(ctx,
		`INSERT INTO user_video_interactions (user_id, video_id, last_viewed, views, likes, comments, shares, watch_time)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
         ON CONFLICT (user_id, video_id)
         DO UPDATE SET
            last_viewed = $3,
            views = user_video_interactions.views + $4,
            likes = $5,
            comments = $6,
            shares = $7,
            watch_time = user_video_interactions.watch_time + $8`,
		interaction.UserID, interaction.VideoID, interaction.LastViewed, interaction.Views, interaction.Likes, interaction.Comments, interaction.Shares, interaction.WatchTime)
	if err != nil {
		return fmt.Errorf("error updating user video interaction: %w", err)
	}
	return nil
}

func (ps *PostgresStore) UpdateUserPreferences(ctx context.Context, preferences *models.UserPreference) error {
	categoriesJSON, err := json.Marshal(preferences.Categories)
	if err != nil {
		return fmt.Errorf("error marshaling user preferences categories: %w", err)
	}

	_, err = ps.pool.Exec(ctx,
		`INSERT INTO user_preferences (user_id, categories, updated_at)
         VALUES ($1, $2, $3)
         ON CONFLICT (user_id)
         DO UPDATE SET
            categories = $2,
            updated_at = $3`,
		preferences.UserID, categoriesJSON, preferences.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error updating user preferences: %w", err)
	}
	return nil
}