package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielCahya/url-shortener/internal/url"
	"github.com/redis/go-redis/v9"
)

const (
	urlPrefix = "url:"
	// defaultTTL is the default time a URL resolution is cached
	defaultTTL = 24 * time.Hour
)

type redisURLCache struct {
	client *redis.Client
}

// NewRedisURLCache creates a new URLCache implementation using Redis.
func NewRedisURLCache(client *redis.Client) url.Cache {
	return &redisURLCache{
		client: client,
	}
}

func (c *redisURLCache) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	key := urlPrefix + shortCode

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", url.ErrCacheMiss
		}
		return "", fmt.Errorf("redis get error: %w", err)
	}

	return val, nil
}

func (c *redisURLCache) SetOriginalURL(ctx context.Context, shortCode, originalURL string) error {
	key := urlPrefix + shortCode

	err := c.client.Set(ctx, key, originalURL, defaultTTL).Err()
	if err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

func (c *redisURLCache) DeleteOriginalURL(ctx context.Context, shortCode string) error {
	key := urlPrefix + shortCode

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	return nil
}
