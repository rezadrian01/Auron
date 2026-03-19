package repository

import (
	"github.com/auron/payment-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(p *models.Payment) error {
	return r.db.Create(p).Error
}

func (r *PaymentRepository) FindByID(id uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.First(&payment, "id = ?", id).Error
	return &payment, err
}

func (r *PaymentRepository) FindByOrderID(orderID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.First(&payment, "order_id = ?", orderID).Error
	return &payment, err
}

func (r *PaymentRepository) Update(p *models.Payment) error {
	return r.db.Save(p).Error
}
