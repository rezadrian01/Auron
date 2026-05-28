package cmd

import (
	"auron/product-service/internal/domain"
	"context"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(resolveGormLogLevel()),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func runMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Inventory{}); err != nil {
		return err
	}

	// Apply tsvector trigger for full-text search (idempotent raw SQL)
	return applySearchIndex(db)
}

func applySearchIndex(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector`,
		`CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN(search_vector)`,
		`CREATE OR REPLACE FUNCTION products_search_vector_trigger() RETURNS trigger AS $$
		BEGIN
			NEW.search_vector := to_tsvector('english', COALESCE(NEW.name, '') || ' ' || COALESCE(NEW.description, ''));
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS products_search_vector_update ON products`,
		`CREATE TRIGGER products_search_vector_update
			BEFORE INSERT OR UPDATE ON products
			FOR EACH ROW EXECUTE FUNCTION products_search_vector_trigger()`,
		`UPDATE products
			SET search_vector = to_tsvector('english', COALESCE(name, '') || ' ' || COALESCE(description, ''))
			WHERE search_vector IS NULL`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func setupRedis(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

func resolveGormLogLevel() logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GORM_LOG_LEVEL"))) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}
