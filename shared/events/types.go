// Package events defines shared event types for Kafka message handling across all Auron services.
package events

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// EVENT STRUCTURES
// ============================================================

// Event is the base event structure for all Kafka messages
type Event struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// BaseEvent creates a new event with the given type and payload
func BaseEvent(eventType string, payload interface{}) Event {
	return Event{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

// ============================================================
// USER EVENTS
// ============================================================

// UserRegisteredPayload is the payload for user.registered events
type UserRegisteredPayload struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// UserRegistered represents a new user registration event
type UserRegistered struct {
	Event
	Payload UserRegisteredPayload `json:"payload"`
}

// NewUserRegistered creates a new user registered event
func NewUserRegistered(userID, email, name string) UserRegistered {
	return UserRegistered{
		Event: BaseEvent("user.registered", nil),
		Payload: UserRegisteredPayload{
			UserID:    userID,
			Email:     email,
			Name:      name,
			CreatedAt: time.Now().UTC(),
		},
	}
}

// ============================================================
// ORDER EVENTS
// ============================================================

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Subtotal  float64 `json:"subtotal"`
}

// ShippingAddress represents a shipping address
type ShippingAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
}

// OrderCreatedPayload is the payload for order.created events
type OrderCreatedPayload struct {
	OrderID          string           `json:"order_id"`
	UserID            string           `json:"user_id"`
	UserEmail         string           `json:"user_email"`
	Items             []OrderItem      `json:"items"`
	TotalAmount       float64          `json:"total_amount"`
	ShippingAddress   ShippingAddress  `json:"shipping_address"`
	CreatedAt         time.Time        `json:"created_at"`
}

// OrderCreated represents an order creation event
type OrderCreated struct {
	Event
	Payload OrderCreatedPayload `json:"payload"`
}

// NewOrderCreated creates a new order created event
func NewOrderCreated(orderID, userID, userEmail string, items []OrderItem, total float64, address ShippingAddress) OrderCreated {
	return OrderCreated{
		Event: BaseEvent("order.created", nil),
		Payload: OrderCreatedPayload{
			OrderID:          orderID,
			UserID:            userID,
			UserEmail:         userEmail,
			Items:             items,
			TotalAmount:       total,
			ShippingAddress:   address,
			CreatedAt:         time.Now().UTC(),
		},
	}
}

// OrderCancelledPayload is the payload for order.cancelled events
type OrderCancelledPayload struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// OrderCancelled represents an order cancellation event
type OrderCancelled struct {
	Event
	Payload OrderCancelledPayload `json:"payload"`
}

// ============================================================
// PAYMENT EVENTS
// ============================================================

// PaymentProcessedPayload is the payload for payment.processed events
type PaymentProcessedPayload struct {
	OrderID                string    `json:"order_id"`
	PaymentID              string    `json:"payment_id"`
	UserID                 string    `json:"user_id"`
	Amount                 float64   `json:"amount"`
	Currency               string    `json:"currency"`
	StripePaymentIntentID  string    `json:"stripe_payment_intent_id"`
	ProcessedAt            time.Time `json:"processed_at"`
}

// PaymentProcessed represents a successful payment event
type PaymentProcessed struct {
	Event
	Payload PaymentProcessedPayload `json:"payload"`
}

// NewPaymentProcessed creates a new payment processed event
func NewPaymentProcessed(orderID, paymentID, userID string, amount float64, currency, stripeID string) PaymentProcessed {
	return PaymentProcessed{
		Event: BaseEvent("payment.processed", nil),
		Payload: PaymentProcessedPayload{
			OrderID:               orderID,
			PaymentID:             paymentID,
			UserID:                userID,
			Amount:                amount,
			Currency:              currency,
			StripePaymentIntentID: stripeID,
			ProcessedAt:           time.Now().UTC(),
		},
	}
}

// PaymentFailedPayload is the payload for payment.failed events
type PaymentFailedPayload struct {
	OrderID   string    `json:"order_id"`
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Reason    string    `json:"reason"`
	FailedAt  time.Time `json:"failed_at"`
}

// PaymentFailed represents a failed payment event
type PaymentFailed struct {
	Event
	Payload PaymentFailedPayload `json:"payload"`
}

