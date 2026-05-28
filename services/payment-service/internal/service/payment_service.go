package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"auron/payment-service/internal/domain"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

type PaymentService struct {
	paymentRepo   domain.PaymentRepository
	paymentCache  domain.PaymentCache
	stripeClient  domain.StripeClient
	publisher     domain.EventPublisher
	webhookSecret string
}

func NewPaymentService(
	paymentRepo domain.PaymentRepository,
	paymentCache domain.PaymentCache,
	stripeClient domain.StripeClient,
	publisher domain.EventPublisher,
	webhookSecret string,
) domain.PaymentService {
	return &PaymentService{
		paymentRepo:   paymentRepo,
		paymentCache:  paymentCache,
		stripeClient:  stripeClient,
		publisher:     publisher,
		webhookSecret: webhookSecret,
	}
}

func (s *PaymentService) GetPaymentByID(ctx context.Context, userID, paymentID uuid.UUID) (*domain.PaymentResponse, error) {
	if cached, err := s.paymentCache.GetPayment(ctx, paymentID); err == nil && cached != nil {
		if cached.UserID != userID {
			return nil, domain.ErrForbidden
		}
		return cached.ToResponse(), nil
	}

	payment, err := s.paymentRepo.GetPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	if payment.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if err := s.paymentCache.SetPayment(ctx, payment); err != nil {
		slog.Warn("failed to cache payment", "payment_id", paymentID, "error", err)
	}

	return payment.ToResponse(), nil
}

func (s *PaymentService) HandleOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error {
	// Idempotency: skip if payment already exists for this order.
	existing, err := s.paymentRepo.GetPaymentByOrderID(event.OrderID)
	if err != nil && !errors.Is(err, domain.ErrPaymentNotFound) {
		return err
	}
	if existing != nil {
		slog.Info("payment already exists for order, skipping", "order_id", event.OrderID)
		return nil
	}

	now := time.Now()
	payment := &domain.Payment{
		ID:        uuid.New(),
		OrderID:   event.OrderID,
		UserID:    event.UserID,
		Amount:    event.TotalAmount,
		Currency:  "usd",
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.paymentRepo.CreatePayment(payment)
	if err != nil {
		return err
	}

	metadata := map[string]string{
		"payment_id": created.ID.String(),
		"order_id":   event.OrderID.String(),
		"user_id":    event.UserID.String(),
	}

	intentID, clientSecret, err := s.stripeClient.CreatePaymentIntent(ctx, event.TotalAmount, "usd", metadata)
	if err != nil {
		slog.Error("failed to create stripe payment intent", "payment_id", created.ID, "error", err)
		return err
	}

	updated, err := s.paymentRepo.UpdateStripeIDs(created.ID, intentID, clientSecret)
	if err != nil {
		return err
	}

	if err := s.paymentCache.SetPayment(ctx, updated); err != nil {
		slog.Warn("failed to cache payment", "payment_id", updated.ID, "error", err)
	}

	// Publish payment.created — client_secret travels only via this event, never HTTP.
	go func() {
		payload := map[string]any{
			"payment_id":    updated.ID,
			"order_id":      updated.OrderID,
			"user_id":       updated.UserID,
			"amount":        updated.Amount,
			"currency":      updated.Currency,
			"client_secret": clientSecret,
		}
		if err := s.publisher.Publish(context.Background(), domain.TopicPaymentCreated, payload); err != nil {
			slog.Warn("failed to publish payment.created", "payment_id", updated.ID, "error", err)
		}
	}()

	return nil
}

func (s *PaymentService) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	var event stripe.Event
	var err error

	if s.webhookSecret == "" {
		// Allow unsigned webhooks in development when no secret is configured.
		if err = json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("webhook: unmarshal event: %w", err)
		}
	} else {
		event, err = webhook.ConstructEvent(payload, signature, s.webhookSecret)
		if err != nil {
			return domain.ErrInvalidWebhookSignature
		}
	}

	switch event.Type {
	case "payment_intent.succeeded":
		return s.handlePaymentSucceeded(ctx, event)
	case "payment_intent.payment_failed":
		return s.handlePaymentFailed(ctx, event)
	case "payment_intent.processing":
		return s.handlePaymentProcessing(ctx, event)
	default:
		slog.Debug("unhandled stripe event type", "type", event.Type)
	}

	return nil
}

