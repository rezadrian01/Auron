package cache

import (
	"context"
	"encoding/json"
	"time"

	"auron/user-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

const (
	UserKeyPrefix = "user:"
	UserListKey   = "users:all"
	CacheTTL      = 5 * time.Minute
)

type UserCache struct {
	client *redis.Client
}

func NewUserCache(client *redis.Client) domain.UserCache {
	return &UserCache{client: client}
}

func (c *UserCache) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	key := UserKeyPrefix + userID
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	var user domain.User
	if err = json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *UserCache) SetUser(ctx context.Context, user *domain.User) error {
	key := UserKeyPrefix + user.ID.String()
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	if err = c.client.Set(ctx, key, data, CacheTTL).Err(); err != nil {
		return err
	}
	return nil
}

func (c *UserCache) DeleteUser(ctx context.Context, userID string) error {
	key := UserKeyPrefix + userID
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

func (c *UserCache) DeleteAll(ctx context.Context) error {
	keys, err := c.client.Keys(ctx, UserKeyPrefix+"*").Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		if err = c.client.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}
	return nil
}