// NewPaymentFailed creates a new payment failed event
func NewPaymentFailed(orderID, paymentID, userID string, amount float64, reason string) PaymentFailed {
	return PaymentFailed{
		Event: BaseEvent("payment.failed", nil),
		Payload: PaymentFailedPayload{
			OrderID:  orderID,
			PaymentID: paymentID,
			UserID:   userID,
			Amount:   amount,
			Reason:   reason,
			FailedAt: time.Now().UTC(),
		},
	}
}

// ============================================================
// INVENTORY EVENTS
// ============================================================

// InventoryUpdatedPayload is the payload for inventory.updated events
type InventoryUpdatedPayload struct {
	ProductID        string    `json:"product_id"`
	OrderID          string    `json:"order_id"`
	ReservedQuantity int       `json:"reserved_quantity"`
	TotalQuantity    int       `json:"total_quantity"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// InventoryUpdated represents an inventory reservation event
type InventoryUpdated struct {
	Event
	Payload InventoryUpdatedPayload `json:"payload"`
}

// NewInventoryUpdated creates a new inventory updated event
func NewInventoryUpdated(productID, orderID string, reserved, total int) InventoryUpdated {
	return InventoryUpdated{
		Event: BaseEvent("inventory.updated", nil),
		Payload: InventoryUpdatedPayload{
			ProductID:        productID,
			OrderID:          orderID,
			ReservedQuantity: reserved,
			TotalQuantity:    total,
			UpdatedAt:        time.Now().UTC(),
		},
	}
}

// InventoryFailedPayload is the payload for inventory.failed events
type InventoryFailedPayload struct {
	ProductID   string    `json:"product_id"`
	OrderID     string    `json:"order_id"`
	Reason      string    `json:"reason"`
	FailedAt    time.Time `json:"failed_at"`
}

// InventoryFailed represents a failed inventory reservation event
type InventoryFailed struct {
	Event
	Payload InventoryFailedPayload `json:"payload"`
}

// NewInventoryFailed creates a new inventory failed event
func NewInventoryFailed(productID, orderID, reason string) InventoryFailed {
	return InventoryFailed{
		Event: BaseEvent("inventory.failed", nil),
		Payload: InventoryFailedPayload{
			ProductID: productID,
			OrderID:   orderID,
			Reason:    reason,
			FailedAt:  time.Now().UTC(),
		},
	}
}

// ============================================================
// NOTIFICATION EVENTS
// ============================================================

// NotificationPayload is the payload for notification events
type NotificationPayload struct {
	UserID     string            `json:"user_id"`
	Email      string            `json:"email"`
	Phone      string            `json:"phone,omitempty"`
	Type       string            `json:"type"`
	Subject    string            `json:"subject"`
	Body       string            `json:"body"`
	TemplateID string            `json:"template_id,omitempty"`
	Data       map[string]string `json:"data,omitempty"`
}

// Notification represents a notification event
type Notification struct {
	Event
	Payload NotificationPayload `json:"payload"`
}

// NewNotification creates a new notification event
func NewNotification(userID, email, notificationType, subject, body string) Notification {
	return Notification{
		Event: BaseEvent("notification.send", nil),
		Payload: NotificationPayload{
			UserID:  userID,
			Email:   email,
			Type:    notificationType,
			Subject: subject,
			Body:    body,
		},
	}
}

// ============================================================
// ENUM DEFINITIONS
// ============================================================

// Order status constants
const (
	OrderStatusPending    = "PENDING"
	OrderStatusConfirmed  = "CONFIRMED"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusShipped    = "SHIPPED"
	OrderStatusDelivered  = "DELIVERED"
	OrderStatusCancelled  = "CANCELLED"
	OrderStatusFailed     = "FAILED"
)

// Payment status constants
const (
	PaymentStatusPending   = "PENDING"
	PaymentStatusCompleted = "COMPLETED"
	PaymentStatusFailed    = "FAILED"
	PaymentStatusRefunded  = "REFUNDED"
)

// Notification type constants
const (
	NotificationTypeEmail = "email"
	NotificationTypeSMS   = "sms"
)

// Event type constants
const (
	EventUserRegistered    = "user.registered"
	EventOrderCreated      = "order.created"
	EventOrderCancelled    = "order.cancelled"
	EventPaymentProcessed  = "payment.processed"
	EventPaymentFailed     = "payment.failed"
	EventInventoryUpdated  = "inventory.updated"
	EventInventoryFailed   = "inventory.failed"
	EventNotificationSend = "notification.send"
)
