package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Name      string    `json:"name" gorm:"not null"`
	Role      string    `json:"role" gorm:"default:customer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

// Address represents a user address
type Address struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Label       string    `json:"label"`
	Street      string    `json:"street" gorm:"not null"`
	City        string    `json:"city" gorm:"not null"`
	State       string    `json:"state"`
	Country     string    `json:"country" gorm:"not null"`
	PostalCode string    `json:"postal_code"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName specifies the table name
func (Address) TableName() string {
	return "addresses"
}
