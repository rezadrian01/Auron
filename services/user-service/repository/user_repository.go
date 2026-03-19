package repository

import (
	"github.com/auron/user-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete deletes a user
func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

// AddressRepository handles address database operations
type AddressRepository struct {
	db *gorm.DB
}

// NewAddressRepository creates a new address repository
func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

// FindByUserID finds all addresses for a user
func (r *AddressRepository) FindByUserID(userID uuid.UUID) ([]models.Address, error) {
	var addresses []models.Address
	err := r.db.Find(&addresses, "user_id = ?", userID).Error
	return addresses, err
}

// Create creates a new address
func (r *AddressRepository) Create(address *models.Address) error {
	return r.db.Create(address).Error
}

// Delete deletes an address
func (r *AddressRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Address{}, "id = ?", id).Error
}
