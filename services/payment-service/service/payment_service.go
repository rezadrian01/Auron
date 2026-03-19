package service

import (
	"github.com/auron/payment-service/config"
	"github.com/auron/payment-service/models"
	"github.com/auron/payment-service/repository"
	"github.com/google/uuid"
)

type PaymentService struct {
	repo *repository.PaymentRepository
	cfg  *config.Config
}

func NewPaymentService(repo *repository.PaymentRepository, cfg *config.Config) *PaymentService {
	return &PaymentService{repo: repo, cfg: cfg}
}

func (s *PaymentService) GetPayment(id uuid.UUID) (*models.Payment, error) {
	return s.repo.FindByID(id)
}

func (s *PaymentService) CreatePayment(orderID, userID uuid.UUID, amount float64) (*models.Payment, error) {
	payment := &models.Payment{
		OrderID: orderID,
		UserID:  userID,
		Amount:  amount,
		Status:  "PENDING",
	}
	err := s.repo.Create(payment)
	return payment, err
}

func (s *PaymentService) ProcessPayment(paymentID uuid.UUID, stripeIntentID string) error {
	payment, err := s.repo.FindByID(paymentID)
	if err != nil {
		return err
	}
	payment.Status = "COMPLETED"
	payment.StripePaymentIntentID = stripeIntentID
	return s.repo.Update(payment)
}
