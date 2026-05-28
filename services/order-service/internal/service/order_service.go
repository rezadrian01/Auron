package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"auron/order-service/internal/domain"

	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo  domain.OrderRepository
	cartRepo   domain.CartRepository
	orderCache domain.OrderCache
	cartCache  domain.CartCache
	publisher  domain.EventPublisher
}

func NewOrderService(
	orderRepo domain.OrderRepository,
	cartRepo domain.CartRepository,
	orderCache domain.OrderCache,
	cartCache domain.CartCache,
	publisher domain.EventPublisher,
) domain.OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		cartRepo:   cartRepo,
		orderCache: orderCache,
		cartCache:  cartCache,
		publisher:  publisher,
	}
}

func (s *OrderService) GetOrders(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.OrderListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	cacheKey := buildOrderListCacheKey(userID.String(), page, limit)
	if cached, err := s.orderCache.GetOrderList(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	offset := (page - 1) * limit
	orders, total, err := s.orderRepo.GetOrdersByUserID(userID, offset, limit)
	if err != nil {
		return nil, err
	}

	resp := &domain.OrderListResponse{
		Orders: orders,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}

	if err := s.orderCache.SetOrderList(ctx, cacheKey, resp); err != nil {
		slog.Warn("failed to cache order list", "user_id", userID, "error", err)
	}
	return resp, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req domain.CreateOrderRequest) (*domain.Order, error) {
	cart, err := s.cartRepo.GetCartByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) {
			return nil, domain.ErrCartEmpty
		}
		return nil, err
	}

	if len(cart.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}

	var orderItems []domain.OrderItem
	var total float64
	now := time.Now()

	for _, item := range cart.Items {
		subtotal := item.Price * float64(item.Quantity)
		total += subtotal
		orderItems = append(orderItems, domain.OrderItem{
			ID:          uuid.New(),
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
			CreatedAt:   now,
		})
	}

	order := &domain.Order{
		ID:              uuid.New(),
		UserID:          userID,
		Status:          domain.OrderStatusPending,
		TotalAmount:     total,
		ShippingName:    req.ShippingName,
		ShippingAddress: req.ShippingAddress,
		Items:           orderItems,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	created, err := s.orderRepo.CreateOrder(order)
	if err != nil {
		return nil, err
	}

	if err := s.cartRepo.ClearCart(cart.ID); err != nil {
		slog.Warn("failed to clear cart after order creation", "cart_id", cart.ID, "error", err)
	}
	if err := s.cartCache.InvalidateCart(ctx, userID.String()); err != nil {
		slog.Warn("failed to invalidate cart cache", "user_id", userID, "error", err)
	}

	if err := s.orderCache.SetOrder(ctx, created); err != nil {
		slog.Warn("failed to cache new order", "order_id", created.ID, "error", err)
	}
	if err := s.orderCache.InvalidateOrderList(ctx, userID.String()); err != nil {
		slog.Warn("failed to invalidate order list cache", "user_id", userID, "error", err)
	}

	go func() {
		if err := s.publisher.Publish(context.Background(), domain.TopicOrderCreated, created); err != nil {
			slog.Warn("failed to publish order.created", "order_id", created.ID, "error", err)
		}
	}()

	return created, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, userID, orderID uuid.UUID) (*domain.Order, error) {
	if cached, err := s.orderCache.GetOrder(ctx, orderID.String()); err == nil && cached != nil {
		if cached.UserID != userID {
			return nil, domain.ErrForbidden
		}
		return cached, nil
	}

	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if err := s.orderCache.SetOrder(ctx, order); err != nil {
		slog.Warn("failed to cache order", "order_id", orderID, "error", err)
	}
	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, userID, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if !order.Status.Cancellable() {
		return nil, domain.ErrOrderNotCancellable
	}

	cancelled, err := s.orderRepo.UpdateOrderStatus(orderID, domain.OrderStatusCancelled)
	if err != nil {
		return nil, err
	}

	if err := s.orderCache.SetOrder(ctx, cancelled); err != nil {
		slog.Warn("failed to update cached order", "order_id", orderID, "error", err)
	}
	if err := s.orderCache.InvalidateOrderList(ctx, userID.String()); err != nil {
		slog.Warn("failed to invalidate order list cache", "user_id", userID, "error", err)
	}

	go func() {
		if err := s.publisher.Publish(context.Background(), domain.TopicOrderCancelled, cancelled); err != nil {
			slog.Warn("failed to publish order.cancelled", "order_id", orderID, "error", err)
		}
	}()

	return cancelled, nil
}

// buildOrderListCacheKey produces a deterministic cache key for a paginated order list.
func buildOrderListCacheKey(userID string, page, limit int) string {
	return fmt.Sprintf("orders:user:%s:page:%d:limit:%d", userID, page, limit)
}