func (s *PaymentService) handlePaymentSucceeded(ctx context.Context, event stripe.Event) error {
	pi, err := extractPaymentIntent(event)
	if err != nil {
		return err
	}

	payment, err := s.resolvePaymentFromIntent(pi)
	if err != nil {
		return err
	}

	updated, err := s.paymentRepo.UpdatePaymentStatus(payment.ID, domain.PaymentStatusCompleted, "")
	if err != nil {
		return err
	}

	if err := s.paymentCache.SetPayment(ctx, updated); err != nil {
		slog.Warn("failed to cache payment after succeeded", "payment_id", updated.ID, "error", err)
	}

	go func() {
		if err := s.publisher.Publish(context.Background(), domain.TopicPaymentCompleted, updated.ToResponse()); err != nil {
			slog.Warn("failed to publish payment.completed", "payment_id", updated.ID, "error", err)
		}
	}()

	return nil
}

func (s *PaymentService) handlePaymentFailed(ctx context.Context, event stripe.Event) error {
	pi, err := extractPaymentIntent(event)
	if err != nil {
		return err
	}

	reason := ""
	if pi.LastPaymentError != nil {
		reason = pi.LastPaymentError.Msg
	}

	payment, err := s.resolvePaymentFromIntent(pi)
	if err != nil {
		return err
	}

	updated, err := s.paymentRepo.UpdatePaymentStatus(payment.ID, domain.PaymentStatusFailed, reason)
	if err != nil {
		return err
	}

	if err := s.paymentCache.SetPayment(ctx, updated); err != nil {
		slog.Warn("failed to cache payment after failed", "payment_id", updated.ID, "error", err)
	}

	go func() {
		if err := s.publisher.Publish(context.Background(), domain.TopicPaymentFailed, updated.ToResponse()); err != nil {
			slog.Warn("failed to publish payment.failed", "payment_id", updated.ID, "error", err)
		}
	}()

	return nil
}

func (s *PaymentService) handlePaymentProcessing(ctx context.Context, event stripe.Event) error {
	pi, err := extractPaymentIntent(event)
	if err != nil {
		return err
	}

	payment, err := s.resolvePaymentFromIntent(pi)
	if err != nil {
		return err
	}

	updated, err := s.paymentRepo.UpdatePaymentStatus(payment.ID, domain.PaymentStatusProcessing, "")
	if err != nil {
		return err
	}

	if err := s.paymentCache.SetPayment(ctx, updated); err != nil {
		slog.Warn("failed to cache payment after processing", "payment_id", updated.ID, "error", err)
	}

	return nil
}

// resolvePaymentFromIntent reads payment_id from the Stripe metadata to look up the payment.
func (s *PaymentService) resolvePaymentFromIntent(pi stripe.PaymentIntent) (*domain.Payment, error) {
	paymentIDStr, ok := pi.Metadata["payment_id"]
	if !ok || paymentIDStr == "" {
		return nil, fmt.Errorf("webhook: payment_id missing from stripe metadata for intent %s", pi.ID)
	}
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		return nil, fmt.Errorf("webhook: invalid payment_id in stripe metadata: %w", err)
	}
	return s.paymentRepo.GetPaymentByID(paymentID)
}

func extractPaymentIntent(event stripe.Event) (stripe.PaymentIntent, error) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return pi, fmt.Errorf("webhook: unmarshal payment intent: %w", err)
	}
	return pi, nil
}
