package domain

import (
	"context"

	"github.com/google/uuid"
)

type InventoryCache interface {
	GetInventory(ctx context.Context, productID uuid.UUID) (*Inventory, error)
	SetInventory(ctx context.Context, inv *Inventory) error
	InvalidateInventory(ctx context.Context, productID uuid.UUID) error
}
