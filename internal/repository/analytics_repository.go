package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/DanielCahya/url-shortener/internal/url"
)

type postgresAnalyticsRepository struct {
	outboxRepo OutboxRepository
}

func NewPostgresAnalyticsRepository(outboxRepo OutboxRepository) url.AnalyticsRepository {
	return &postgresAnalyticsRepository{
		outboxRepo: outboxRepo,
	}
}

func (r *postgresAnalyticsRepository) RecordClick(ctx context.Context, event url.ClickEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	outboxEvent := &OutboxEvent{
		ID:            event.EventID,
		EventType:     "url.clicked",
		AggregateType: "url",
		AggregateID:   event.URLID,
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
		Attempts:      0,
	}

	return r.outboxRepo.Create(ctx, outboxEvent)
}
