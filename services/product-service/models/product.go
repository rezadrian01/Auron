package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name      string    `json:"name" gorm:"not null"`
	Slug      string    `json:"slug" gorm:"uniqueIndex;not null"`
	ParentID  *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	CreatedAt time.Time `json:"created_at"`
}

func (Category) TableName() string { return "categories" }

type Product struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	CategoryID   *uuid.UUID `json:"category_id" gorm:"type:uuid"`
	Name         string    `json:"name" gorm:"not null"`
	Description  string    `json:"description"`
	Price        float64   `json:"price" gorm:"not null"`
	ImageURL     string    `json:"image_url"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Product) TableName() string { return "products" }

type Inventory struct {
	ProductID        uuid.UUID `json:"product_id" gorm:"type:uuid;primary_key"`
	TotalQuantity    int       `json:"total_quantity" gorm:"default:0"`
	ReservedQuantity int       `json:"reserved_quantity" gorm:"default:0"`
	Version          int       `json:"version" gorm:"default:0"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Inventory) TableName() string { return "inventory" }
