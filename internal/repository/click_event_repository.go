package repository

import (
	"context"

	"github.com/DanielCahya/url-shortener/internal/metrics"
	"github.com/DanielCahya/url-shortener/internal/url"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClickEventRepository interface {
	Insert(ctx context.Context, event url.ClickEvent) error
	GetStats(ctx context.Context, urlID string) (*url.AnalyticsStats, error)
}

type postgresClickEventRepository struct {
	db *pgxpool.Pool
}

func NewPostgresClickEventRepository(db *pgxpool.Pool) ClickEventRepository {
	return &postgresClickEventRepository{db: db}
}

func (r *postgresClickEventRepository) Insert(ctx context.Context, event url.ClickEvent) error {
	query := `
		INSERT INTO click_events (
			id, event_id, url_id, country, device, browser, operating_system, referrer, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) ON CONFLICT (event_id) DO NOTHING
	`

	tag, err := r.db.Exec(ctx, query,
		uuid.NewString(),
		event.EventID,
		event.URLID,
		event.Country,
		event.Device,
		event.Browser,
		event.OperatingSystem,
		event.Referrer,
		event.Timestamp,
	)

	if err == nil && tag.RowsAffected() == 0 {
		metrics.AnalyticsEventDuplicateTotal.Inc()
	}

	return err
}

func (r *postgresClickEventRepository) GetStats(ctx context.Context, urlID string) (*url.AnalyticsStats, error) {
	stats := &url.AnalyticsStats{
		ByCountry: make(map[string]int64),
		ByDevice:  make(map[string]int64),
		ByBrowser: make(map[string]int64),
	}

	// 1. Get total clicks
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM click_events WHERE url_id = $1", urlID).Scan(&stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	// 2. Get clicks by country
	rows, err := r.db.Query(ctx, "SELECT country, COUNT(*) FROM click_events WHERE url_id = $1 AND country IS NOT NULL GROUP BY country", urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var country string
		var count int64
		if err := rows.Scan(&country, &count); err == nil {
			stats.ByCountry[country] = count
		}
	}

	// 3. Get clicks by device
	rows, err = r.db.Query(ctx, "SELECT device, COUNT(*) FROM click_events WHERE url_id = $1 AND device IS NOT NULL GROUP BY device", urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var device string
		var count int64
		if err := rows.Scan(&device, &count); err == nil {
			stats.ByDevice[device] = count
		}
	}

	// 4. Get clicks by browser
	rows, err = r.db.Query(ctx, "SELECT browser, COUNT(*) FROM click_events WHERE url_id = $1 AND browser IS NOT NULL GROUP BY browser", urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var browser string
		var count int64
		if err := rows.Scan(&browser, &count); err == nil {
			stats.ByBrowser[browser] = count
		}
	}

	return stats, nil
}
