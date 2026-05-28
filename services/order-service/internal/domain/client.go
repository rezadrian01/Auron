package domain

import (
	"context"

	"github.com/google/uuid"
)

// ProductSnapshot holds the product fields snapshotted at the time an item is added to a cart.
type ProductSnapshot struct {
	ID       uuid.UUID
	Name     string
	Price    float64
	IsActive bool
}

// ProductClient fetches product data from the product-service.
type ProductClient interface {
	GetProduct(ctx context.Context, id uuid.UUID) (*ProductSnapshot, error)
}
