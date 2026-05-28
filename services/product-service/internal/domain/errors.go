package domain

import "errors"

var (
	// product errors
	ErrProductNotFound      = errors.New("product not found")
	ErrInvalidProductID     = errors.New("invalid product ID")
	ErrProductAlreadyExists = errors.New("product with this name already exists")

	// category errors
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCategorySlugExists = errors.New("category with this slug already exists")
	ErrInvalidCategoryID  = errors.New("invalid category ID")

	// inventory errors
	ErrInvalidSortParam    = errors.New("invalid sort parameter. allowed price_asc, price_desc, newest, name_asc, name_desc")
	ErrInvalidPageParam    = errors.New("page must be >= 1")
	ErrInvalidLimitParam   = errors.New("limit must be >= 1 and <= 100")
	ErrPriceMustBePositive = errors.New("price must be a positive number")

	//generic
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)