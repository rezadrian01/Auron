package repository

import (
	// "auron/product-service/internal/domain"

	"auron/product-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) GetProducts(filter domain.ProductFilter) (*domain.ProductListResponse, error) {
	query := r.db.Model(&domain.Product{}).Where("is_active = ?", true)

	// filter by category
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}

	// price range filters
	if filter.MinPrice != nil {
		query = query.Where("price >= ?", *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		query = query.Where("price <= ?", *filter.MaxPrice)
	}

	// full text search
	if filter.Q != "" {
		query = query.Where("search_vector @@ plainto_tsquery('english', ?)", &filter.Q)
	}

	// apply sorting
	query = r.applySort(query, filter.Sort)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// apply pagination
	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	// execite query with category preload
	var products []domain.Product
	if err := query.Preload("Category").Find(&products).Error; err != nil {
		return nil, err
	}

	return &domain.ProductListResponse{
		Products: products,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
	}, nil
}

func (r *ProductRepository) GetProductByID(id uuid.UUID) (*domain.Product, error) {
	var product domain.Product
	if err := r.db.Preload("Category").First(&product, "id = ? AND is_active = ?", id, true).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) CreateProduct(product *domain.Product) (*domain.Product, error) {
	if err := r.db.Create(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func (r *ProductRepository) UpdateProduct(product *domain.Product) (*domain.Product, error) {
	if err := r.db.Save(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func (r *ProductRepository) DeleteProduct(id uuid.UUID) error {
	if err := r.db.Where("id = ?", id).Delete(&domain.Product{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *ProductRepository) GetCategories() ([]domain.Category, error) {
	var categories []domain.Category
	if err := r.db.Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *ProductRepository) GetCategoryByID(id uuid.UUID) (*domain.Category, error) {
	var category domain.Category
	if err := r.db.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

func (r *ProductRepository) GetCategoryBySlug(slug string) (*domain.Category, error) {
	var category domain.Category
	if err := r.db.First(&category, "slug = ?", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

func (r *ProductRepository) CreateCategory(category *domain.Category) (*domain.Category, error) {
	if err := r.db.Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (r *ProductRepository) applySort(query *gorm.DB, sort string) *gorm.DB {
	switch sort {
	case "price_asc":
		return query.Order("price ASC")

	case "price_desc":
		return query.Order("price DESC")

	case "newest":
		return query.Order("created_at DESC")

	case "name_asc":
		return query.Order("name ASC")

	case "name_desc":
		return query.Order("name DESC")

	default:
		return query.Order("created_at DESC") // default newest first
	}

}
