package url

import (
	"context"
	"time"
)

type ClickEvent struct {
	EventID         string    `json:"event_id"`
	URLID           string    `json:"url_id"`
	Country         *string   `json:"country,omitempty"`
	Device          *string   `json:"device,omitempty"`
	Browser         *string   `json:"browser,omitempty"`
	OperatingSystem *string   `json:"operating_system,omitempty"`
	Referrer        *string   `json:"referrer,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

type AnalyticsRepository interface {
	RecordClick(ctx context.Context, event ClickEvent) error
}

type AnalyticsStats struct {
	TotalClicks int64            `json:"total_clicks"`
	ByCountry   map[string]int64 `json:"by_country"`
	ByDevice    map[string]int64 `json:"by_device"`
	ByBrowser   map[string]int64 `json:"by_browser"`
}

type StatsRepository interface {
	GetStats(ctx context.Context, urlID string) (*AnalyticsStats, error)
}
