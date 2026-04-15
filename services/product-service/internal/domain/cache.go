package domain

import "context"

type ProductCache interface {
	// product detail cache
	GetProduct(ctx context.Context, id string) (*Product, error)
	SetProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, id string) error

	// product list cache (paginated result)
	GetProductList(ctx context.Context, cacheKey string) (*ProductListResponse, error)
	SetProductList(ctx context.Context, cacheKey string, response *ProductListResponse) error
	InvalidateProductList(ctx context.Context) error

	// utility
	ClearAll(ctx context.Context) error
}
