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
