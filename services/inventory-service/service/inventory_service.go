package service

import (
	"github.com/auron/inventory-service/config"
	"github.com/auron/inventory-service/models"
	"github.com/auron/inventory-service/repository"
	"github.com/google/uuid"
)

type InventoryService struct {
	repo *repository.InventoryRepository
	cfg  *config.Config
}

func NewInventoryService(repo *repository.InventoryRepository, cfg *config.Config) *InventoryService {
	return &InventoryService{repo: repo, cfg: cfg}
}

func (s *InventoryService) GetInventory(productID uuid.UUID) (*models.Inventory, error) {
	return s.repo.FindByProductID(productID)
}

func (s *InventoryService) ReserveStock(productID uuid.UUID, quantity int) (bool, error) {
	return s.repo.ReserveStock(productID, quantity)
}

func (s *InventoryService) UpdateStock(productID uuid.UUID, totalQuantity int) error {
	inv, err := s.repo.FindByProductID(productID)
	if err != nil {
		return err
	}
	inv.TotalQuantity = totalQuantity
	return s.repo.Update(inv)
}

func (s *InventoryService) StartConsumer() {
	// Placeholder for Kafka consumer
}

func (s *InventoryService) Stop() {
	// Placeholder for stopping consumer
}
