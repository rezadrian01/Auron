package cache

import (
	"context"
	"encoding/json"
	"time"

	"auron/inventory-service/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	inventoryPrefix = "inventory:"
	inventoryTTL    = 5 * time.Minute
)

type InventoryCache struct {
	redis *redis.Client
}

func NewInventoryCache(redisClient *redis.Client) domain.InventoryCache {
	return &InventoryCache{redis: redisClient}
}

func (c *InventoryCache) GetInventory(ctx context.Context, productID uuid.UUID) (*domain.Inventory, error) {
	key := inventoryPrefix + productID.String()
	cached, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var inv domain.Inventory
	if err := json.Unmarshal([]byte(cached), &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (c *InventoryCache) SetInventory(ctx context.Context, inv *domain.Inventory) error {
	key := inventoryPrefix + inv.ProductID.String()
	data, err := json.Marshal(inv)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, inventoryTTL).Err()
}

func (c *InventoryCache) InvalidateInventory(ctx context.Context, productID uuid.UUID) error {
	return c.redis.Del(ctx, inventoryPrefix+productID.String()).Err()
}
