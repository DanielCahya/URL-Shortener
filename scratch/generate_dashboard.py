import json
import os

dashboard = {
  "title": "URL Shortener Metrics",
  "panels": [
    {
      "type": "stat",
      "title": "Total Requests",
      "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0},
      "targets": [{"expr": "sum(http_requests_total)", "refId": "A"}]
    },
    {
      "type": "stat",
      "title": "Redirects Served",
      "gridPos": {"h": 8, "w": 6, "x": 6, "y": 0},
      "targets": [{"expr": "sum(redirect_total)", "refId": "A"}]
    },
    {
      "type": "stat",
      "title": "URLs Created",
      "gridPos": {"h": 8, "w": 6, "x": 12, "y": 0},
      "targets": [{"expr": "sum(url_creation_total)", "refId": "A"}]
    },
    {
      "type": "timeseries",
      "title": "HTTP Request Rate",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8},
      "targets": [{"expr": "sum(rate(http_requests_total[1m])) by (method, status)", "legendFormat": "{{method}} - {{status}}"}]
    },
    {
      "type": "timeseries",
      "title": "P99 Latency (seconds)",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8},
      "targets": [{"expr": "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))", "legendFormat": "p99"}]
    },
    {
      "type": "timeseries",
      "title": "Cache Hit/Miss Rate",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16},
      "targets": [
        {"expr": "rate(redis_cache_hit_total[1m])", "legendFormat": "Hits"},
        {"expr": "rate(redis_cache_miss_total[1m])", "legendFormat": "Misses"}
      ]
    },
    {
      "type": "timeseries",
      "title": "Analytics Pipeline",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 16},
      "targets": [
        {"expr": "rate(outbox_publish_total[1m])", "legendFormat": "Published"},
        {"expr": "rate(analytics_event_processed_total[1m])", "legendFormat": "Processed"}
      ]
    }
  ],
  "schemaVersion": 38,
  "timezone": "browser",
  "refresh": "5s"
}

os.makedirs('deploy/grafana/dashboards', exist_ok=True)
with open('deploy/grafana/dashboards/url_shortener.json', 'w') as f:
    json.dump(dashboard, f, indent=2)
print("Dashboard generated successfully.")
