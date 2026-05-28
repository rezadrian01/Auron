package domain

import "context"

// CartCache defines Redis cache operations for carts.
type CartCache interface {
	GetCart(ctx context.Context, userID string) (*Cart, error)
	SetCart(ctx context.Context, cart *Cart) error
	InvalidateCart(ctx context.Context, userID string) error
}

// OrderCache defines Redis cache operations for orders.
type OrderCache interface {
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	SetOrder(ctx context.Context, order *Order) error
	InvalidateOrder(ctx context.Context, orderID string) error
	GetOrderList(ctx context.Context, cacheKey string) (*OrderListResponse, error)
	SetOrderList(ctx context.Context, cacheKey string, resp *OrderListResponse) error
	InvalidateOrderList(ctx context.Context, userID string) error
}
