package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxEvent struct {
	ID            string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
	Attempts      int
}

type OutboxRepository interface {
	Create(ctx context.Context, event *OutboxEvent) error

	// Methods for the background publisher worker
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error
	FetchUnpublishedTx(ctx context.Context, tx pgx.Tx, batchSize int) ([]*OutboxEvent, error)
	MarkPublishedTx(ctx context.Context, tx pgx.Tx, ids []string) error
	IncrementAttemptsTx(ctx context.Context, tx pgx.Tx, ids []string) error
}

type postgresOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOutboxRepository(pool *pgxpool.Pool) OutboxRepository {
	return &postgresOutboxRepository{pool: pool}
}

func (r *postgresOutboxRepository) Create(ctx context.Context, event *OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, created_at, published_at, attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		event.ID, event.EventType, event.AggregateType, event.AggregateID, event.Payload,
		event.CreatedAt, event.PublishedAt, event.Attempts,
	)
	return err
}

func (r *postgresOutboxRepository) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *postgresOutboxRepository) FetchUnpublishedTx(ctx context.Context, tx pgx.Tx, batchSize int) ([]*OutboxEvent, error) {
	query := `
		SELECT id, event_type, aggregate_type, aggregate_id, payload, created_at, published_at, attempts
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.Query(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		err := rows.Scan(
			&e.ID, &e.EventType, &e.AggregateType, &e.AggregateID, &e.Payload,
			&e.CreatedAt, &e.PublishedAt, &e.Attempts,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *postgresOutboxRepository) MarkPublishedTx(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `
		UPDATE outbox_events
		SET published_at = NOW()
		WHERE id = ANY($1)
	`
	_, err := tx.Exec(ctx, query, ids)
	return err
}

func (r *postgresOutboxRepository) IncrementAttemptsTx(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `
		UPDATE outbox_events
		SET attempts = attempts + 1
		WHERE id = ANY($1)
	`
	_, err := tx.Exec(ctx, query, ids)
	return err
}
