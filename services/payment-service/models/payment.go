package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID                     uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	OrderID                uuid.UUID `json:"order_id" gorm:"type:uuid;uniqueIndex;not null"`
	UserID                 uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	StripePaymentIntentID  string    `json:"stripe_payment_intent_id" gorm:"uniqueIndex"`
	Amount                 float64   `json:"amount" gorm:"not null"`
	Currency               string    `json:"currency" gorm:"default:usd"`
	Status                 string    `json:"status" gorm:"default:PENDING"`
	StripeEventIDs         []string  `json:"stripe_event_ids" gorm:"type:text[]"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (Payment) TableName() string { return "payments" }
