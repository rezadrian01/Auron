package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"auron/order-service/internal/domain"

	"github.com/google/uuid"
)

type CartService struct {
	cartRepo      domain.CartRepository
	cartCache     domain.CartCache
	productClient domain.ProductClient
}

func NewCartService(
	cartRepo domain.CartRepository,
	cartCache domain.CartCache,
	productClient domain.ProductClient,
) domain.CartService {
	return &CartService{
		cartRepo:      cartRepo,
		cartCache:     cartCache,
		productClient: productClient,
	}
}

func (s *CartService) GetCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	if cached, err := s.cartCache.GetCart(ctx, userID.String()); err == nil && cached != nil {
		return cached, nil
	}

	cart, err := s.cartRepo.GetCartByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) {
			return s.createEmptyCart(ctx, userID)
		}
		return nil, err
	}

	if err := s.cartCache.SetCart(ctx, cart); err != nil {
		slog.Warn("failed to cache cart", "user_id", userID, "error", err)
	}
	return cart, nil
}

func (s *CartService) AddItem(ctx context.Context, userID uuid.UUID, req domain.AddItemRequest) (*domain.Cart, error) {
	snapshot, err := s.productClient.GetProduct(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}

	cart, err := s.cartRepo.GetCartByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) {
			cart, err = s.createEmptyCart(ctx, userID)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// if product already in cart, merge quantities and refresh price
	for _, item := range cart.Items {
		if item.ProductID == req.ProductID {
			item.Quantity += req.Quantity
			item.Price = snapshot.Price
			item.UpdatedAt = time.Now()
			if _, err := s.cartRepo.UpdateCartItem(&item); err != nil {
				return nil, err
			}
			return s.reloadCart(ctx, userID)
		}
	}

	now := time.Now()
	newItem := &domain.CartItem{
		ID:          uuid.New(),
		CartID:      cart.ID,
		ProductID:   snapshot.ID,
		ProductName: snapshot.Name,
		Price:       snapshot.Price,
		Quantity:    req.Quantity,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := s.cartRepo.CreateCartItem(newItem); err != nil {
		return nil, err
	}

	return s.reloadCart(ctx, userID)
}

func (s *CartService) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetCartByUserID(userID)
	if err != nil {
		return nil, err
	}

	item, err := s.cartRepo.GetCartItemByID(itemID)
	if err != nil {
		return nil, err
	}

	if item.CartID != cart.ID {
		return nil, domain.ErrForbidden
	}

	item.Quantity = quantity
	item.UpdatedAt = time.Now()
	if _, err := s.cartRepo.UpdateCartItem(item); err != nil {
		return nil, err
	}

	return s.reloadCart(ctx, userID)
}

func (s *CartService) RemoveItem(ctx context.Context, userID, itemID uuid.UUID) error {
	cart, err := s.cartRepo.GetCartByUserID(userID)
	if err != nil {
		return err
	}

	item, err := s.cartRepo.GetCartItemByID(itemID)
	if err != nil {
		return err
	}

	if item.CartID != cart.ID {
		return domain.ErrForbidden
	}

	if err := s.cartRepo.DeleteCartItem(itemID); err != nil {
		return err
	}

	if err := s.cartCache.InvalidateCart(ctx, userID.String()); err != nil {
		slog.Warn("failed to invalidate cart cache", "user_id", userID, "error", err)
	}
	return nil
}

// createEmptyCart inserts a new cart for the user and returns it.
func (s *CartService) createEmptyCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	cart := &domain.Cart{
		ID:     uuid.New(),
		UserID: userID,
		Items:  []domain.CartItem{},
	}
	created, err := s.cartRepo.CreateCart(cart)
	if err != nil {
		return nil, err
	}
	if err := s.cartCache.SetCart(ctx, created); err != nil {
		slog.Warn("failed to cache new cart", "user_id", userID, "error", err)
	}
	return created, nil
}

// reloadCart re-fetches the cart from DB and updates the cache.
func (s *CartService) reloadCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetCartByUserID(userID)
	if err != nil {
		return nil, err
	}
	if err := s.cartCache.SetCart(ctx, cart); err != nil {
		slog.Warn("failed to cache cart", "user_id", userID, "error", err)
	}
	return cart, nil
}
