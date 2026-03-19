package service

import (
	"github.com/auron/product-service/config"
	"github.com/auron/product-service/models"
	"github.com/auron/product-service/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ProductService struct {
	repo        *repository.ProductRepository
	categoryRepo *repository.CategoryRepository
	redis       *redis.Client
	cfg         *config.Config
}

func NewProductService(repo *repository.ProductRepository, categoryRepo *repository.CategoryRepository, redisClient *redis.Client, cfg *config.Config) *ProductService {
	return &ProductService{
		repo:        repo,
		categoryRepo: categoryRepo,
		redis:       redisClient,
		cfg:         cfg,
	}
}

func (s *ProductService) GetProduct(id uuid.UUID) (*models.Product, error) {
	return s.repo.FindByID(id)
}

func (s *ProductService) ListProducts() ([]models.Product, error) {
	return s.repo.FindAll()
}

func (s *ProductService) CreateProduct(p *models.Product) error {
	return s.repo.Create(p)
}

func (s *ProductService) UpdateProduct(p *models.Product) error {
	return s.repo.Update(p)
}

func (s *ProductService) DeleteProduct(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *ProductService) ListCategories() ([]models.Category, error) {
	return s.categoryRepo.FindAll()
}

func (s *ProductService) CreateCategory(c *models.Category) error {
	return s.categoryRepo.Create(c)
}
