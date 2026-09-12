package url

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/google/uuid"
)

const (
	MaxCollisionRetries = 5
)

// Repository defines the data access methods for URL entities.
type Repository interface {
	Create(ctx context.Context, url *URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*URL, error)
	Delete(ctx context.Context, shortCode string, userID uuid.UUID) error
}

// Service defines the business logic for URL operations.
type Service interface {
	CreateURL(ctx context.Context, req CreateURLRequest) (*URLResponse, error)
	ResolveURL(ctx context.Context, shortCode string) (string, error)
	DeleteURL(ctx context.Context, shortCode string) error
}

type service struct {
	repo    Repository
	cache   Cache
	baseURL string
}

// NewService creates a new URL service.
func NewService(repo Repository, cache Cache, baseURL string) Service {
	return &service{
		repo:    repo,
		cache:   cache,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *service) CreateURL(ctx context.Context, req CreateURLRequest) (*URLResponse, error) {
	// 1. Validate Original URL
	trimmedURL := strings.TrimSpace(req.OriginalURL)
	if trimmedURL == "" {
		return nil, ErrInvalidURL
	}

	parsedURL, err := url.ParseRequestURI(trimmedURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return nil, ErrInvalidURL
	}

	// 2. Validate Expiration
	now := time.Now().UTC()
	if req.ExpiresAt != nil && !req.ExpiresAt.After(now) {
		return nil, ErrExpirationInPast
	}

	// 3. Handle Custom Alias vs Generated Short Code
	if req.CustomAlias != nil && strings.TrimSpace(*req.CustomAlias) != "" {
		alias := strings.TrimSpace(*req.CustomAlias)
		if err := ValidateCustomAlias(alias); err != nil {
			return nil, err
		}

		u := &URL{
			ID:          uuid.NewString(),
			ShortCode:   alias,
			OriginalURL: trimmedURL,
			ExpiresAt:   req.ExpiresAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if userID, ok := auth.UserIDFromContext(ctx); ok {
			u.UserID = &userID
		}

		if err := s.repo.Create(ctx, u); err != nil {
			if errors.Is(err, ErrAliasAlreadyExists) {
				return nil, ErrAliasAlreadyExists
			}
			return nil, fmt.Errorf("failed to save custom url: %w", err)
		}

		return s.toResponse(u), nil
	}

	// 4. Generate Random Base62 Short Code with Database Retry on Collision
	for attempt := 0; attempt < MaxCollisionRetries; attempt++ {
		code, err := GenerateRandomBase62(DefaultCodeLen)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random code: %w", err)
		}

		u := &URL{
			ID:          uuid.NewString(),
			ShortCode:   code,
			OriginalURL: trimmedURL,
			ExpiresAt:   req.ExpiresAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if userID, ok := auth.UserIDFromContext(ctx); ok {
			u.UserID = &userID
		}

		err = s.repo.Create(ctx, u)
		if err == nil {
			return s.toResponse(u), nil
		}

		if errors.Is(err, ErrAliasAlreadyExists) {
			// Collision detected: retry generation
			continue
		}

		return nil, fmt.Errorf("failed to persist generated url: %w", err)
	}

	return nil, ErrGenerationCollision
}

func (s *service) ResolveURL(ctx context.Context, shortCode string) (string, error) {
	code := strings.TrimSpace(shortCode)
	if code == "" {
		return "", ErrNotFound
	}

	// 1. Try to fetch from cache first
	if s.cache != nil {
		cachedURL, err := s.cache.GetOriginalURL(ctx, code)
		if err == nil {
			return cachedURL, nil
		}
		// On cache miss, we proceed to DB
	}

	// 2. Cache miss, fetch from database
	u, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return "", err
	}

	// 3. Check expiration
	if u.ExpiresAt != nil && time.Now().UTC().After(*u.ExpiresAt) {
		return "", ErrExpired
	}

	// 4. Update cache asynchronously (or synchronously) to speed up future requests
	if s.cache != nil {
		_ = s.cache.SetOriginalURL(ctx, code, u.OriginalURL) // ignore error on set
	}

	return u.OriginalURL, nil
}

func (s *service) DeleteURL(ctx context.Context, shortCode string) error {
	code := strings.TrimSpace(shortCode)
	if code == "" {
		return ErrNotFound
	}

	// 1. Get user ID from context
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return auth.ErrUnauthorized
	}

	// 2. Delete from DB (ensures ownership)
	err := s.repo.Delete(ctx, code, userID)
	if err != nil {
		return err
	}

	// 3. Invalidate Cache
	if s.cache != nil {
		_ = s.cache.DeleteOriginalURL(ctx, code) // best effort
	}

	return nil
}

func (s *service) toResponse(u *URL) *URLResponse {
	return &URLResponse{
		ID:          u.ID,
		ShortCode:   u.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, u.ShortCode),
		OriginalURL: u.OriginalURL,
		ExpiresAt:   u.ExpiresAt,
		CreatedAt:   u.CreatedAt,
	}
}
