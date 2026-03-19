package models

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID              uuid.UUID              `json:"id" gorm:"type:uuid;primary_key"`
	UserID          uuid.UUID              `json:"user_id" gorm:"type:uuid;not null"`
	Status          string                 `json:"status" gorm:"default:PENDING"`
	TotalAmount     float64                `json:"total_amount" gorm:"not null"`
	ShippingAddress map[string]interface{} `json:"shipping_address" gorm:"type:jsonb"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	OrderID     uuid.UUID `json:"order_id" gorm:"type:uuid;not null"`
	ProductID   uuid.UUID `json:"product_id" gorm:"type:uuid;not null"`
	ProductName string    `json:"product_name" gorm:"not null"`
	Price       float64   `json:"price" gorm:"not null"`
	Quantity    int       `json:"quantity" gorm:"not null"`
	Subtotal    float64   `json:"subtotal" gorm:"not null"`
}

func (OrderItem) TableName() string { return "order_items" }

type CartItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int    `json:"quantity"`
	Image     string `json:"image"`
}
