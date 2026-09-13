package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/repository"
	"github.com/redis/go-redis/v9"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func Idempotency(redisClient *redis.Client, repo repository.IdempotencyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idempotencyKey := r.Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Spec: If URL creation is currently protected by authentication, require authentication.
			// Since URL creation has OptionalAuth, we enforce auth when idempotency is used.
			userID, ok := auth.UserIDFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "idempotency requires authentication"}`))
				return
			}

			ctx := r.Context()
			redisKey := fmt.Sprintf("idempotency:%s:%s", userID.String(), idempotencyKey)

			// Layer 1: Redis coordination
			// SETNX to prevent concurrent identical requests from hammering the database
			acquired, err := redisClient.SetNX(ctx, redisKey, "processing", 24*time.Hour).Result()
			if err != nil {
				// Fail-open the coordination layer, fallback to DB
				acquired = true
			}

			if !acquired {
				// Another request is processing or recently completed.
				// We poll the database for a short time to wait for completion.
				for i := 0; i < 5; i++ {
					record, dbErr := repo.Find(ctx, userID, idempotencyKey)
					if dbErr == nil && record != nil {
						if record.Status == repository.IdempotencyStateCompleted {
							w.Header().Set("Content-Type", "application/json")
							w.Header().Set("X-Idempotent-Replayed", "true")
							w.WriteHeader(*record.ResponseStatus)
							w.Write(record.ResponseBody)
							return
						}
					}
					time.Sleep(200 * time.Millisecond)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "request already processing"}`))
				return
			}

			// Layer 2: PostgreSQL correctness
			record, err := repo.Find(ctx, userID, idempotencyKey)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if record != nil {
				// Already processed (cache miss, DB hit)
				if record.Status == repository.IdempotencyStateCompleted {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Idempotent-Replayed", "true")
					w.WriteHeader(*record.ResponseStatus)
					w.Write(record.ResponseBody)
					return
				}
				// If it's failed or stuck in processing in DB, we could retry.
				// For simplicity, we just reject if it's currently marked as processing in DB.
				if record.Status == repository.IdempotencyStateProcessing {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					w.Write([]byte(`{"error": "request already processing in db"}`))
					return
				}
			}

			// Record is new, create in DB
			expiresAt := time.Now().Add(24 * time.Hour)
			err = repo.Create(ctx, userID, idempotencyKey, expiresAt)
			if err != nil {
				// Could be a unique constraint violation from a concurrent request that bypassed Redis
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error": "request already processing"}`))
				return
			}

			// Execute the actual handler, capturing the response
			rec := &responseRecorder{
				ResponseWriter: w,
				status:         0,
				body:           new(bytes.Buffer),
			}

			next.ServeHTTP(rec, r)

			// Store the response
			if rec.status >= 200 && rec.status < 300 {
				repo.UpdateResponse(ctx, userID, idempotencyKey, rec.status, rec.body.Bytes())
			} else {
				// If it failed (e.g. 400 Bad Request), we might mark as failed so they can retry
				// Or we store the 400 so we don't re-process invalid data.
				// Storing the 400 is safer idempotency.
				repo.UpdateResponse(ctx, userID, idempotencyKey, rec.status, rec.body.Bytes())
			}
		})
	}
}
