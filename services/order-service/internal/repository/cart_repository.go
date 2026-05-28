package repository

import (
	"auron/order-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) domain.CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) GetCartByUserID(userID uuid.UUID) (*domain.Cart, error) {
	var cart domain.Cart
	if err := r.db.Preload("Items").First(&cart, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCartNotFound
		}
		return nil, err
	}
	return &cart, nil
}

func (r *CartRepository) CreateCart(cart *domain.Cart) (*domain.Cart, error) {
	if err := r.db.Create(cart).Error; err != nil {
		return nil, err
	}
	return cart, nil
}

func (r *CartRepository) GetCartItemByID(itemID uuid.UUID) (*domain.CartItem, error) {
	var item domain.CartItem
	if err := r.db.First(&item, "id = ?", itemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCartItemNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *CartRepository) CreateCartItem(item *domain.CartItem) (*domain.CartItem, error) {
	if err := r.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *CartRepository) UpdateCartItem(item *domain.CartItem) (*domain.CartItem, error) {
	if err := r.db.Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *CartRepository) DeleteCartItem(itemID uuid.UUID) error {
	return r.db.Delete(&domain.CartItem{}, "id = ?", itemID).Error
}

func (r *CartRepository) ClearCart(cartID uuid.UUID) error {
	return r.db.Delete(&domain.CartItem{}, "cart_id = ?", cartID).Error
}
