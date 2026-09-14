package url

import (
	"context"
	"time"
)

// CachedURL represents the URL data stored in cache.
type CachedURL struct {
	ID          string `json:"id"`
	OriginalURL string `json:"original_url"`
}

// Cache defines the operations for caching URL resolutions.
type Cache interface {
	GetURL(ctx context.Context, shortCode string) (*CachedURL, error)
	SetURL(ctx context.Context, shortCode string, u *CachedURL, ttl time.Duration) error
	DeleteURL(ctx context.Context, shortCode string) error
}
