package service

import (
	"github.com/auron/order-service/config"
	"github.com/auron/order-service/models"
	"github.com/auron/order-service/repository"
	"github.com/google/uuid"
)

type OrderService struct {
	repo     *repository.OrderRepository
	cartRepo *repository.CartRepository
	cfg      *config.Config
}

func NewOrderService(repo *repository.OrderRepository, cartRepo *repository.CartRepository, cfg *config.Config) *OrderService {
	return &OrderService{
		repo:     repo,
		cartRepo: cartRepo,
		cfg:      cfg,
	}
}

func (s *OrderService) CreateOrder(userID uuid.UUID, items []models.CartItem, address map[string]interface{}) (*models.Order, error) {
	order := &models.Order{
		UserID:          userID,
		Status:          "PENDING",
		TotalAmount:     0,
		ShippingAddress: address,
	}
	// Calculate total, add items, etc.
	return order, s.repo.Create(order)
}

func (s *OrderService) GetOrder(id uuid.UUID) (*models.Order, error) {
	return s.repo.FindByID(id)
}

func (s *OrderService) ListOrders(userID uuid.UUID) ([]models.Order, error) {
	return s.repo.FindByUserID(userID)
}

func (s *OrderService) CancelOrder(id uuid.UUID) error {
	order, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	order.Status = "CANCELLED"
	return s.repo.Update(order)
}

func (s *OrderService) GetCart(userID uuid.UUID) ([]models.CartItem, error) {
	return []models.CartItem{}, nil
}
