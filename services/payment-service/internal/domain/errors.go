package domain

import "errors"

var (
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrPaymentAlreadyExists    = errors.New("payment already exists for this order")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
)
