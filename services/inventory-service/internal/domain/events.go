package domain

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
	Close() error
}

const (
	// Consumed topics
	TopicOrderCreated   = "order.created"
	TopicOrderCancelled = "order.cancelled"

	// Published topics
	TopicInventoryUpdated  = "inventory.updated"
	TopicInventoryLowStock = "inventory.low_stock"
)
