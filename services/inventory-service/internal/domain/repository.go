package domain

import "github.com/google/uuid"

type InventoryRepository interface {
	GetByProductID(productID uuid.UUID) (*Inventory, error)
	SetTotalQuantity(productID uuid.UUID, quantity int) (*Inventory, error)
	ReserveStock(productID uuid.UUID, quantity int) (*Inventory, error)
	ReleaseStock(productID uuid.UUID, quantity int) (*Inventory, error)
}
