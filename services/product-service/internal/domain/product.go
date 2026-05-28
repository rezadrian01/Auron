package domain

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// CONSTANTS
// ============================================================

const (
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortNewest    = "newest"
	SortNameAsc   = "name_asc"
	SortNameDesc  = "name_desc"
)

var ValidSorts = map[string]bool{
	SortPriceAsc:  true,
	SortPriceDesc: true,
	SortNewest:    true,
	SortNameAsc:   true,
	SortNameDesc:  true,
}

// ============================================================
// ENTITIES
// ============================================================

type Category struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string     `json:"name" gorm:"type:varchar(255);not null"`
	Slug      string     `json:"slug" gorm:"type:varchar(255);not null;uniqueIndex"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" gorm:"type:uuid;index"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null;default:now()"`
}

func (Category) TableName() string {
	return "categories"
}

type Product struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CategoryID   uuid.UUID       `json:"category_id" gorm:"type:uuid;not null;index"`
	Name         string          `json:"name" gorm:"type:varchar(500);not null"`
	Description  string          `json:"description" gorm:"type:text"`
	Price        float64         `json:"price" gorm:"type:numeric(12,2);not null;index"`
	ImageURL     string          `json:"image_url" gorm:"type:text"`
	SearchVector string          `json:"-" gorm:"-"`
	IsActive     bool            `json:"is_active" gorm:"not null;default:true;index"`
	CreatedAt    time.Time       `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"not null;default:now()"`
	Category     *Category       `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
}

func (Product) TableName() string {
	return "products"
}

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

// AvailableQuantity returns the stock available for purchase.
func (i *Inventory) AvailableQuantity() int {
	return i.TotalQuantity - i.ReservedQuantity
}

// ============================================================
// DTOs — REQUESTS
// ============================================================

type CategoryRequest struct {
	Name     string     `json:"name" binding:"required"`
	Slug     string     `json:"slug" binding:"required"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

type ProductRequest struct {
	CategoryID  uuid.UUID       `json:"category_id" binding:"required"`
	Name        string          `json:"name" binding:"required,max=500"`
	Description string          `json:"description" binding:"required"`
	Price       float64         `json:"price" binding:"required,gt=0"`
	ImageURL    string          `json:"image_url" binding:"omitempty,url"`
	IsActive    *bool           `json:"is_active"`
}

// ============================================================
// DTOs — RESPONSES
// ============================================================

type CategoryResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type ProductResponse struct {
	ID          uuid.UUID       `json:"id"`
	CategoryID  uuid.UUID       `json:"category_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Price       float64         `json:"price"`
	ImageURL    string          `json:"image_url,omitempty"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Category    *Category       `json:"category,omitempty"`
}

type InventoryResponse struct {
	ProductID         uuid.UUID `json:"product_id"`
	TotalQuantity     int       `json:"total_quantity"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	Version           int       `json:"version"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// ToResponse converts a Product entity to its response DTO.
func (p *Product) ToResponse() *ProductResponse {
	resp := &ProductResponse{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		ImageURL:    p.ImageURL,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	if p.Category != nil {
		resp.Category = p.Category
	}

	return resp
}

// ToResponse converts a Category entity to its response DTO.
func (c *Category) ToResponse() *CategoryResponse {
	return &CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		ParentID:  c.ParentID,
		CreatedAt: c.CreatedAt,
	}
}

// ToResponse converts an Inventory entity to its response DTO.
func (i *Inventory) ToResponse() *InventoryResponse {
	return &InventoryResponse{
		ProductID:         i.ProductID,
		TotalQuantity:     i.TotalQuantity,
		ReservedQuantity:  i.ReservedQuantity,
		AvailableQuantity: i.AvailableQuantity(),
		Version:           i.Version,
		UpdatedAt:         i.UpdatedAt,
	}
}
