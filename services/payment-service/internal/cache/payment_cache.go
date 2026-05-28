package cache

import (
	"context"
	"encoding/json"
	"time"

	"auron/payment-service/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	paymentPrefix = "payment:"
	paymentTTL    = time.Hour
)

type PaymentCache struct {
	redis *redis.Client
}

func NewPaymentCache(redisClient *redis.Client) domain.PaymentCache {
	return &PaymentCache{redis: redisClient}
}

func (c *PaymentCache) GetPayment(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error) {
	key := paymentPrefix + paymentID.String()
	cached, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var payment domain.Payment
	if err := json.Unmarshal([]byte(cached), &payment); err != nil {
		return nil, err
	}
	return &payment, nil
}

func (c *PaymentCache) SetPayment(ctx context.Context, payment *domain.Payment) error {
	key := paymentPrefix + payment.ID.String()
	data, err := json.Marshal(payment)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, paymentTTL).Err()
}

func (c *PaymentCache) InvalidatePayment(ctx context.Context, paymentID uuid.UUID) error {
	return c.redis.Del(ctx, paymentPrefix+paymentID.String()).Err()
}
