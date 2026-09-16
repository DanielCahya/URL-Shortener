package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/DanielCahya/url-shortener/internal/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Metrics records HTTP request duration and count.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()

		// Use chi's route context to get the matched route pattern,
		// to avoid high cardinality issues with raw URLs (e.g. /urls/123).
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "unmatched"
		}

		status := strconv.Itoa(ww.Status())

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, routePattern, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration)

		if ww.Status() >= 500 {
			metrics.HTTPErrorsTotal.WithLabelValues(routePattern).Inc()
		}
	})
}
