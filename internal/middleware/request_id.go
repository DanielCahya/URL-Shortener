package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/DanielCahya/url-shortener/internal/logger"
)

const HeaderXRequestID = "X-Request-ID"

// RequestID middleware ensures every request has a unique request ID.
// If the incoming request has an X-Request-ID header, it is used; otherwise, a random ID is generated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderXRequestID)
		if reqID == "" {
			reqID = generateRequestID()
		}

		// Set header in response
		w.Header().Set(HeaderXRequestID, reqID)

		// Attach to context
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(logger.RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(b)
}
