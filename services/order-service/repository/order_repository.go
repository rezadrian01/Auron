package repository

import (
	"github.com/auron/order-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct{ db *gorm.DB }

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(o *models.Order) error {
	return r.db.Create(o).Error
}

func (r *OrderRepository) FindByID(id uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.First(&order, "id = ?", id).Error
	return &order, err
}

func (r *OrderRepository) FindByUserID(userID uuid.UUID) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.Find(&orders, "user_id = ?", userID).Error
	return orders, err
}

func (r *OrderRepository) Update(o *models.Order) error {
	return r.db.Save(o).Error
}

type CartRepository struct{}

func NewCartRepository(redisClient interface{}) *CartRepository {
	return &CartRepository{}
}
