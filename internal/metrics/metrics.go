package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	HTTPErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP 5xx errors",
		},
		[]string{"path"},
	)

	RedirectTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redirect_total",
			Help: "Total number of successful URL redirects",
		},
	)

	URLCreationTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_creation_total",
			Help: "Total number of shortened URLs created",
		},
	)

	RedisCacheHitTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_hit_total",
			Help: "Total number of successful Redis cache hits",
		},
	)

	RedisCacheMissTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_miss_total",
			Help: "Total number of Redis cache misses",
		},
	)

	DatabaseErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_errors_total",
			Help: "Total number of PostgreSQL database errors",
		},
		[]string{"operation"},
	)

	AuthenticationFailureTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authentication_failure_total",
			Help: "Total number of failed authentication attempts",
		},
		[]string{"reason"},
	)

	RateLimitExceededTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_exceeded_total",
			Help: "Total number of rate limit rejections",
		},
		[]string{"path"},
	)

	OutboxPublishTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "outbox_publish_total",
			Help: "Total number of events published from the outbox",
		},
	)

	OutboxPublishFailureTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "outbox_publish_failure_total",
			Help: "Total number of failed outbox publish attempts",
		},
	)

	AnalyticsEventProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_event_processed_total",
			Help: "Total number of analytics events successfully processed",
		},
	)

	AnalyticsEventFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_event_failed_total",
			Help: "Total number of analytics events that failed processing",
		},
	)

	AnalyticsEventDuplicateTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_event_duplicate_total",
			Help: "Total number of duplicate analytics events dropped",
		},
	)
)
