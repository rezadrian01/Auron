package service

import (
	"context"
	"time"

	"github.com/auron/user-service/config"
	"github.com/auron/user-service/models"
	"github.com/auron/user-service/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// UserService handles user business logic
type UserService struct {
	repo       *repository.UserRepository
	addressRepo *repository.AddressRepository
	redis      *redis.Client
	cfg        *config.Config
}

// NewUserService creates a new user service
func NewUserService(repo *repository.UserRepository, addressRepo *repository.AddressRepository, redisClient *redis.Client, cfg *config.Config) *UserService {
	return &UserService{
		repo:       repo,
		addressRepo: addressRepo,
		redis:      redisClient,
		cfg:        cfg,
	}
}

// Register creates a new user
func (s *UserService) Register(email, password, name string) (*models.User, error) {
	user := &models.User{
		Email:    email,
		Password: password, // Should be hashed
		Name:     name,
		Role:     "customer",
	}

	err := s.repo.Create(user)
	return user, err
}

// Login authenticates a user
func (s *UserService) Login(email, password string) (string, string, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return "", "", err
	}

	// TODO: Verify password

	// Generate tokens
	accessToken := "placeholder-access-token"
	refreshToken := uuid.New().String()

	// Store refresh token in Redis
	s.redis.Set(context.Background(), "refresh:"+refreshToken, user.ID.String(), 7*24*time.Hour)

	return accessToken, refreshToken, nil
}

// GetProfile gets user profile
func (s *UserService) GetProfile(userID uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(userID)
}

// GetAddresses gets user addresses
func (s *UserService) GetAddresses(userID uuid.UUID) ([]models.Address, error) {
	return s.addressRepo.FindByUserID(userID)
}

// AddAddress adds a new address
func (s *UserService) AddAddress(userID uuid.UUID, address *models.Address) error {
	address.UserID = userID
	return s.addressRepo.Create(address)
}

// DeleteAddress deletes an address
func (s *UserService) DeleteAddress(addressID uuid.UUID) error {
	return s.addressRepo.Delete(addressID)
}
