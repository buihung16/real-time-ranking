package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/segmentio/kafka-go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"realtime-ranking/consumer"
	"realtime-ranking/handlers"
	"realtime-ranking/handlers/videos"
	"realtime-ranking/services"
	"realtime-ranking/store"
	"syscall"
	"time"
	_ "realtime-ranking/docs"
)

// @title       Real-time Ranking API
// @version     1.0
// @description API for managing and retrieving real-time video rankings.
// @host        localhost:8080
// @BasePath    /
func main() {
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://myuser:mypassword@postgres:5432/mydb"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis:6379/0"
	}

	kafkaBrokersRaw := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokersRaw == "" {
		kafkaBrokersRaw = "kafka:9092"
	}
	kafkaBrokers := []string{kafkaBrokersRaw}

	// Use pgxpool for connection pooling
	pgConfig, err := pgxpool.ParseConfig(postgresURL)
	if err != nil {
		log.Fatalf("Failed to parse Postgres URL: %v", err)
	}
	pgPool, err := pgxpool.ConnectConfig(context.Background(), pgConfig)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer pgPool.Close()

	redisStore, err := store.NewRedisStore(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisStore.Close()

	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers...),
		Topic:    "video-events",
		Balancer: &kafka.LeastBytes{},
		// Configure for better performance and reliability
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    10 * time.Second,
		RequiredAcks:   kafka.RequireOne,
		MaxAttempts:    3,
		Async:          true, // Enable asynchronous writes
		Completion:     nil,  // Handle completion callbacks if needed
		BatchSize:      100,
		BatchTimeout:   time.Millisecond * 10,
		//Linger:         time.Millisecond * 5,
		//ReadBatchTimeout: time.Millisecond * 10,
	}
	defer kafkaWriter.Close()

	postgresStore := store.NewPostgresStore(pgPool)
	rankingService := services.NewRankingService(redisStore, postgresStore, kafkaWriter)
	videoEventHandler := handlers.NewVideoEventHandler(rankingService)

	router := gin.Default()

	// Register validator
	validate := validator.New()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	videoHandler := videos.NewVideoHandler(rankingService, validate)
	router.POST("/videos", videoHandler.CreateVideo)
	router.PUT("/videos/:id", videoHandler.UpdateVideo)
	router.POST("/videos/:id/view", videoHandler.HandleView)
	router.POST("/videos/:id/like", videoHandler.HandleLike)
	router.POST("/videos/:id/comment", videoHandler.HandleComment)
	router.POST("/videos/:id/share", videoHandler.HandleShare)
	router.POST("/videos/:id/watch", videoHandler.HandleWatch)
	router.GET("/videos/top", videoHandler.GetTopVideos)
	router.GET("/users/:userID/videos/top", videoHandler.GetTopVideosPerUser)
	router.POST("/users/:userID/preferences", videoHandler.UpdateUserPreferences)

	go consumer.ConsumeVideoEvents(kafkaBrokers, videoEventHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
		// Configure timeouts for better resilience
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server exited")
}