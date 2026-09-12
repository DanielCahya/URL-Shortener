package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/config"
	"github.com/DanielCahya/url-shortener/internal/middleware"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test, Redis is not running: %v", err)
	}
	// Clean the DB before each test
	client.FlushDB(ctx)
	return client
}

func TestRateLimiter_Anonymous(t *testing.T) {
	client := setupRedis(t)
	defer client.Close()

	cfg := &config.Config{
		RateLimitAnon:   2,
		RateLimitAuth:   10,
		RateLimitWindow: 1, // 1 second window for quick tests
	}

	limiter := middleware.RateLimiter(client, cfg)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1: Allowed
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// Request 2: Allowed
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// Request 3: Rate Limited
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	assert.Equal(t, http.StatusTooManyRequests, rr3.Code)

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Request 4: Allowed again
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.RemoteAddr = "192.168.1.1:1234"
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, req4)
	assert.Equal(t, http.StatusOK, rr4.Code)
}

func TestRateLimiter_Authenticated(t *testing.T) {
	client := setupRedis(t)
	defer client.Close()

	cfg := &config.Config{
		RateLimitAnon:   2,
		RateLimitAuth:   3,
		RateLimitWindow: 60,
	}

	limiter := middleware.RateLimiter(client, cfg)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	userID := uuid.New()

	// Create request with authenticated context
	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := auth.ContextWithUserID(req.Context(), userID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}

	// 1-3 allowed
	assert.Equal(t, http.StatusOK, makeReq().Code)
	assert.Equal(t, http.StatusOK, makeReq().Code)
	assert.Equal(t, http.StatusOK, makeReq().Code)

	// 4th rate limited
	rr := makeReq()
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Contains(t, rr.Body.String(), "rate limit exceeded")
}

func TestRateLimiter_FailOpen(t *testing.T) {
	// Point to an invalid Redis port to simulate Redis being down
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:9999",
		MaxRetries: 0,
		DialTimeout: 10 * time.Millisecond,
	})
	defer client.Close()

	cfg := &config.Config{
		RateLimitAnon:   1, // Very strict limit
		RateLimitWindow: 60,
	}

	limiter := middleware.RateLimiter(client, cfg)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Because it fails open, it should allow requests indefinitely
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}
