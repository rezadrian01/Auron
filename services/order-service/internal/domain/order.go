package domain

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// CONSTANTS
// ============================================================

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// cancellable returns true if the order can still be cancelled.
func (s OrderStatus) Cancellable() bool {
	return s == OrderStatusPending || s == OrderStatusConfirmed || s == OrderStatusProcessing
}

// ============================================================
// ENTITIES
// ============================================================

type Order struct {
	ID              uuid.UUID   `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          uuid.UUID   `json:"user_id" gorm:"type:uuid;not null;index"`
	Status          OrderStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending';index"`
	TotalAmount     float64     `json:"total_amount" gorm:"type:decimal(12,2);not null"`
	ShippingName    string      `json:"shipping_name" gorm:"type:varchar(255);not null"`
	ShippingAddress string      `json:"shipping_address" gorm:"type:text;not null"`
	Items           []OrderItem `json:"items,omitempty" gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	CreatedAt       time.Time   `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time   `json:"updated_at" gorm:"not null;default:now()"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrderID     uuid.UUID `json:"order_id" gorm:"type:uuid;not null;index"`
	ProductID   uuid.UUID `json:"product_id" gorm:"type:uuid;not null"`
	ProductName string    `json:"product_name" gorm:"type:varchar(500);not null"`
	Price       float64   `json:"price" gorm:"type:decimal(12,2);not null"`
	Quantity    int       `json:"quantity" gorm:"not null"`
	Subtotal    float64   `json:"subtotal" gorm:"type:decimal(12,2);not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (OrderItem) TableName() string {
	return "order_items"
}

// ============================================================
// DTOs — REQUESTS
// ============================================================

type CreateOrderRequest struct {
	ShippingName    string `json:"shipping_name" binding:"required"`
	ShippingAddress string `json:"shipping_address" binding:"required"`
}

// ============================================================
// DTOs — RESPONSES
// ============================================================

type OrderItemResponse struct {
	ID          uuid.UUID `json:"id"`
	OrderID     uuid.UUID `json:"order_id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Subtotal    float64   `json:"subtotal"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderResponse struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"user_id"`
	Status          OrderStatus        `json:"status"`
	TotalAmount     float64            `json:"total_amount"`
	ShippingName    string             `json:"shipping_name"`
	ShippingAddress string             `json:"shipping_address"`
	Items           []OrderItemResponse `json:"items"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type OrderListResponse struct {
	Orders []Order `json:"orders"`
	Total  int64   `json:"total"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func (o *Order) ToResponse() *OrderResponse {
	resp := &OrderResponse{
		ID:              o.ID,
		UserID:          o.UserID,
		Status:          o.Status,
		TotalAmount:     o.TotalAmount,
		ShippingName:    o.ShippingName,
		ShippingAddress: o.ShippingAddress,
		Items:           make([]OrderItemResponse, 0, len(o.Items)),
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}

	for _, item := range o.Items {
		resp.Items = append(resp.Items, OrderItemResponse{
			ID:          item.ID,
			OrderID:     item.OrderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
			CreatedAt:   item.CreatedAt,
		})
	}

	return resp
}
