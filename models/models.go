package models

import (
	"github.com/google/uuid"
	"time"
)

const (
	ViewAction      = "view"
	LikeAction      = "like"
	CommentAction   = "comment"
	ShareAction     = "share"
	WatchTimeAction = "watch_time"
)

type Video struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Data      string    `json:"data"`
	Score     float64   `json:"score"`
	Views     int       `json:"views"`
	Likes     int       `json:"likes"`
	Comments  int       `json:"comments"`
	Shares    int       `json:"shares"`
	WatchTime int       `json:"watchTime"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateVideoRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
	Data  string `json:"data" binding:"required"`
}

type UpdateVideoRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
	Data  string `json:"data" binding:"required"`
}

type VideoEvent struct {
	VideoID uuid.UUID `json:"video_id"`
	Action  string    `json:"action"`
	UserID  string    `json:"user_id"`
	Value   interface{} `json:"value,omitempty"`
}

type UserVideoInteraction struct {
	UserID    string    `json:"userId"`
	VideoID   uuid.UUID `json:"videoId"`
	LastViewed time.Time `json:"lastViewed"`
	Views     int       `json:"views"`
	Likes     int       `json:"likes"`
	Comments  int       `json:"comments"`
	Shares    int       `json:"shares"`
	WatchTime int       `json:"watchTime"`
}

type UserPreference struct {
	UserID     string   `json:"userId"`
	Categories []string `json:"categories"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UpdateUserPreferencesRequest struct {
	Categories []string `json:"categories" binding:"required,dive,min=1,max=255"`
}