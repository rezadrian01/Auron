package domain

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
	Close() error
}

const (
	TopicPaymentCreated   = "payment.created"
	TopicPaymentCompleted = "payment.completed"
	TopicPaymentFailed    = "payment.failed"
)
