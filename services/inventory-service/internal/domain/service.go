package domain

import (
	"context"

	"github.com/google/uuid"
)

type InventoryService interface {
	GetInventory(ctx context.Context, productID uuid.UUID) (*InventoryResponse, error)
	SetInventory(ctx context.Context, productID uuid.UUID, req UpdateInventoryRequest) (*InventoryResponse, error)
	HandleOrderCreated(ctx context.Context, event OrderCreatedEvent) error
	HandleOrderCancelled(ctx context.Context, event OrderCreatedEvent) error
}
