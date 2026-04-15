package cache

import (
	"auron/product-service/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ProductDetailPrefix = "product:"
	ProductListPrefix   = "product:list"
	cacheTTL            = 5 * time.Minute
)

type ProductCache struct {
	redis *redis.Client
}

func NewProductCache(redisClient *redis.Client) domain.ProductCache {
	return &ProductCache{redis: redisClient}
}

func (c *ProductCache) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	key := ProductDetailPrefix + id

	cached, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var product domain.Product
	if err := json.Unmarshal([]byte(cached), &product); err != nil {
		return nil, err
	}

	return &product, nil
}

func (c *ProductCache) SetProduct(ctx context.Context, product *domain.Product) error {
	key := ProductDetailPrefix + product.ID.String()
	data, err := json.Marshal(product)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, data, cacheTTL).Err()
}

func (c *ProductCache) DeleteProduct(ctx context.Context, id string) error {
	key := ProductDetailPrefix + id
	return c.redis.Del(ctx, key).Err()
}

func (c *ProductCache) GetProductList(ctx context.Context, cacheKey string) (*domain.ProductListResponse, error) {
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var response domain.ProductListResponse
	if err := json.Unmarshal([]byte(cached), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *ProductCache) SetProductList(ctx context.Context, cacheKey string, response *domain.ProductListResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, cacheKey, data, cacheTTL).Err()
}

func (c *ProductCache) InvalidateProductList(ctx context.Context) error {
	var cursor uint64
	pattern := ProductListPrefix + "*"

	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.redis.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (c *ProductCache) ClearAll(ctx context.Context) error {
	return c.redis.FlushDB(ctx).Err()
}

func GenerateCacheKey(filter domain.ProductFilter) string {
	// create a has from filter params for list caching
	hash := fmt.Sprintf("%s_%s_%s_%s_%d_%d", filter.Q, filter.CategoryID.String(), fmt.Sprintf("%.2f", *filter.MinPrice), fmt.Sprintf("%.2f", *filter.MaxPrice), filter.Page, filter.Limit)

	return ProductListPrefix + hash
}
