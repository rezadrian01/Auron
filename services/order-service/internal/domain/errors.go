package domain

import "errors"

var (
	// cart errors
	ErrCartNotFound     = errors.New("cart not found")
	ErrCartItemNotFound = errors.New("cart item not found")
	ErrCartEmpty        = errors.New("cart is empty")

	// order errors
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderNotCancellable = errors.New("order cannot be cancelled at its current status")

	// product errors
	ErrProductNotFound = errors.New("product not found")
	ErrProductInactive = errors.New("product is no longer available")

	// validation errors
	ErrInvalidQuantity = errors.New("quantity must be at least 1")
	ErrInvalidPageParam  = errors.New("page must be >= 1")
	ErrInvalidLimitParam = errors.New("limit must be >= 1 and <= 100")

	// generic
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
