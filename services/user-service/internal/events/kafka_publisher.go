package events

import (
	"auron/user-service/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type kafkaPublisher struct {
	writers map[string]*kafka.Writer
}

func NewKafkaPublisher(writers map[string]*kafka.Writer) domain.EventPublisher {
	return &kafkaPublisher{writers: writers}
}

func (w *kafkaPublisher) Publish(ctx context.Context, topic string, payload any) error {
	writer, ok := w.writers[topic]
	if !ok {
		return fmt.Errorf("publisher: no writer registered for topic %q", topic)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("publisher: marshal payload: %w", err)
	}

	err = writer.WriteMessages(ctx, kafka.Message{
		Value: data,
	})
	if err != nil {
		return fmt.Errorf("publisher: write to topic %q: %w", topic, err)
	}

	slog.Debug("event published", slog.String("topic", topic))
	return nil
}
