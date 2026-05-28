package domain

import (
	"context"

	"github.com/google/uuid"
)

type PaymentCache interface {
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
	SetPayment(ctx context.Context, payment *Payment) error
	InvalidatePayment(ctx context.Context, paymentID uuid.UUID) error
}
