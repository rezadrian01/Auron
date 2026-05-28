package domain

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

const (
	TopicOrderCreated   = "order.created"
	TopicOrderUpdated   = "order.updated"
	TopicOrderCancelled = "order.cancelled"
)
