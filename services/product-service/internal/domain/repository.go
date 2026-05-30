package domain

import "github.com/google/uuid"

type ProductFilter struct {
	Q          string
	CategoryID *uuid.UUID
	MinPrice   *float64
	MaxPrice   *float64
	Sort       string
	Page       int
	Limit      int
}

type ProductListResponse struct {
	Products []Product
	Total    int64
	Page     int
	Limit    int
}

type ProductRepository interface {
	// Product operations
	GetProducts(filter ProductFilter) (*ProductListResponse, error)
	GetProductByID(id uuid.UUID) (*Product, error)
	CreateProduct(product *Product) (*Product, error)
	UpdateProduct(product *Product) (*Product, error)
	DeleteProduct(id uuid.UUID) error

	// Category operations
	GetCategories() ([]Category, error)
	GetCategoryByID(id uuid.UUID) (*Category, error)
	GetCategoryBySlug(slug string) (*Category, error)
	CreateCategory(category *Category) (*Category, error)

	// Image operations
	AddProductImage(image *ProductImage) (*ProductImage, error)
	GetProductImage(productID, imageID uuid.UUID) (*ProductImage, error)
	GetProductImages(productID uuid.UUID) ([]ProductImage, error)
	DeleteProductImage(productID, imageID uuid.UUID) error
	UpdateProductImagePositions(images []ProductImage) error
}
