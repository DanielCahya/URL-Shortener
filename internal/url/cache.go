package url

import (
	"context"
	"time"
)

// Cache defines the operations for caching URL resolutions.
type Cache interface {
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	SetOriginalURL(ctx context.Context, shortCode, originalURL string, ttl time.Duration) error
	DeleteOriginalURL(ctx context.Context, shortCode string) error
}
