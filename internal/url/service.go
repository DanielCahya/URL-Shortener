package url

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/metrics"
	"github.com/google/uuid"
	"github.com/mssola/user_agent"
)

const (
	MaxCollisionRetries = 5
)

// Repository defines the data access methods for URL entities.
type Repository interface {
	Create(ctx context.Context, url *URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*URL, error)
	ConsumeURL(ctx context.Context, shortCode string) (*URL, error)
	UpdateEnabled(ctx context.Context, shortCode string, userID uuid.UUID, isEnabled bool) error
	Delete(ctx context.Context, shortCode string, userID uuid.UUID) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*URL, error)
}

type ResolveRequest struct {
	ShortCode string
	UserAgent string
	IPCountry string
	Referer   string
}

// Service defines the business logic for URL operations.
type Service interface {
	CreateURL(ctx context.Context, req CreateURLRequest) (*URLResponse, error)
	ResolveURL(ctx context.Context, req ResolveRequest) (string, error)
	UnlockURL(ctx context.Context, shortCode, password string) (string, error)
	UpdateEnabled(ctx context.Context, shortCode string, isEnabled bool) error
	DeleteURL(ctx context.Context, shortCode string) error
	GetAnalytics(ctx context.Context, shortCode string) (*AnalyticsStats, error)
	ListURLs(ctx context.Context) ([]URLResponse, error)
}

type service struct {
	repo          Repository
	cache         Cache
	analyticsRepo AnalyticsRepository
	statsRepo     StatsRepository
	baseURL       string
}

