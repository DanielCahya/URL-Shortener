package url

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func mockRequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "test-request-id")
		next.ServeHTTP(w, r)
	})
}

type mockService struct {
	createFunc  func(ctx context.Context, req CreateURLRequest) (*URLResponse, error)
	resolveFunc   func(ctx context.Context, req ResolveRequest) (string, error)
	deleteFunc    func(ctx context.Context, shortCode string) error
	analyticsFunc func(ctx context.Context, shortCode string) (*AnalyticsStats, error)
}

func (m *mockService) CreateURL(ctx context.Context, req CreateURLRequest) (*URLResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockService) ResolveURL(ctx context.Context, req ResolveRequest) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, req)
	}
	return "", nil
}

func (m *mockService) DeleteURL(ctx context.Context, shortCode string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, shortCode)
	}
	return nil
}

func (m *mockService) GetAnalytics(ctx context.Context, shortCode string) (*AnalyticsStats, error) {
	if m.analyticsFunc != nil {
		return m.analyticsFunc(ctx, shortCode)
	}
	return nil, nil
}

func TestHandler_Create(t *testing.T) {
	t.Run("successful url creation returns 201", func(t *testing.T) {
		mockSvc := &mockService{
			createFunc: func(ctx context.Context, req CreateURLRequest) (*URLResponse, error) {
				return &URLResponse{
					ID:          "test-id-1",
					ShortCode:   "Ab3xYz1",
					ShortURL:    "http://localhost:8080/Ab3xYz1",
					OriginalURL: req.OriginalURL,
					CreatedAt:   time.Now().UTC(),
				}, nil
			},
		}

		handler := NewHandler(mockSvc)
		r := chi.NewRouter()
		r.Use(mockRequestIDMiddleware)
		r.Post("/api/v1/urls", handler.Create)

		payload := []byte(`{"original_url": "https://example.com/item/1"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp URLResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.ShortCode != "Ab3xYz1" {
			t.Errorf("expected short_code 'Ab3xYz1', got %q", resp.ShortCode)
		}
	})

	t.Run("conflict error returns 409", func(t *testing.T) {
		mockSvc := &mockService{
			createFunc: func(ctx context.Context, req CreateURLRequest) (*URLResponse, error) {
				return nil, ErrAliasAlreadyExists
			},
		}

		handler := NewHandler(mockSvc)
		r := chi.NewRouter()
		r.Use(mockRequestIDMiddleware)
		r.Post("/api/v1/urls", handler.Create)

		payload := []byte(`{"original_url": "https://example.com", "custom_alias": "taken"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d", rec.Code)
		}

		var errResp APIErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error.Code != "ALIAS_ALREADY_EXISTS" {
			t.Errorf("expected code 'ALIAS_ALREADY_EXISTS', got %q", errResp.Error.Code)
		}
	})
}

func TestHandler_Redirect(t *testing.T) {
	t.Run("successful resolution returns 307 with Location header", func(t *testing.T) {
		mockSvc := &mockService{
			resolveFunc: func(ctx context.Context, req ResolveRequest) (string, error) {
				if req.ShortCode == "Ab3xYz" {
					return "https://example.com/target", nil
				}
				return "", ErrNotFound
			},
		}

		handler := NewHandler(mockSvc)
		r := chi.NewRouter()
		r.Use(mockRequestIDMiddleware)
		r.Get("/{short_code}", handler.Redirect)

		req := httptest.NewRequest(http.MethodGet, "/Ab3xYz", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("expected status 307, got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "https://example.com/target" {
			t.Fatalf("expected Location 'https://example.com/target', got %q", location)
		}
	})

	t.Run("not found returns 404 with structured error", func(t *testing.T) {
		mockSvc := &mockService{
			resolveFunc: func(ctx context.Context, req ResolveRequest) (string, error) {
				return "", ErrNotFound
			},
		}

		handler := NewHandler(mockSvc)
		r := chi.NewRouter()
		r.Use(mockRequestIDMiddleware)
		r.Get("/{short_code}", handler.Redirect)

		req := httptest.NewRequest(http.MethodGet, "/unknownCode", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}

		var errResp APIErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error.Code != "URL_NOT_FOUND" {
			t.Errorf("expected code 'URL_NOT_FOUND', got %q", errResp.Error.Code)
		}
	})

	t.Run("expired url returns 410 Gone", func(t *testing.T) {
		mockSvc := &mockService{
			resolveFunc: func(ctx context.Context, req ResolveRequest) (string, error) {
				return "", ErrExpired
			},
		}

		handler := NewHandler(mockSvc)
		r := chi.NewRouter()
		r.Use(mockRequestIDMiddleware)
		r.Get("/{short_code}", handler.Redirect)

		req := httptest.NewRequest(http.MethodGet, "/expiredCode", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusGone {
			t.Fatalf("expected status 410, got %d", rec.Code)
		}

		var errResp APIErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error.Code != "URL_EXPIRED" {
			t.Errorf("expected code 'URL_EXPIRED', got %q", errResp.Error.Code)
		}
	})
}
