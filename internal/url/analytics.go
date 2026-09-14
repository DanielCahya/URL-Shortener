package url

import (
	"context"
	"time"
)

type ClickEvent struct {
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
