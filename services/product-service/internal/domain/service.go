package domain

import (
	"context"
	"github.com/google/uuid"
)

// ProductService defines the business logic for products and categories.
type ProductService interface {
	// Product operations
	GetProducts(ctx context.Context, filter ProductFilter) (*ProductListResponse, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error)
	CreateProduct(ctx context.Context, req ProductRequest) (*Product, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, req ProductRequest) (*Product, error)
	DeleteProduct(ctx context.Context, id uuid.UUID) error

	// Category operations
	GetCategories(ctx context.Context) ([]Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	CreateCategory(ctx context.Context, req CategoryRequest) (*Category, error)
}
