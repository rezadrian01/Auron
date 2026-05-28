package repository

import (
	"time"

	"auron/inventory-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) domain.InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) GetByProductID(productID uuid.UUID) (*domain.Inventory, error) {
	var inv domain.Inventory
	if err := r.db.First(&inv, "product_id = ?", productID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrInventoryNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InventoryRepository) SetTotalQuantity(productID uuid.UUID, quantity int) (*domain.Inventory, error) {
	now := time.Now()
	inv := &domain.Inventory{
		ProductID:     productID,
		TotalQuantity: quantity,
		UpdatedAt:     now,
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total_quantity": quantity,
			"version":        gorm.Expr("inventory.version + 1"),
			"updated_at":     now,
		}),
	}).Create(inv).Error; err != nil {
		return nil, err
	}
	return r.GetByProductID(productID)
}

func (r *InventoryRepository) ReserveStock(productID uuid.UUID, quantity int) (*domain.Inventory, error) {
	result := r.db.Model(&domain.Inventory{}).
		Where("product_id = ? AND (total_quantity - reserved_quantity) >= ?", productID, quantity).
		Updates(map[string]any{
			"reserved_quantity": gorm.Expr("reserved_quantity + ?", quantity),
			"version":           gorm.Expr("version + 1"),
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		// Distinguish between not found and insufficient stock
		if _, err := r.GetByProductID(productID); err != nil {
			return nil, err
		}
		return nil, domain.ErrInsufficientStock
	}
	return r.GetByProductID(productID)
}

func (r *InventoryRepository) ReleaseStock(productID uuid.UUID, quantity int) (*domain.Inventory, error) {
	result := r.db.Model(&domain.Inventory{}).
		Where("product_id = ?", productID).
		Updates(map[string]any{
			"reserved_quantity": gorm.Expr("GREATEST(reserved_quantity - ?, 0)", quantity),
			"version":           gorm.Expr("version + 1"),
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrInventoryNotFound
	}
	return r.GetByProductID(productID)
}
