package consumer

import (
	"context"
	"encoding/json"
	"log"
	"realtime-ranking/handlers"
	"realtime-ranking/models"
	"time"

	"github.com/segmentio/kafka-go"
)

func ConsumeVideoEvents(kafkaBrokers []string, eventHandler *handlers.VideoEventHandler) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: kafkaBrokers,
		Topic:   "video-events",
		GroupID: "ranking-service-consumer",
		// Configure for better performance and reliability
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     time.Millisecond * 10,
		ReadBackoffMin: time.Millisecond * 1,
		ReadBackoffMax: time.Millisecond * 100,
	})
	defer reader.Close()

	ctx := context.Background()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading message from Kafka: %v", err)
			continue
		}

		log.Printf("Received message at Topic:%v Partition:%v Offset:%v Key:%s Value:%s\n", msg.Topic, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))

		var event models.VideoEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshaling Kafka message: %v", err)
			continue
		}

		if err := eventHandler.ProcessVideoEvent(ctx, &event); err != nil {
			log.Printf("Error processing video event: %v", err)
		}
	}
}