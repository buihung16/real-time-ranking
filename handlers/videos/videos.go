package videos

import (
	"context"
	"net/http"
	"realtime-ranking/models"
	"realtime-ranking/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type VideoHandler struct {
	rankingService *services.RankingService
	validate       *validator.Validate
}

func NewVideoHandler(rankingService *services.RankingService, validate *validator.Validate) *VideoHandler {
	return &VideoHandler{rankingService: rankingService, validate: validate}
}

// CreateVideo godoc
// @Summary     Create a new video
// @Description Creates a new video with the given title and Base64 encoded data
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       video body models.CreateVideoRequest true "Video object to be created"
// @Success     201 {object} models.Video
// @Failure     400 {object} ErrorResponse
// @Failure     500 {object} ErrorResponse
// @Router      /videos [post]
func (vh *VideoHandler) CreateVideo(c *gin.Context) {
	var video models.CreateVideoRequest
	if err := c.ShouldBindJSON(&video); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request payload", Details: err.Error()})
		return
	}

	if err := vh.validate.Struct(video); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Validation error", Details: err.Error()})
		return
	}

	newVideo := models.Video{
		ID:    uuid.New(),
		Title: video.Title,
		Data:  video.Data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := vh.rankingService.CreateVideo(ctx, &newVideo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to create video", Details: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newVideo)
}

// UpdateVideo godoc
// @Summary     Update video
// @Description Updates a video's title and Base64 encoded data
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id    path   string                true "Video ID"
// @Param       video body models.UpdateVideoRequest true "Video object to be updated"
// @Success     200 {object} models.Video
// @Failure     400 {object} ErrorResponse
// @Failure     500 {object} ErrorResponse
// @Router      /videos/{id} [put]
func (vh *VideoHandler) UpdateVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid video ID", Details: err.Error()})
		return
	}

	var updateRequest models.UpdateVideoRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request payload", Details: err.Error()})
		return
	}

	if err := vh.validate.Struct(updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Validation error", Details: err.Error()})
		return
	}

	updatedVideo := models.Video{
		ID:    id,
		Title: updateRequest.Title,
		Data:  updateRequest.Data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = vh.rankingService.UpdateVideo(ctx, &updatedVideo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to update video", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedVideo)
}

// HandleView godoc
// @Summary     Handle video view event
// @Description Records a video view and updates the score
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id     path   string true "Video ID"
// @Param       userID query  string true "User ID"
// @Success     200    {object} SuccessResponse
// @Failure     400    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Router      /videos/{id}/view [post]
func (vh *VideoHandler) HandleView(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid video ID", Details: err.Error()})
		return
	}

	userID := c.Query("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "User ID is required", Details: "Missing userID query parameter"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := &models.VideoEvent{
		VideoID: id,
		Action:  models.ViewAction,
		UserID:  userID,
	}

	err = vh.rankingService.PublishVideoEvent(ctx, event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to record view", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "View recorded successfully"})
}

// HandleLike godoc
// @Summary     Handle video like event
// @Description Records a video like and updates the score
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id     path   string true "Video ID"
// @Param       userID query  string true "User ID"
// @Success     200    {object} SuccessResponse
// @Failure     400    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Router      /videos/{id}/like [post]
func (vh *VideoHandler) HandleLike(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid video ID", Details: err.Error()})
		return
	}

	userID := c.Query("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "User ID is required", Details: "Missing userID query parameter"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := &models.VideoEvent{
		VideoID: id,
		Action:  models.LikeAction,
		UserID:  userID,
	}

	err = vh.rankingService.PublishVideoEvent(ctx, event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to record like", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Like recorded successfully"})
}

// HandleComment godoc
// @Summary     Handle video comment event
// @Description Records a video comment and updates the score
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id     path   string true "Video ID"
// @Param       userID query  string true "User ID"
// @Success     200    {object} SuccessResponse
// @Failure     400    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Router      /videos/{id}/comment [post]
func (vh *VideoHandler) HandleComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid video ID", Details: err.Error()})
		return
	}

	userID := c.Query("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "User ID is required", Details: "Missing userID query parameter"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := &models.VideoEvent{
		VideoID: id,
		Action:  models.CommentAction,
		UserID:  userID,
	}

	err = vh.rankingService.PublishVideoEvent(ctx, event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to record comment", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Comment recorded successfully"})
}

