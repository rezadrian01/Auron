package repository

import (
	"auron/order-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) domain.OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) GetOrdersByUserID(userID uuid.UUID, offset, limit int) ([]domain.Order, int64, error) {
	var total int64
	if err := r.db.Model(&domain.Order{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []domain.Order
	if err := r.db.Preload("Items").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *OrderRepository) GetOrderByID(orderID uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) CreateOrder(order *domain.Order) (*domain.Order, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Items").Create(order).Error; err != nil {
			return err
		}
		for i := range order.Items {
			order.Items[i].OrderID = order.ID
			if err := tx.Create(&order.Items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *OrderRepository) UpdateOrderStatus(orderID uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	if err := r.db.Model(&domain.Order{}).Where("id = ?", orderID).Update("status", status).Error; err != nil {
		return nil, err
	}
	return r.GetOrderByID(orderID)
}
