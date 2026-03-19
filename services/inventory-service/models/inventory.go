package models

import (
	"time"

	"github.com/google/uuid"
)

type Inventory struct {
	ProductID        uuid.UUID `json:"product_id" gorm:"type:uuid;primary_key"`
	TotalQuantity    int       `json:"total_quantity" gorm:"default:0"`
	ReservedQuantity int       `json:"reserved_quantity" gorm:"default:0"`
	Version          int       `json:"version" gorm:"default:0"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Inventory) TableName() string { return "inventory" }
