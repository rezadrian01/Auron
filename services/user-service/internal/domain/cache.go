package domain

import "context"

type UserCache interface {
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	SetUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, userID string) error
}
