package client

import (
	"context"
	"fmt"

	"auron/payment-service/internal/domain"

	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
)

type stripeClient struct {
	secretKey string
}

func NewStripeClient(secretKey string) domain.StripeClient {
	stripe.Key = secretKey
	return &stripeClient{secretKey: secretKey}
}

func (c *stripeClient) CreatePaymentIntent(_ context.Context, amount float64, currency string, metadata map[string]string) (string, string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount * 100)),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: metadata,
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", "", fmt.Errorf("stripe: create payment intent: %w", err)
	}

	return pi.ID, pi.ClientSecret, nil
}
