package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(pool *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		pool:  pool,
		redis: redisClient,
	}
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// Live returns 200 OK as long as the process is alive.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:    "up",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready verifies dependencies before accepting traffic.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	details := make(map[string]string)
	isReady := true

	if err := h.pool.Ping(ctx); err != nil {
		isReady = false
		details["postgres"] = "unreachable: " + err.Error()
	} else {
		details["postgres"] = "ok"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		isReady = false
		details["redis"] = "unreachable: " + err.Error()
	} else {
		details["redis"] = "ok"
	}

	w.Header().Set("Content-Type", "application/json")
	if !isReady {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:    "down",
			Details:   details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:    "up",
		Details:   details,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
