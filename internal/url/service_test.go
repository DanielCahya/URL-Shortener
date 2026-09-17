package url

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepository struct {
	mu            sync.Mutex
	urls          map[string]*URL
	failCount     int
	attemptsCount int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		urls: make(map[string]*URL),
	}
}

func (m *mockRepository) Create(ctx context.Context, u *URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.attemptsCount++
	if m.failCount > 0 {
		m.failCount--
		return ErrAliasAlreadyExists
	}

	if _, exists := m.urls[u.ShortCode]; exists {
		return ErrAliasAlreadyExists
	}
	m.urls[u.ShortCode] = u
	return nil
}

func (m *mockRepository) GetByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.urls[shortCode]
	if !exists {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *mockRepository) Delete(ctx context.Context, shortCode string, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.urls[shortCode]
	if !exists {
		return ErrNotFound
	}

	if u.UserID == nil || *u.UserID != userID {
		return ErrNotFound // Or unauthorized, but ErrNotFound is what we return if rows = 0
	}

	delete(m.urls, shortCode)
	return nil
}

func TestService_CreateURL(t *testing.T) {
	ctx := context.Background()

	t.Run("successful random generation", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		resp, err := svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com/item/123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.ID == "" {
			t.Error("expected non-empty ID")
		}
		if len(resp.ShortCode) != DefaultCodeLen {
			t.Errorf("expected short_code length %d, got %d", DefaultCodeLen, len(resp.ShortCode))
		}
		if resp.ShortURL != "http://localhost:8080/"+resp.ShortCode {
			t.Errorf("unexpected short_url: %s", resp.ShortURL)
		}
		if resp.OriginalURL != "https://example.com/item/123" {
			t.Errorf("unexpected original_url: %s", resp.OriginalURL)
		}
	})

	t.Run("invalid url schemes", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		invalidURLs := []string{
			"",
			"   ",
			"ftp://example.com/file",
			"example.com",
			"http://",
		}

		for _, u := range invalidURLs {
			_, err := svc.CreateURL(ctx, CreateURLRequest{OriginalURL: u})
			if err != ErrInvalidURL {
				t.Errorf("expected ErrInvalidURL for %q, got %v", u, err)
			}
		}
	})

	t.Run("expiration in past returns error", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		past := time.Now().UTC().Add(-1 * time.Hour)
		_, err := svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com",
			ExpiresAt:   &past,
		})
		if err != ErrExpirationInPast {
			t.Fatalf("expected ErrExpirationInPast, got %v", err)
		}
	})

	t.Run("custom alias success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		alias := "custom-doc"
		resp, err := svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com/docs",
			CustomAlias: &alias,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ShortCode != alias {
			t.Errorf("expected short_code %s, got %s", alias, resp.ShortCode)
		}
	})

	t.Run("custom alias duplicate returns error", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		alias := "my-project"
		_, err := svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com/proj",
			CustomAlias: &alias,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com/another",
			CustomAlias: &alias,
		})
		if err != ErrAliasAlreadyExists {
			t.Fatalf("expected ErrAliasAlreadyExists, got %v", err)
		}
	})

	t.Run("collision retry logic succeeds within limit", func(t *testing.T) {
		repo := newMockRepository()
		repo.failCount = 2 // fail first 2 attempts
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		resp, err := svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com/test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if repo.attemptsCount != 3 {
			t.Errorf("expected 3 attempts, got %d", repo.attemptsCount)
		}
	})

	t.Run("collision retry exceeds limit", func(t *testing.T) {
		repo := newMockRepository()
		repo.failCount = 10 // exceed MaxCollisionRetries (5)
		svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

		_, err := svc.CreateURL(ctx, CreateURLRequest{
			OriginalURL: "https://example.com/test",
		})
		if err != ErrGenerationCollision {
			t.Fatalf("expected ErrGenerationCollision, got %v", err)
		}
	})
}

func TestService_ResolveURL(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	svc := NewService(repo, nil, nil, nil, "http://localhost:8080")

	// Pre-populate active URL
	_ = repo.Create(ctx, &URL{
		ID:          "url-1",
		ShortCode:   "active1",
		OriginalURL: "https://example.com/active",
	})

	// Pre-populate expired URL
	past := time.Now().UTC().Add(-1 * time.Hour)
	_ = repo.Create(ctx, &URL{
		ID:          "url-2",
		ShortCode:   "expired1",
		OriginalURL: "https://example.com/expired",
		ExpiresAt:   &past,
	})

	t.Run("resolve existing active url", func(t *testing.T) {
		target, err := svc.ResolveURL(ctx, ResolveRequest{ShortCode: "active1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if target != "https://example.com/active" {
			t.Errorf("unexpected target url: %s", target)
		}
	})

	t.Run("resolve non-existent url returns ErrNotFound", func(t *testing.T) {
		_, err := svc.ResolveURL(ctx, ResolveRequest{ShortCode: "nonexistent"})
		if err != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("resolve expired url returns ErrExpired", func(t *testing.T) {
		_, err := svc.ResolveURL(ctx, ResolveRequest{ShortCode: "expired1"})
		if err != ErrExpired {
			t.Fatalf("expected ErrExpired, got %v", err)
		}
	})
}
