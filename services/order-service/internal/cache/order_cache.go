package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"auron/order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

const (
	OrderPrefix     = "order:"
	OrderListPrefix = "orders:user:"
	orderTTL        = time.Hour
	orderListTTL    = 5 * time.Minute
)

type OrderCache struct {
	redis *redis.Client
}

func NewOrderCache(redisClient *redis.Client) domain.OrderCache {
	return &OrderCache{redis: redisClient}
}

func (c *OrderCache) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	key := OrderPrefix + orderID

	cached, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(cached), &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (c *OrderCache) SetOrder(ctx context.Context, order *domain.Order) error {
	key := OrderPrefix + order.ID.String()
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, orderTTL).Err()
}

func (c *OrderCache) InvalidateOrder(ctx context.Context, orderID string) error {
	return c.redis.Del(ctx, OrderPrefix+orderID).Err()
}

func (c *OrderCache) GetOrderList(ctx context.Context, cacheKey string) (*domain.OrderListResponse, error) {
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var resp domain.OrderListResponse
	if err := json.Unmarshal([]byte(cached), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *OrderCache) SetOrderList(ctx context.Context, cacheKey string, resp *domain.OrderListResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, cacheKey, data, orderListTTL).Err()
}

// InvalidateOrderList scans and deletes all list cache keys for the given user.
func (c *OrderCache) InvalidateOrderList(ctx context.Context, userID string) error {
	pattern := OrderListPrefix + userID + ":*"
	var cursor uint64

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

// BuildOrderListCacheKey produces a deterministic key for an order list query.
func BuildOrderListCacheKey(userID string, page, limit int) string {
	return fmt.Sprintf("%s%s:page:%d:limit:%d", OrderListPrefix, userID, page, limit)
}
