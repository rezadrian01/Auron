package domain

import "context"

type NotificationService interface {
	HandleUserCreated(ctx context.Context, event UserCreatedEvent) error
	HandleOrderCreated(ctx context.Context, event OrderEvent) error
	HandleOrderCancelled(ctx context.Context, event OrderEvent) error
	HandlePaymentCompleted(ctx context.Context, event PaymentEvent) error
	HandlePaymentFailed(ctx context.Context, event PaymentEvent) error
}
