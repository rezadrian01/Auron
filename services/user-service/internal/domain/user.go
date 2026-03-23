package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleCustomer = "customer"
	RoleSeller   = "seller"
	RoleAdmin    = "admin"
)

type User struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email         string         `json:"email" gorm:"type:varchar(255);not null;uniqueIndex"`
	Password      string         `json:"-" gorm:"type:varchar(255);not null"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null"`
	Role          string         `json:"role" gorm:"type:varchar(50);not null;default:'customer'"`
	IsActive      bool           `json:"is_active" gorm:"not null;default:true"`
	CreatedAt     time.Time      `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"not null;default:now()"`
	Addresses     []Address      `json:"addresses,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	RefreshTokens []RefreshToken `json:"refresh_tokens,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (User) TableName() string {
	return "users"
}

type Address struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Label      *string   `json:"label,omitempty" gorm:"type:varchar(100)"`
	Street     string    `json:"street" gorm:"type:text;not null"`
	City       string    `json:"city" gorm:"type:varchar(100);not null"`
	State      *string   `json:"state,omitempty" gorm:"type:varchar(100)"`
	Country    string    `json:"country" gorm:"type:varchar(100);not null"`
	PostalCode *string   `json:"postal_code,omitempty" gorm:"type:varchar(20)"`
	IsDefault  bool      `json:"is_default" gorm:"not null;default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:now()"`
	User       User      `json:"-" gorm:"foreignKey:UserID;references:ID"`
}

func (Address) TableName() string {
	return "addresses"
}

type RefreshToken struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash string     `json:"token_hash" gorm:"type:varchar(512);not null;uniqueIndex"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null;default:now()"`
	User      User       `json:"-" gorm:"foreignKey:UserID;references:ID"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// -------------------------------------------------------
// DTOs
// -------------------------------------------------------
type CookieConfig struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	MaxAge   int    `json:"max_age"`
	HttpOnly bool   `json:"http_only"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"same_site"`
}

type AuthResponse struct {
	User               *User        `json:"user"`
	AccessToken        string       `json:"access_token"`
	RefreshToken       string       `json:"refresh_token"`
	AccessTokenCookie  CookieConfig `json:"access_token_cookie"`
	RefreshTokenCookie CookieConfig `json:"refresh_token_cookie"`
}

type CreateUserRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
	Name            string `json:"name" binding:"required"`
	Role            string `json:"role" binding:"omitempty,oneof=customer seller admin"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email" binding:"omitempty,email"`
	Name     *string `json:"name" binding:"omitempty"`
	Password *string `json:"password" binding:"omitempty,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type UserEnvelopeResponse struct {
	User UserResponse `json:"user"`
}

type AddressEnvelopeResponse struct {
	Address AddressResponse `json:"address"`
}

type AddressesEnvelopeResponse struct {
	Addresses []AddressResponse `json:"addresses"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RevokeTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

type CreateAddressRequest struct {
	Label      *string `json:"label,omitempty"`
	Street     string  `json:"street" binding:"required"`
	City       string  `json:"city" binding:"required"`
	State      *string `json:"state,omitempty"`
	Country    string  `json:"country" binding:"required"`
	PostalCode *string `json:"postal_code,omitempty"`
	IsDefault  bool    `json:"is_default"`
}

type UpdateAddressRequest struct {
	Label      *string `json:"label,omitempty"`
	Street     *string `json:"street,omitempty"`
	City       *string `json:"city,omitempty"`
	State      *string `json:"state,omitempty"`
	Country    *string `json:"country,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`
	IsDefault  *bool   `json:"is_default,omitempty"`
}

type DeleteAddressRequest struct {
	AddressID uuid.UUID `json:"address_id" binding:"required"`
}

type DeleteAddressResponse struct {
	Message string `json:"message"`
}

type AddressResponse struct {
	ID         uuid.UUID `json:"id"`
	Label      *string   `json:"label,omitempty"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	State      *string   `json:"state,omitempty"`
	Country    string    `json:"country"`
	PostalCode *string   `json:"postal_code,omitempty"`
	IsDefault  bool      `json:"is_default"`
}

// -------------------------------------------------------
// Repository interface
// -------------------------------------------------------
type UserRepository interface {
	CreateUser(user *User) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id uuid.UUID) (*User, error)
	UpdateUser(user *User) (*User, error)
	DeleteUser(id uuid.UUID) error
	GetAddressesByUserID(userID uuid.UUID) ([]Address, error)
	AddAddress(address *Address) (*Address, error)
	UpdateAddress(address *Address) (*Address, error)
	DeleteAddress(addressID uuid.UUID) error
	AddRefreshToken(token *RefreshToken) (*RefreshToken, error)
	GetRefreshToken(tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(tokenHash string) error
}

// -------------------------------------------------------
// Service interface
// -------------------------------------------------------
type UserService interface {
	RegisterUser(req *CreateUserRequest) (*User, error)
	Login(req *LoginRequest) (*User, string, error)
	RefreshToken(req *RefreshTokenRequest) (string, error)
	RevokeToken(req *RevokeTokenRequest) error
	GetUserProfile(userID uuid.UUID) (*User, error)
	UpdateUserProfile(userID uuid.UUID, req *UpdateUserRequest) (*User, error)
	AddAddress(userID uuid.UUID, address *Address) (*Address, error)
	GetAddresses(userID uuid.UUID) ([]Address, error)
	UpdateAddress(address *Address) (*Address, error)
	DeleteAddress(addressID string) error
}
