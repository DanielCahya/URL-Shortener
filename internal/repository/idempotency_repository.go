package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyState string

const (
	IdempotencyStateProcessing IdempotencyState = "processing"
	IdempotencyStateCompleted  IdempotencyState = "completed"
	IdempotencyStateFailed     IdempotencyState = "failed"
)

type IdempotencyKey struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	IdempotencyKey string
	Status         IdempotencyState
	ResponseStatus *int
	ResponseBody   []byte
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

type IdempotencyRepository interface {
	Find(ctx context.Context, userID uuid.UUID, key string) (*IdempotencyKey, error)
	Create(ctx context.Context, userID uuid.UUID, key string, expiresAt time.Time) error
	UpdateResponse(ctx context.Context, userID uuid.UUID, key string, status int, body []byte) error
	MarkFailed(ctx context.Context, userID uuid.UUID, key string) error
}

type postgresIdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIdempotencyRepository(pool *pgxpool.Pool) IdempotencyRepository {
	return &postgresIdempotencyRepository{pool: pool}
}

func (r *postgresIdempotencyRepository) Find(ctx context.Context, userID uuid.UUID, key string) (*IdempotencyKey, error) {
	query := `
		SELECT id, user_id, idempotency_key, status, response_status, response_body, created_at, expires_at
		FROM idempotency_keys
		WHERE user_id = $1 AND idempotency_key = $2
	`
	var i IdempotencyKey
	err := r.pool.QueryRow(ctx, query, userID, key).Scan(
		&i.ID, &i.UserID, &i.IdempotencyKey, &i.Status, &i.ResponseStatus, &i.ResponseBody, &i.CreatedAt, &i.ExpiresAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *postgresIdempotencyRepository) Create(ctx context.Context, userID uuid.UUID, key string, expiresAt time.Time) error {
	query := `
		INSERT INTO idempotency_keys (user_id, idempotency_key, status, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, userID, key, IdempotencyStateProcessing, expiresAt)
	return err
}

func (r *postgresIdempotencyRepository) UpdateResponse(ctx context.Context, userID uuid.UUID, key string, status int, body []byte) error {
	query := `
		UPDATE idempotency_keys
		SET status = $1, response_status = $2, response_body = $3
		WHERE user_id = $4 AND idempotency_key = $5
	`
	_, err := r.pool.Exec(ctx, query, IdempotencyStateCompleted, status, body, userID, key)
	return err
}

func (r *postgresIdempotencyRepository) MarkFailed(ctx context.Context, userID uuid.UUID, key string) error {
	query := `
		UPDATE idempotency_keys
		SET status = $1
		WHERE user_id = $2 AND idempotency_key = $3
	`
	_, err := r.pool.Exec(ctx, query, IdempotencyStateFailed, userID, key)
	return err
}
