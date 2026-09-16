package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/config"
	"github.com/DanielCahya/url-shortener/internal/metrics"
	"github.com/redis/go-redis/v9"
)

// RateLimiter returns a middleware that limits requests using a Redis fixed-window counter.
// It fails open if Redis is unavailable.
func RateLimiter(redisClient *redis.Client, cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var key string
			var limit int

			// Determine if request is authenticated
			userID, ok := auth.UserIDFromContext(ctx)
			if ok {
				key = fmt.Sprintf("rate_limit:auth:%s", userID.String())
				limit = cfg.RateLimitAuth
			} else {
				// Fallback to IP address for anonymous users
				ip := r.RemoteAddr // In a real app behind a proxy, extract from X-Forwarded-For securely
				key = fmt.Sprintf("rate_limit:anonymous:%s", ip)
				limit = cfg.RateLimitAnon
			}

			// Use Redis pipeline for INCR and EXPIRE to minimize roundtrips
			// Note: We only want to set expiration if it's a new key, but doing it every time
			// Let's execute INCR first, check if it's 1, then EXPIRE.
			// Actually, a simple INCR then EXPIRE if 1 is standard.
			count, redisErr := redisClient.Incr(ctx, key).Result()
			if redisErr != nil {
				// Fail-open
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				redisClient.Expire(ctx, key, time.Duration(cfg.RateLimitWindow)*time.Second)
			}

			if count > int64(limit) {
				metrics.RateLimitExceededTotal.WithLabelValues(r.URL.Path).Inc()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", cfg.RateLimitWindow))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error": "rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
