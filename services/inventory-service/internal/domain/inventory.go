package domain

import (
	"time"

	"github.com/google/uuid"
)

const LowStockThreshold = 10

type Inventory struct {
	ProductID        uuid.UUID `json:"product_id" gorm:"type:uuid;primaryKey"`
	TotalQuantity    int       `json:"total_quantity" gorm:"not null;default:0"`
	ReservedQuantity int       `json:"reserved_quantity" gorm:"not null;default:0"`
	Version          int       `json:"version" gorm:"not null;default:0"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (Inventory) TableName() string {
	return "inventory"
}

func (i *Inventory) AvailableQuantity() int {
	return i.TotalQuantity - i.ReservedQuantity
}

func (i *Inventory) ToResponse() *InventoryResponse {
	return &InventoryResponse{
		ProductID:         i.ProductID,
		TotalQuantity:     i.TotalQuantity,
		ReservedQuantity:  i.ReservedQuantity,
		AvailableQuantity: i.AvailableQuantity(),
		UpdatedAt:         i.UpdatedAt,
	}
}

type InventoryResponse struct {
	ProductID         uuid.UUID `json:"product_id"`
	TotalQuantity     int       `json:"total_quantity"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type UpdateInventoryRequest struct {
	TotalQuantity int `json:"total_quantity" binding:"required,min=0"`
}

// OrderCreatedEvent is the shape of the Kafka message from order-service.
// Used for both order.created and order.cancelled — order-service publishes
// the full Order struct for both events, which has json:"id" for the order ID.
type OrderCreatedEvent struct {
	OrderID uuid.UUID        `json:"id"`
	UserID  uuid.UUID        `json:"user_id"`
	Items   []OrderEventItem `json:"items"`
}

type OrderEventItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}
