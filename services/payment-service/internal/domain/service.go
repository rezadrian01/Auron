package domain

import (
	"context"

	"github.com/google/uuid"
)

type PaymentService interface {
	GetPaymentByID(ctx context.Context, userID, paymentID uuid.UUID) (*PaymentResponse, error)
	HandleOrderCreated(ctx context.Context, event OrderCreatedEvent) error
	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error
}
