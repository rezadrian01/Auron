package domain

import "github.com/google/uuid"

type PaymentRepository interface {
	GetPaymentByID(id uuid.UUID) (*Payment, error)
	GetPaymentByOrderID(orderID uuid.UUID) (*Payment, error)
	CreatePayment(payment *Payment) (*Payment, error)
	UpdatePaymentStatus(id uuid.UUID, status PaymentStatus, failureReason string) (*Payment, error)
	UpdateStripeIDs(id uuid.UUID, intentID, clientSecret string) (*Payment, error)
}
