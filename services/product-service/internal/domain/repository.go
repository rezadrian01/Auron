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
	Total   int64
	Page    int
	Limit   int
}

type ProductRepository interface {
	// product operations
	GetProducts(filter ProductFilter) (*ProductListResponse, error)
	GetProductByID(id uuid.UUID) (*Product, error)
	CreateProduct(product *Product) (*Product, error)
	UpdateProduct(product *Product) (*Product, error)
	DeleteProduct(id uuid.UUID) error

	// category operations
	GetCategories() ([]Category, error)
	GetCategoryByID(id uuid.UUID) (*Category, error)
	GetCategoryBySlug(slug string) (*Category, error)
	CreateCategory(category *Category) (*Category, error)
}
