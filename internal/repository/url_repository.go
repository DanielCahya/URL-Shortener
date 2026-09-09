package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/DanielCahya/url-shortener/internal/url"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URLRepository defines persistent operations for URLs.
type URLRepository interface {
	Create(ctx context.Context, u *url.URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*url.URL, error)
}

type postgresURLRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresURLRepository creates a new instance of PostgreSQL URL repository.
func NewPostgresURLRepository(pool *pgxpool.Pool) URLRepository {
	return &postgresURLRepository{pool: pool}
}

func (r *postgresURLRepository) Create(ctx context.Context, u *url.URL) error {
	query := `
		INSERT INTO urls (id, user_id, short_code, original_url, expires_at, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.UserID,
		u.ShortCode,
		u.OriginalURL,
		u.ExpiresAt,
		u.CreatedAt,
		u.UpdatedAt,
		u.DeletedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return url.ErrAliasAlreadyExists
		}
		return fmt.Errorf("failed to insert url: %w", err)
	}

	return nil
}

func (r *postgresURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	query := `
		SELECT id, user_id, short_code, original_url, expires_at, created_at, updated_at, deleted_at
		FROM urls
		WHERE short_code = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, shortCode)

	var u url.URL
	err := row.Scan(
		&u.ID,
		&u.UserID,
		&u.ShortCode,
		&u.OriginalURL,
		&u.ExpiresAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, url.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get url by short code: %w", err)
	}

	return &u, nil
}
