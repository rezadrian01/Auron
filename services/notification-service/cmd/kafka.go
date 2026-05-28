package cmd

import (
	"context"
	"log/slog"
	"strings"

	"auron/notification-service/internal/domain"
	"auron/notification-service/internal/events"
)

func setupKafkaConsumer(kafkaBrokers string, svc domain.NotificationService) *events.KafkaConsumer {
	brokers := parseBrokers(kafkaBrokers)
	return events.NewKafkaConsumer(brokers, svc)
}

func startKafkaConsumer(ctx context.Context, consumer *events.KafkaConsumer) {
	consumer.Start(ctx)
	slog.Info("kafka consumer started",
		"topics", []string{
			domain.TopicUserCreated,
			domain.TopicOrderCreated,
			domain.TopicOrderCancelled,
			domain.TopicPaymentCompleted,
			domain.TopicPaymentFailed,
			domain.TopicInventoryLowStock,
		},
		"group", "notification-service",
	)
}

func closeKafkaConsumer(consumer *events.KafkaConsumer) {
	if err := consumer.Close(); err != nil {
		slog.Warn("failed to close kafka consumer", "error", err)
	}
}

func parseBrokers(kafkaBrokers string) []string {
	parts := strings.Split(kafkaBrokers, ",")
	brokers := make([]string, 0, len(parts))
	for _, b := range parts {
		if trimmed := strings.TrimSpace(b); trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	if len(brokers) == 0 {
		return []string{"localhost:9092"}
	}
	return brokers
}
