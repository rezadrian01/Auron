package domain

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

const (
	TopicProductCreated = "product.created"
	TopicProductUpdated = "product.updated"
	TopicProductDeleted = "product.deleted"
)
