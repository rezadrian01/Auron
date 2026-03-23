package repository

import (
	"auron/user-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *domain.User) (*domain.User, error) {
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateUser(user *domain.User) (*domain.User, error) {
	if err := r.db.Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) DeleteUser(id uuid.UUID) error {
	if err := r.db.Where("id = ?", id).Delete(&domain.User{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetAddressesByUserID(userID uuid.UUID) ([]domain.Address, error) {
	var addresses []domain.Address
	if err := r.db.Where("user_id = ?", userID).Find(&addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

func (r *UserRepository) AddAddress(address *domain.Address) (*domain.Address, error) {
	if err := r.db.Create(address).Error; err != nil {
		return nil, err
	}
	return address, nil
}

func (r *UserRepository) UpdateAddress(userID, addressID uuid.UUID, req *domain.UpdateAddressRequest) (*domain.Address, error) {
	var address domain.Address
	if err := r.db.Where("id = ? AND user_id = ?", addressID, userID).First(&address).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrAddressNotFound
		}
		return nil, err
	}

	updates := map[string]any{}
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.Street != nil {
		updates["street"] = *req.Street
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.State != nil {
		updates["state"] = *req.State
	}
	if req.Country != nil {
		updates["country"] = *req.Country
	}
	if req.PostalCode != nil {
		updates["postal_code"] = *req.PostalCode
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}

	if len(updates) == 0 {
		return &address, nil
	}

	if err := r.db.Model(&address).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := r.db.Where("id = ? AND user_id = ?", addressID, userID).First(&address).Error; err != nil {
		return nil, err
	}

	return &address, nil
}

func (r *UserRepository) DeleteAddress(userID, addressID uuid.UUID) error {
	result := r.db.Where("id = ? AND user_id = ?", addressID, userID).Delete(&domain.Address{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrAddressNotFound
	}

	return nil
}

func (r *UserRepository) AddRefreshToken(token *domain.RefreshToken) (*domain.RefreshToken, error) {
	if err := r.db.Create(token).Error; err != nil {
		return nil, err
	}
	return token, nil
}

func (r *UserRepository) GetRefreshToken(tokenHash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	if err := r.db.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}
	return &token, nil
}

func (r *UserRepository) RevokeRefreshToken(tokenHash string) error {
	if err := r.db.Where("token_hash = ?", tokenHash).Delete(&domain.RefreshToken{}).Error; err != nil {
		return err
	}
	return nil
}
