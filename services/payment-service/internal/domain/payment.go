package domain

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type Payment struct {
	ID                    uuid.UUID     `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrderID               uuid.UUID     `json:"order_id" gorm:"type:uuid;not null;uniqueIndex"`
	UserID                uuid.UUID     `json:"user_id" gorm:"type:uuid;not null;index"`
	Amount                float64       `json:"amount" gorm:"type:numeric(12,2);not null"`
	Currency              string        `json:"currency" gorm:"type:varchar(10);not null;default:'usd'"`
	Status                PaymentStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending';index"`
	StripePaymentIntentID string        `json:"stripe_payment_intent_id,omitempty" gorm:"type:varchar(255)"`
	StripeClientSecret    string        `json:"-" gorm:"type:text"`
	FailureReason         string        `json:"failure_reason,omitempty" gorm:"type:text"`
	CreatedAt             time.Time     `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt             time.Time     `json:"updated_at" gorm:"not null;default:now()"`
}

func (Payment) TableName() string {
	return "payments"
}

type PaymentResponse struct {
	ID                    uuid.UUID     `json:"id"`
	OrderID               uuid.UUID     `json:"order_id"`
	UserID                uuid.UUID     `json:"user_id"`
	Amount                float64       `json:"amount"`
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	StripePaymentIntentID string        `json:"stripe_payment_intent_id,omitempty"`
	FailureReason         string        `json:"failure_reason,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

func (p *Payment) ToResponse() *PaymentResponse {
	return &PaymentResponse{
		ID:                    p.ID,
		OrderID:               p.OrderID,
		UserID:                p.UserID,
		Amount:                p.Amount,
		Currency:              p.Currency,
		Status:                p.Status,
		StripePaymentIntentID: p.StripePaymentIntentID,
		FailureReason:         p.FailureReason,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// PaymentCheckoutResponse is returned to the frontend after order placement.
// It includes client_secret so the frontend can confirm the payment via Stripe.js.
type PaymentCheckoutResponse struct {
	ID                    uuid.UUID     `json:"id"`
	OrderID               uuid.UUID     `json:"order_id"`
	UserID                uuid.UUID     `json:"user_id"`
	Amount                float64       `json:"amount"`
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	StripePaymentIntentID string        `json:"stripe_payment_intent_id,omitempty"`
	ClientSecret          string        `json:"client_secret,omitempty"`
	FailureReason         string        `json:"failure_reason,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

func (p *Payment) ToCheckoutResponse() *PaymentCheckoutResponse {
	return &PaymentCheckoutResponse{
		ID:                    p.ID,
		OrderID:               p.OrderID,
		UserID:                p.UserID,
		Amount:                p.Amount,
		Currency:              p.Currency,
		Status:                p.Status,
		StripePaymentIntentID: p.StripePaymentIntentID,
		ClientSecret:          p.StripeClientSecret,
		FailureReason:         p.FailureReason,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// OrderCreatedEvent is the shape of the Kafka message consumed from order-service.
// JSON tags must match the Order struct in order-service (ID is published as "id").
type OrderCreatedEvent struct {
	OrderID     uuid.UUID        `json:"id"`
	UserID      uuid.UUID        `json:"user_id"`
	TotalAmount float64          `json:"total_amount"`
	Items       []OrderEventItem `json:"items"`
}

type OrderEventItem struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
}
