package cache

import (
	"context"
	"encoding/json"
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

func (c *redisURLCache) GetURL(ctx context.Context, shortCode string) (*url.CachedURL, error) {
	key := urlPrefix + shortCode

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, url.ErrCacheMiss
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var cachedURL url.CachedURL
	if err := json.Unmarshal([]byte(val), &cachedURL); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached url: %w", err)
	}

	return &cachedURL, nil
}

func (c *redisURLCache) SetURL(ctx context.Context, shortCode string, u *url.CachedURL, ttl time.Duration) error {
	key := urlPrefix + shortCode

	if ttl == 0 {
		ttl = defaultTTL
	}

	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("failed to marshal cached url: %w", err)
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

func (c *redisURLCache) DeleteURL(ctx context.Context, shortCode string) error {
	key := urlPrefix + shortCode

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	return nil
}
