package domain

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// ENTITIES
// ============================================================

type Cart struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	Items     []CartItem `json:"items,omitempty" gorm:"foreignKey:CartID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"not null;default:now()"`
}

func (Cart) TableName() string {
	return "carts"
}

type CartItem struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CartID      uuid.UUID `json:"cart_id" gorm:"type:uuid;not null;index"`
	ProductID   uuid.UUID `json:"product_id" gorm:"type:uuid;not null"`
	ProductName string    `json:"product_name" gorm:"type:varchar(500);not null"`
	Price       float64   `json:"price" gorm:"type:decimal(12,2);not null"`
	Quantity    int       `json:"quantity" gorm:"not null;default:1"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (CartItem) TableName() string {
	return "cart_items"
}

// ============================================================
// DTOs — REQUESTS
// ============================================================

type AddItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

// ============================================================
// DTOs — RESPONSES
// ============================================================

type CartItemResponse struct {
	ID          uuid.UUID `json:"id"`
	CartID      uuid.UUID `json:"cart_id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Subtotal    float64   `json:"subtotal"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CartResponse struct {
	ID        uuid.UUID         `json:"id"`
	UserID    uuid.UUID         `json:"user_id"`
	Items     []CartItemResponse `json:"items"`
	Total     float64            `json:"total"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func (c *Cart) ToResponse() *CartResponse {
	resp := &CartResponse{
		ID:        c.ID,
		UserID:    c.UserID,
		Items:     make([]CartItemResponse, 0, len(c.Items)),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}

	var total float64
	for _, item := range c.Items {
		subtotal := item.Price * float64(item.Quantity)
		total += subtotal
		resp.Items = append(resp.Items, CartItemResponse{
			ID:          item.ID,
			CartID:      item.CartID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	resp.Total = total

	return resp
}
