package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"auron/product-service/internal/domain"
	gcsStorage "auron/product-service/internal/storage"

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
	// Skip AutoMigrate if tables already exist — GORM generates malformed ALTER
	// statements when column types use precision specifiers (e.g. numeric(12,2))
	// that differ only in name from what PostgreSQL reports.
	if !db.Migrator().HasTable(&domain.Product{}) {
		if err := db.AutoMigrate(
			&domain.Category{},
			&domain.Product{},
			&domain.Inventory{},
		); err != nil {
			return err
		}
	}

	// product_images is managed via raw SQL so it is always created idempotently
	// regardless of whether the products table already existed.
	return applySearchIndex(db)
}

func applySearchIndex(db *gorm.DB) error {
	statements := []string{
		// product_images — idempotent, safe to run whether the table exists or not
		`CREATE TABLE IF NOT EXISTS product_images (
			id         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
			product_id UUID      NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			url        TEXT      NOT NULL,
			position   INT       NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_product_images_product_id ON product_images(product_id)`,

		// full-text search vector
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

func setupGCS(ctx context.Context, bucketName, credJSON string) (domain.StorageService, error) {
	if bucketName == "" {
		log.Println("GCS_BUCKET_NAME not set — image upload disabled")
		return domain.NoopStorage{}, nil
	}

	svc, err := gcsStorage.NewGCSStorage(ctx, bucketName, credJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise GCS: %w", err)
	}

	log.Printf("GCS storage initialised (bucket: %s)", bucketName)
	return svc, nil
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
