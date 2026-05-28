package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	TopicUserCreated       = "user.created"
	TopicOrderCreated      = "order.created"
	TopicOrderCancelled    = "order.cancelled"
	TopicPaymentCompleted  = "payment.completed"
	TopicPaymentFailed     = "payment.failed"
	TopicInventoryLowStock = "inventory.low_stock"
)

// UserCreatedEvent matches the User struct published by user-service (json tags on domain.User).
type UserCreatedEvent struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

// OrderEvent matches the Order struct published by order-service.
// Both order.created and order.cancelled use the same payload shape.
type OrderEvent struct {
	ID              uuid.UUID        `json:"id"`
	UserID          uuid.UUID        `json:"user_id"`
	Status          string           `json:"status"`
	TotalAmount     float64          `json:"total_amount"`
	ShippingName    string           `json:"shipping_name"`
	ShippingAddress string           `json:"shipping_address"`
	Items           []OrderEventItem `json:"items"`
	CreatedAt       time.Time        `json:"created_at"`
}

type OrderEventItem struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Subtotal    float64   `json:"subtotal"`
}

// PaymentEvent matches the PaymentResponse struct published by payment-service.
type PaymentEvent struct {
	ID            uuid.UUID `json:"id"`
	OrderID       uuid.UUID `json:"order_id"`
	UserID        uuid.UUID `json:"user_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	FailureReason string    `json:"failure_reason,omitempty"`
}

// InventoryLowStockEvent matches the InventoryResponse struct published by inventory-service.
type InventoryLowStockEvent struct {
	ProductID         uuid.UUID `json:"product_id"`
	TotalQuantity     int       `json:"total_quantity"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
}
