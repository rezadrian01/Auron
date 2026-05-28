package domain

import "github.com/google/uuid"

// CartRepository defines persistence operations for carts and their items.
type CartRepository interface {
	GetCartByUserID(userID uuid.UUID) (*Cart, error)
	CreateCart(cart *Cart) (*Cart, error)
	GetCartItemByID(itemID uuid.UUID) (*CartItem, error)
	CreateCartItem(item *CartItem) (*CartItem, error)
	UpdateCartItem(item *CartItem) (*CartItem, error)
	DeleteCartItem(itemID uuid.UUID) error
	ClearCart(cartID uuid.UUID) error
}

// OrderRepository defines persistence operations for orders.
type OrderRepository interface {
	GetOrdersByUserID(userID uuid.UUID, offset, limit int) ([]Order, int64, error)
	GetOrderByID(orderID uuid.UUID) (*Order, error)
	CreateOrder(order *Order) (*Order, error)
	UpdateOrderStatus(orderID uuid.UUID, status OrderStatus) (*Order, error)
}
