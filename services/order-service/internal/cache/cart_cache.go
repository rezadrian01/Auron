package cache

import (
	"context"
	"encoding/json"
	"time"

	"auron/order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

const (
	CartPrefix  = "cart:"
	cartTTL     = 24 * time.Hour
)

type CartCache struct {
	redis *redis.Client
}

func NewCartCache(redisClient *redis.Client) domain.CartCache {
	return &CartCache{redis: redisClient}
}

func (c *CartCache) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	key := CartPrefix + userID

	cached, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var cart domain.Cart
	if err := json.Unmarshal([]byte(cached), &cart); err != nil {
		return nil, err
	}

	return &cart, nil
}

func (c *CartCache) SetCart(ctx context.Context, cart *domain.Cart) error {
	key := CartPrefix + cart.UserID.String()
	data, err := json.Marshal(cart)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, cartTTL).Err()
}

func (c *CartCache) InvalidateCart(ctx context.Context, userID string) error {
	return c.redis.Del(ctx, CartPrefix+userID).Err()
}
