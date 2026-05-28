package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"auron/notification-service/internal/domain"
	"auron/notification-service/internal/email"
)

type notificationService struct {
	sender email.EmailSender
}

func NewNotificationService(sender email.EmailSender) domain.NotificationService {
	return &notificationService{sender: sender}
}

func (s *notificationService) HandleUserCreated(_ context.Context, event domain.UserCreatedEvent) error {
	subject := fmt.Sprintf("Welcome to Auron, %s!", event.Name)
	body := fmt.Sprintf(
		"Hi %s,\n\nYour account has been created successfully.\nStart shopping at Auron!\n\nWelcome aboard,\nThe Auron Team",
		event.Name,
	)
	return s.sender.Send(event.Email, subject, body)
}

func (s *notificationService) HandleOrderCreated(_ context.Context, event domain.OrderEvent) error {
	// order.created payload is the raw Order struct — no user email included.
	// Log and skip; a follow-up can add a user-service lookup to resolve email.
	slog.Info("order.created received — no user email in event, skipping notification",
		"order_id", event.ID,
		"user_id", event.UserID,
		"total_amount", event.TotalAmount,
	)
	return nil
}

func (s *notificationService) HandleOrderCancelled(_ context.Context, event domain.OrderEvent) error {
	// Same constraint as HandleOrderCreated — no user email in the event payload.
	slog.Info("order.cancelled received — no user email in event, skipping notification",
		"order_id", event.ID,
		"user_id", event.UserID,
	)
	return nil
}

func (s *notificationService) HandlePaymentCompleted(_ context.Context, event domain.PaymentEvent) error {
	// payment.completed carries user_id (UUID) but not the user's email.
	// Log for now; a follow-up can resolve email via user-service HTTP call.
	slog.Info("payment.completed received — no user email in event, skipping notification",
		"payment_id", event.ID,
		"order_id", event.OrderID,
		"user_id", event.UserID,
		"amount", fmt.Sprintf("%.2f %s", event.Amount, strings.ToUpper(event.Currency)),
	)
	return nil
}

func (s *notificationService) HandlePaymentFailed(_ context.Context, event domain.PaymentEvent) error {
	slog.Info("payment.failed received — no user email in event, skipping notification",
		"payment_id", event.ID,
		"order_id", event.OrderID,
		"user_id", event.UserID,
		"reason", event.FailureReason,
	)
	return nil
}
