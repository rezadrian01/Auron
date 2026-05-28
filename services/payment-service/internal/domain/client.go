package domain

import "context"

type StripeClient interface {
	CreatePaymentIntent(ctx context.Context, amount float64, currency string, metadata map[string]string) (intentID, clientSecret string, err error)
}
