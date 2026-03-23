package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type MessageHandler func(ctx context.Context, msg kafka.Message) error

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
}

func NewConsumer(reader *kafka.Reader, handler MessageHandler) *Consumer {
	return &Consumer{reader: reader, handler: handler}
}

func (c *Consumer) Start(ctx context.Context) {
	slog.Info("kafka consumer started",
		slog.String("topic", c.reader.Config().Topic),
		slog.String("group", c.reader.Config().GroupID),
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("kafka consumer stoping")
				return
			}
			slog.Error("kafka fetch error", slog.Any("error", err))
			continue
		}
		if err := c.handler(ctx, msg); err != nil {
			slog.Error("kafka handler error",
				slog.String("topic", msg.Topic),
				slog.Int64("offset", msg.Offset),
				slog.Any("error", err),
			)
			continue
		}

		// Commit only after successful processing
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("kafka commit error", slog.Any("error", err))
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func LogUserEvent(ctx context.Context, msg kafka.Message) error {
	var payload map[string]any
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		slog.Error("failed to unmarshal user event", slog.Any("error", err))
		return err
	}

	slog.Info("received user event",
		slog.String("topic", msg.Topic),
		slog.Any("payload", payload),
		slog.Int64("offset", msg.Offset),
	)
	return nil
}
