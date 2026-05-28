package repository

import (
	"auron/payment-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) domain.PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) GetPaymentByID(id uuid.UUID) (*domain.Payment, error) {
	var payment domain.Payment
	if err := r.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) GetPaymentByOrderID(orderID uuid.UUID) (*domain.Payment, error) {
	var payment domain.Payment
	if err := r.db.First(&payment, "order_id = ?", orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) CreatePayment(payment *domain.Payment) (*domain.Payment, error) {
	if err := r.db.Create(payment).Error; err != nil {
		return nil, err
	}
	return payment, nil
}

func (r *PaymentRepository) UpdatePaymentStatus(id uuid.UUID, status domain.PaymentStatus, failureReason string) (*domain.Payment, error) {
	updates := map[string]any{
		"status": status,
	}
	if failureReason != "" {
		updates["failure_reason"] = failureReason
	}
	if err := r.db.Model(&domain.Payment{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetPaymentByID(id)
}

func (r *PaymentRepository) UpdateStripeIDs(id uuid.UUID, intentID, clientSecret string) (*domain.Payment, error) {
	updates := map[string]any{
		"stripe_payment_intent_id": intentID,
		"stripe_client_secret":     clientSecret,
	}
	if err := r.db.Model(&domain.Payment{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetPaymentByID(id)
}