// NewService creates a new URL service.
func NewService(repo Repository, cache Cache, analyticsRepo AnalyticsRepository, statsRepo StatsRepository, baseURL string) Service {
	return &service{
		repo:          repo,
		cache:         cache,
		analyticsRepo: analyticsRepo,
		statsRepo:     statsRepo,
		baseURL:       strings.TrimRight(baseURL, "/"),
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

	var pwdHash *string
	if req.Password != nil && *req.Password != "" {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		pwdHash = &hash
	}

	// 3. Handle Custom Alias vs Generated Short Code
	if req.CustomAlias != nil && strings.TrimSpace(*req.CustomAlias) != "" {
		alias := strings.TrimSpace(*req.CustomAlias)
		if err := ValidateCustomAlias(alias); err != nil {
			return nil, err
		}

		u := &URL{
			ID:           uuid.NewString(),
			ShortCode:    alias,
			OriginalURL:  trimmedURL,
			ExpiresAt:    req.ExpiresAt,
			MaxAccesses:  req.MaxAccesses,
			PasswordHash: pwdHash,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
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

		metrics.URLCreationTotal.Inc()
		return s.toResponse(u), nil
	}

	// 4. Generate Random Base62 Short Code with Database Retry on Collision
	for attempt := 0; attempt < MaxCollisionRetries; attempt++ {
		code, err := GenerateRandomBase62(DefaultCodeLen)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random code: %w", err)
		}

		u := &URL{
			ID:           uuid.NewString(),
			ShortCode:    code,
			OriginalURL:  trimmedURL,
			ExpiresAt:    req.ExpiresAt,
			MaxAccesses:  req.MaxAccesses,
			PasswordHash: pwdHash,
			IsEnabled:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if userID, ok := auth.UserIDFromContext(ctx); ok {
			u.UserID = &userID
		}

		err = s.repo.Create(ctx, u)
		if err == nil {
			metrics.URLCreationTotal.Inc()
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

func (s *service) ResolveURL(ctx context.Context, req ResolveRequest) (string, error) {
	code := strings.TrimSpace(req.ShortCode)
	if code == "" {
		return "", ErrNotFound
	}

	var urlID string
	var originalURL string

	// 1. Fetch from DB
	u, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return "", err
	}

	// 2. Enforce Link Controls
	if !u.IsEnabled {
		return "", ErrNotFound // Or a specific disabled error
	}

	if u.ExpiresAt != nil && time.Now().UTC().After(*u.ExpiresAt) {
		return "", ErrExpired
	}

	if u.MaxAccesses != nil && u.AccessCount >= *u.MaxAccesses {
		return "", ErrExpired // Effectively "Gone"
	}

	if u.PasswordHash != nil {
		return "", ErrPasswordRequired
	}

	// 3. Atomically consume the URL now that it's validated
	u, err = s.repo.ConsumeURL(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Concurrent consumption reached max_accesses
			return "", ErrExpired
		}
		return "", err
	}

	urlID = u.ID
	originalURL = u.OriginalURL

	// 5. Asynchronously record the click event in the outbox
	// Even though it is recorded "asynchronously" relative to the redirect from the user's perspective (fire and forget),
	// we do it in a non-blocking goroutine or synchronously.
	// Wait, the PDF says: "The redirect request must not synchronously perform analytics database operations."
	// However, if we do it in a goroutine, what if the goroutine crashes before `outbox_events` is written?
	// The outbox pattern usually means writing the outbox event in the same ACID transaction as the business entity.
	// But since this is a read (ResolveURL), there is no business entity transaction!
	// Writing to outbox_events synchronously takes <2ms, but doing it in a goroutine takes 0ms for the redirect.
	// We'll write to outbox_events synchronously but without blocking the HTTP response, OR just synchronously to ensure guaranteed delivery.
	// The PDF: "The redirect request must not synchronously perform analytics database operations... Return 307. The event is processed asynchronously."
	// To strictly follow "must not synchronously perform analytics database operations", we could launch a goroutine to write to the outbox.
	// But the outbox *is* the database operation. If we don't write it synchronously, we risk losing it on process crash.
	// Actually, the typical outbox pattern means writing to the outbox IS the synchronous database operation, and publishing to RabbitMQ is asynchronous.
	// Wait, the PDF says "Application -> BEGIN TRANSACTION -> Business Data -> Outbox Event -> COMMIT." -> "Do not directly depend on successful RabbitMQ publishing".
	// "The business database state and the outbox event must be committed atomically."
	// Since there is no "business state" mutation during a redirect, we just insert the outbox event.
	// If we must not do it synchronously, then what's the point of the outbox? We could just publish to RabbitMQ asynchronously.
	// The intent is likely "Do not synchronously publish to RabbitMQ or update the heavy click_events tables." Writing to the append-only outbox table IS the fast synchronous part.

	// Let's parse user agent
	ua := user_agent.New(req.UserAgent)
	browser, _ := ua.Browser()
	os := ua.OS()
	device := "desktop"
	if ua.Mobile() {
		device = "mobile"
	}
	if ua.Bot() {
		device = "bot"
	}

	clickEvent := ClickEvent{
		EventID:   uuid.NewString(),
		URLID:     urlID,
		Timestamp: time.Now().UTC(),
	}

	if req.IPCountry != "" {
		clickEvent.Country = &req.IPCountry
	}
	if req.UserAgent != "" {
		clickEvent.Device = &device
		clickEvent.Browser = &browser
		clickEvent.OperatingSystem = &os
	}
	if req.Referer != "" {
		clickEvent.Referrer = &req.Referer
	}

	if s.analyticsRepo != nil {
		// Execute in background to ensure redirect isn't blocked by PostgreSQL latency
		go func() {
			// use a new background context with a timeout so it isn't cancelled if the HTTP request closes
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = s.analyticsRepo.RecordClick(ctx, clickEvent)
		}()
	}

	metrics.RedirectTotal.Inc()
	return originalURL, nil
}

func (s *service) UnlockURL(ctx context.Context, shortCode, password string) (string, error) {
	code := strings.TrimSpace(shortCode)
	if code == "" {
		return "", ErrNotFound
	}

	u, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return "", err
	}

	if !u.IsEnabled {
		return "", ErrNotFound
	}
	if u.ExpiresAt != nil && time.Now().UTC().After(*u.ExpiresAt) {
		return "", ErrExpired
	}
	if u.MaxAccesses != nil && u.AccessCount >= *u.MaxAccesses { // >= because we didn't consume it yet
		return "", ErrExpired
	}

	if u.PasswordHash == nil || !auth.CheckPasswordHash(password, *u.PasswordHash) {
		return "", ErrUnauthorized
	}

	// Atomically consume the URL
	u, err = s.repo.ConsumeURL(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrExpired
		}
		return "", err
	}

	return u.OriginalURL, nil
}

func (s *service) UpdateEnabled(ctx context.Context, shortCode string, isEnabled bool) error {
	code := strings.TrimSpace(shortCode)
	if code == "" {
		return ErrNotFound
	}
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return auth.ErrUnauthorized
	}
	return s.repo.UpdateEnabled(ctx, code, userID, isEnabled)
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
		_ = s.cache.DeleteURL(ctx, code) // best effort
	}

	return nil
}

func (s *service) GetAnalytics(ctx context.Context, shortCode string) (*AnalyticsStats, error) {
	code := strings.TrimSpace(shortCode)
	if code == "" {
		return nil, ErrNotFound
	}

	// 1. Check if user is logged in
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthorized
	}

	// 2. Fetch URL to verify existence and ownership
	u, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if u.UserID == nil || u.UserID.String() != userID.String() {
		return nil, ErrNotFound // Hide the fact that it exists from unauthorized users
	}

	// 3. Query stats repository
	return s.statsRepo.GetStats(ctx, u.ID)
}

func (s *service) ListURLs(ctx context.Context) ([]URLResponse, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthorized
	}

	urls, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]URLResponse, 0, len(urls))
	for _, u := range urls {
		responses = append(responses, *s.toResponse(u))
	}

	return responses, nil
}

func (s *service) toResponse(u *URL) *URLResponse {
	return &URLResponse{
		ID:          u.ID,
		ShortCode:   u.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, u.ShortCode),
		OriginalURL: u.OriginalURL,
		ExpiresAt:   u.ExpiresAt,
		IsEnabled:   u.IsEnabled,
		CreatedAt:   u.CreatedAt,
	}
}
