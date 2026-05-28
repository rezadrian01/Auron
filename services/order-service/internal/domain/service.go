package domain

import (
	"context"

	"github.com/google/uuid"
)

// CartService defines the business logic for cart operations.
type CartService interface {
	GetCart(ctx context.Context, userID uuid.UUID) (*Cart, error)
	AddItem(ctx context.Context, userID uuid.UUID, req AddItemRequest) (*Cart, error)
	UpdateItem(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*Cart, error)
	RemoveItem(ctx context.Context, userID, itemID uuid.UUID) error
}

// OrderService defines the business logic for order operations.
type OrderService interface {
	GetOrders(ctx context.Context, userID uuid.UUID, page, limit int) (*OrderListResponse, error)
	CreateOrder(ctx context.Context, userID uuid.UUID, req CreateOrderRequest) (*Order, error)
	GetOrderByID(ctx context.Context, userID, orderID uuid.UUID) (*Order, error)
	CancelOrder(ctx context.Context, userID, orderID uuid.UUID) (*Order, error)
}
