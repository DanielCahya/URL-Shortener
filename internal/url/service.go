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

// Repository defines the persistence interface required by URLService.
type Repository interface {
	Create(ctx context.Context, u *URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*URL, error)
}

// Service defines business operations for URLs.
type Service interface {
	CreateURL(ctx context.Context, req CreateURLRequest) (*URLResponse, error)
	ResolveURL(ctx context.Context, shortCode string) (string, error)
}

type service struct {
	repo    Repository
	baseURL string
}

// NewService creates a new URL service instance.
func NewService(repo Repository, baseURL string) Service {
	return &service{
		repo:    repo,
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

	u, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return "", err
	}

	// Check expiration
	if u.ExpiresAt != nil && time.Now().UTC().After(*u.ExpiresAt) {
		return "", ErrExpired
	}

	return u.OriginalURL, nil
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
