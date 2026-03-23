package domain

import "errors"

// Sentinel errors - use errors.Is() to compare

var (
	ErrPasswordMismatch   = errors.New("password and confirm password do not match")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
)
