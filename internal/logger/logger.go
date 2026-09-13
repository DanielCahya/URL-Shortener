package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// New initializes and returns a structured JSON logger using log/slog.
func New(serviceName string, levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	baseHandler := slog.NewJSONHandler(os.Stdout, opts)
	handler := &contextHandler{Handler: baseHandler}

	logger := slog.New(handler).With(slog.String("service", serviceName))
	slog.SetDefault(logger)
	return logger
}

// contextHandler injects context values such as request_id into every log record.
type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
	}
	return h.Handler.Handle(ctx, r)
}
