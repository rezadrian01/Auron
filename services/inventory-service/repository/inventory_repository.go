package repository

import (
	"github.com/auron/inventory-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryRepository struct{ db *gorm.DB }

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) FindByProductID(productID uuid.UUID) (*models.Inventory, error) {
	var inv models.Inventory
	err := r.db.First(&inv, "product_id = ?", productID).Error
	return &inv, err
}

func (r *InventoryRepository) Update(inv *models.Inventory) error {
	return r.db.Save(inv).Error
}

func (r *InventoryRepository) ReserveStock(productID uuid.UUID, quantity int) (bool, error) {
	result := r.db.Model(&models.Inventory{}).
		Where("product_id = ? AND version = ? AND (total_quantity - reserved_quantity) >= ?",
			productID, 0, quantity).
		Updates(map[string]interface{}{
			"reserved_quantity": gorm.Expr("reserved_quantity + ?", quantity),
			"version":          gorm.Expr("version + 1"),
		})
	return result.RowsAffected > 0, result.Error
}