// HandleShare godoc
// @Summary     Handle video share event
// @Description Records a video share and updates the score
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id     path   string true "Video ID"
// @Param       userID query  string true "User ID"
// @Success     200    {object} SuccessResponse
// @Failure     400    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Router      /videos/{id}/share [post]
func (vh *VideoHandler) HandleShare(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid video ID", Details: err.Error()})
		return
	}

	userID := c.Query("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "User ID is required", Details: "Missing userID query parameter"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := &models.VideoEvent{
		VideoID: id,
		Action:  models.ShareAction,
		UserID:  userID,
	}

	err = vh.rankingService.PublishVideoEvent(ctx, event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to record share", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Share recorded successfully"})
}

// WatchTimeAction handles the event when a user watches a video for a certain duration.
// @Summary Record video watch time
// @Description Records the amount of time a user watched a specific video and updates the video's watch time and potentially its ranking.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id     path   string true "Video ID"
// @Param       userID query  string true "User ID"
// @Param       duration query  string true "Duration"
// @Success     200    {object} SuccessResponse
// @Failure     400    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Router /videos/{id}/watch [post]
func (vh *VideoHandler) HandleWatch(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid video ID", Details: err.Error()})
		return
	}

	userID := c.Query("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "User ID is required", Details: "Missing userID query parameter"})
		return
	}

	durationStr := c.Query("duration")
	if durationStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Duration is required", Details: "Missing duration query parameter"})
		return
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid duration", Details: "Duration must be a positive integer"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := &models.VideoEvent{
		VideoID:  id,
		Action:   models.WatchTimeAction,
		UserID:   userID,
		Value: duration, // Include the duration in the event
	}

	err = vh.rankingService.PublishVideoEvent(ctx, event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to record watch", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Watch recorded successfully"})
}
// GetTopVideos godoc
// @Summary     Get top-ranked videos
// @Description Retrieve the top-ranked videos
// @Tags        videos
// @Produce     json
// @Param       start query int false "Start index"
// @Param       count query int false "Number of videos to retrieve"
// @Success     200   {array} models.Video
// @Failure     500   {object} ErrorResponse
// @Router      /videos/top [get]
func (vh *VideoHandler) GetTopVideos(c *gin.Context) {
	start, _ := strconv.ParseInt(c.DefaultQuery("start", "0"), 10, 64)
	count, _ := strconv.ParseInt(c.DefaultQuery("count", "10"), 10, 64)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	videos, err := vh.rankingService.GetTopVideos(ctx, start, count-1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to get top videos", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// GetTopVideosPerUser godoc
// @Summary     Get top-ranked videos for a user
// @Description Retrieve the top-ranked videos for a specific user
// @Tags        users
// @Produce     json
// @Param       userID path   string true "User ID"
// @Param       start  query  int    false "Start index"
// @Param       count  query  int    false "Number of videos to retrieve"
// @Success     200    {array} models.Video
// @Failure     500    {object} ErrorResponse
// @Router      /users/{userID}/videos/top [get]
func (vh *VideoHandler) GetTopVideosPerUser(c *gin.Context) {
	userID := c.Param("userID")
	start, _ := strconv.ParseInt(c.DefaultQuery("start", "0"), 10, 64)
	count, _ := strconv.ParseInt(c.DefaultQuery("count", "10"), 10, 64)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Longer timeout for personalization
	defer cancel()

	videos, err := vh.rankingService.GetTopVideosPerUser(ctx, userID, start, count-1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to get top videos for user", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// UpdateUserPreferences godoc
// @Summary     Update user's video category preferences
// @Description Updates a user's video category preferences
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       userID path   string                    true "User ID"
// @Param       prefs  body   models.UpdateUserPreferencesRequest true "User's video category preferences"
// @Success     200    {object} SuccessResponse
// @Failure     400    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Router      /users/{userID}/preferences [post]
func (vh *VideoHandler) UpdateUserPreferences(c *gin.Context) {
	userID := c.Param("userID")

	var prefsRequest models.UpdateUserPreferencesRequest
	if err := c.ShouldBindJSON(&prefsRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request payload", Details: err.Error()})
		return
	}

	if err := vh.validate.Struct(prefsRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Validation error", Details: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	preferences := &models.UserPreference{
		UserID:     userID,
		Categories: prefsRequest.Categories,
		UpdatedAt:  time.Now().UTC(),
	}

	err := vh.rankingService.UpdateUserPreferences(ctx, preferences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Failed to update user preferences", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "User preferences updated successfully"})
}

// ErrorResponse is a generic error response.
type ErrorResponse struct {
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse is a generic success response.
type SuccessResponse struct {
	Message string `json:"message"`
}