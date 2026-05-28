package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"auron/notification-service/internal/domain"

	"github.com/segmentio/kafka-go"
)

type readerEntry struct {
	reader *kafka.Reader
	topic  string
}

type KafkaConsumer struct {
	readers []readerEntry
	service domain.NotificationService
}

func NewKafkaConsumer(brokers []string, svc domain.NotificationService) *KafkaConsumer {
	topics := []string{
		domain.TopicUserCreated,
		domain.TopicOrderCreated,
		domain.TopicOrderCancelled,
		domain.TopicPaymentCompleted,
		domain.TopicPaymentFailed,
		domain.TopicInventoryLowStock,
	}

	readers := make([]readerEntry, 0, len(topics))
	for _, topic := range topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "notification-service",
			MinBytes: 1,
			MaxBytes: 10e6,
		})
		readers = append(readers, readerEntry{reader: r, topic: topic})
	}

	return &KafkaConsumer{readers: readers, service: svc}
}

func (c *KafkaConsumer) Start(ctx context.Context) {
	for _, entry := range c.readers {
		go c.consumeTopic(ctx, entry)
	}
}

func (c *KafkaConsumer) Close() error {
	var lastErr error
	for _, entry := range c.readers {
		if err := entry.reader.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *KafkaConsumer) consumeTopic(ctx context.Context, entry readerEntry) {
	for {
		msg, err := entry.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("kafka read error", "topic", entry.topic, "error", err)
			continue
		}
		c.handleMessage(ctx, entry.topic, msg.Value)
	}
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, topic string, value []byte) {
	var err error
	switch topic {
	case domain.TopicUserCreated:
		var event domain.UserCreatedEvent
		if err = json.Unmarshal(value, &event); err != nil {
			break
		}
		err = c.service.HandleUserCreated(ctx, event)

	case domain.TopicOrderCreated:
		var event domain.OrderEvent
		if err = json.Unmarshal(value, &event); err != nil {
			break
		}
		err = c.service.HandleOrderCreated(ctx, event)

	case domain.TopicOrderCancelled:
		var event domain.OrderEvent
		if err = json.Unmarshal(value, &event); err != nil {
			break
		}
		err = c.service.HandleOrderCancelled(ctx, event)

	case domain.TopicPaymentCompleted:
		var event domain.PaymentEvent
		if err = json.Unmarshal(value, &event); err != nil {
			break
		}
		err = c.service.HandlePaymentCompleted(ctx, event)

	case domain.TopicPaymentFailed:
		var event domain.PaymentEvent
		if err = json.Unmarshal(value, &event); err != nil {
			break
		}
		err = c.service.HandlePaymentFailed(ctx, event)

	case domain.TopicInventoryLowStock:
		var event domain.InventoryLowStockEvent
		if err = json.Unmarshal(value, &event); err != nil {
			break
		}
		slog.Warn("inventory low stock",
			"product_id", event.ProductID,
			"available", event.AvailableQuantity,
			"total", event.TotalQuantity,
		)

	default:
		slog.Warn("kafka consumer received unknown topic", "topic", topic)
	}

	if err != nil {
		slog.Error("failed to handle kafka message", "topic", topic, "error", err)
	}
}
