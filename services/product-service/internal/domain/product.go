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

func (Category) TableName() string { return "categories" }

// ProductImage represents a single image attached to a product.
// Position 0 is the primary (display) image shown on product cards.
type ProductImage struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProductID uuid.UUID `json:"product_id" gorm:"type:uuid;not null;index"`
	URL       string    `json:"url" gorm:"type:text;not null"`
	Position  int       `json:"position" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (ProductImage) TableName() string { return "product_images" }

type Product struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CategoryID   uuid.UUID      `json:"category_id" gorm:"type:uuid;not null;index"`
	Name         string         `json:"name" gorm:"type:varchar(500);not null"`
	Description  string         `json:"description" gorm:"type:text"`
	Price        float64        `json:"price" gorm:"type:numeric(12,2);not null;index"`
	Images       []ProductImage `json:"images,omitempty" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:CASCADE"`
	SearchVector string         `json:"-" gorm:"-"`
	IsActive     bool           `json:"is_active" gorm:"not null;default:true;index"`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"not null;default:now()"`
	Category     *Category      `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
}

func (Product) TableName() string { return "products" }

type Inventory struct {
	ProductID        uuid.UUID `json:"product_id" gorm:"type:uuid;primaryKey"`
	TotalQuantity    int       `json:"total_quantity" gorm:"not null;default:0"`
	ReservedQuantity int       `json:"reserved_quantity" gorm:"not null;default:0"`
	Version          int       `json:"version" gorm:"not null;default:0"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (Inventory) TableName() string { return "inventory" }

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

// ProductRequest no longer includes ImageURL — images are managed via
// POST /products/:id/images after the product is created.
type ProductRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	Name        string    `json:"name" binding:"required,max=500"`
	Description string    `json:"description" binding:"required"`
	Price       float64   `json:"price" binding:"required,gt=0"`
	IsActive    *bool     `json:"is_active"`
}

type ReorderImagesRequest struct {
	ImageIDs []uuid.UUID `json:"image_ids" binding:"required,min=1"`
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
	ID          uuid.UUID      `json:"id"`
	CategoryID  uuid.UUID      `json:"category_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Price       float64        `json:"price"`
	ImageURL    string         `json:"image_url"` // computed: images[0].URL or ""
	Images      []ProductImage `json:"images"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Category    *Category      `json:"category,omitempty"`
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

func (p *Product) ToResponse() *ProductResponse {
	resp := &ProductResponse{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Images:      p.Images,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Category:    p.Category,
	}
	if resp.Images == nil {
		resp.Images = []ProductImage{} // always array, never null in JSON
	}
	if len(p.Images) > 0 {
		resp.ImageURL = p.Images[0].URL // position 0 = primary
	}
	return resp
}

func (c *Category) ToResponse() *CategoryResponse {
	return &CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		ParentID:  c.ParentID,
		CreatedAt: c.CreatedAt,
	}
}

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
