package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"auron/inventory-service/internal/domain"

	"github.com/segmentio/kafka-go"
)

type readerEntry struct {
	reader *kafka.Reader
	topic  string
}

type KafkaConsumer struct {
	readers []readerEntry
	service domain.InventoryService
}

func NewKafkaConsumer(brokers []string, service domain.InventoryService) *KafkaConsumer {
	topics := []string{domain.TopicOrderCreated, domain.TopicOrderCancelled}
	entries := make([]readerEntry, 0, len(topics))
	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "inventory-service-orders",
			MinBytes: 10e3,
			MaxBytes: 10e6,
		})
		entries = append(entries, readerEntry{reader: reader, topic: topic})
	}
	return &KafkaConsumer{readers: entries, service: service}
}

// Start launches one consumer goroutine per subscribed topic.
func (c *KafkaConsumer) Start(ctx context.Context) {
	for _, entry := range c.readers {
		go c.consumeTopic(ctx, entry)
	}
}

func (c *KafkaConsumer) consumeTopic(ctx context.Context, entry readerEntry) {
	for {
		msg, err := entry.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("kafka consumer: fetch error", "topic", entry.topic, "error", err)
			continue
		}

		if err := c.handleMessage(ctx, entry.topic, msg.Value); err != nil {
			slog.Error("kafka consumer: handle error", "topic", entry.topic, "offset", msg.Offset, "error", err)
		}

		if err := entry.reader.CommitMessages(ctx, msg); err != nil {
			slog.Warn("kafka consumer: commit failed", "topic", entry.topic, "error", err)
		}
	}
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, topic string, payload []byte) error {
	switch topic {
	case domain.TopicOrderCreated:
		var event domain.OrderCreatedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("unmarshal order.created: %w", err)
		}
		return c.service.HandleOrderCreated(ctx, event)
	case domain.TopicOrderCancelled:
		var event domain.OrderCreatedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("unmarshal order.cancelled: %w", err)
		}
		return c.service.HandleOrderCancelled(ctx, event)
	default:
		slog.Warn("kafka consumer: unhandled topic", "topic", topic)
		return nil
	}
}

func (c *KafkaConsumer) Close() error {
	var closeErr error
	for _, entry := range c.readers {
		if err := entry.reader.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}
