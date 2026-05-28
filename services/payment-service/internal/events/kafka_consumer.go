package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"auron/payment-service/internal/domain"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader  *kafka.Reader
	service domain.PaymentService
}

func NewKafkaConsumer(brokers []string, topic, groupID string, service domain.PaymentService) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &KafkaConsumer{reader: reader, service: service}
}

// Start launches the consumer loop in a background goroutine.
func (c *KafkaConsumer) Start(ctx context.Context) {
	go func() {
		for {
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("kafka consumer: fetch error", "error", err)
				continue
			}

			var event domain.OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				slog.Error("kafka consumer: unmarshal error", "error", err, "offset", msg.Offset)
			} else if err := c.service.HandleOrderCreated(ctx, event); err != nil {
				slog.Error("kafka consumer: HandleOrderCreated failed", "order_id", event.OrderID, "error", err)
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				slog.Warn("kafka consumer: commit failed", "error", err)
			}
		}
	}()
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
