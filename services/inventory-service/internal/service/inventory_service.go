package service

import (
	"context"
	"log/slog"

	"auron/inventory-service/internal/domain"

	"github.com/google/uuid"
)

type InventoryService struct {
	repo      domain.InventoryRepository
	cache     domain.InventoryCache
	publisher domain.EventPublisher
}

func NewInventoryService(
	repo domain.InventoryRepository,
	cache domain.InventoryCache,
	publisher domain.EventPublisher,
) domain.InventoryService {
	return &InventoryService{
		repo:      repo,
		cache:     cache,
		publisher: publisher,
	}
}

func (s *InventoryService) GetInventory(ctx context.Context, productID uuid.UUID) (*domain.InventoryResponse, error) {
	if cached, err := s.cache.GetInventory(ctx, productID); err == nil && cached != nil {
		return cached.ToResponse(), nil
	}

	inv, err := s.repo.GetByProductID(productID)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetInventory(ctx, inv); err != nil {
		slog.Warn("failed to cache inventory", "product_id", productID, "error", err)
	}

	return inv.ToResponse(), nil
}

func (s *InventoryService) SetInventory(ctx context.Context, productID uuid.UUID, req domain.UpdateInventoryRequest) (*domain.InventoryResponse, error) {
	inv, err := s.repo.SetTotalQuantity(productID, req.TotalQuantity)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetInventory(ctx, inv); err != nil {
		slog.Warn("failed to cache inventory after set", "product_id", productID, "error", err)
	}

	go s.publishUpdated(inv)
	if inv.AvailableQuantity() <= domain.LowStockThreshold {
		go s.publishLowStock(inv)
	}

	return inv.ToResponse(), nil
}

func (s *InventoryService) HandleOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error {
	for _, item := range event.Items {
		inv, err := s.repo.ReserveStock(item.ProductID, item.Quantity)
		if err != nil {
			slog.Error("failed to reserve stock",
				"order_id", event.OrderID,
				"product_id", item.ProductID,
				"quantity", item.Quantity,
				"error", err)
			continue
		}

		if err := s.cache.InvalidateInventory(ctx, item.ProductID); err != nil {
			slog.Warn("failed to invalidate inventory cache", "product_id", item.ProductID, "error", err)
		}

		go s.publishUpdated(inv)
		if inv.AvailableQuantity() <= domain.LowStockThreshold {
			go s.publishLowStock(inv)
		}
	}
	return nil
}

func (s *InventoryService) HandleOrderCancelled(ctx context.Context, event domain.OrderCreatedEvent) error {
	for _, item := range event.Items {
		inv, err := s.repo.ReleaseStock(item.ProductID, item.Quantity)
		if err != nil {
			slog.Error("failed to release stock",
				"order_id", event.OrderID,
				"product_id", item.ProductID,
				"quantity", item.Quantity,
				"error", err)
			continue
		}

		if err := s.cache.InvalidateInventory(ctx, item.ProductID); err != nil {
			slog.Warn("failed to invalidate inventory cache", "product_id", item.ProductID, "error", err)
		}

		go s.publishUpdated(inv)
	}
	return nil
}

func (s *InventoryService) publishUpdated(inv *domain.Inventory) {
	if err := s.publisher.Publish(context.Background(), domain.TopicInventoryUpdated, inv.ToResponse()); err != nil {
		slog.Warn("failed to publish inventory.updated", "product_id", inv.ProductID, "error", err)
	}
}

func (s *InventoryService) publishLowStock(inv *domain.Inventory) {
	if err := s.publisher.Publish(context.Background(), domain.TopicInventoryLowStock, inv.ToResponse()); err != nil {
		slog.Warn("failed to publish inventory.low_stock", "product_id", inv.ProductID, "error", err)
	}
}
