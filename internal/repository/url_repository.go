package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/DanielCahya/url-shortener/internal/url"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URLRepository defines persistent operations for URLs.
type URLRepository interface {
	Create(ctx context.Context, u *url.URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*url.URL, error)
	Delete(ctx context.Context, shortCode string, userID uuid.UUID) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*url.URL, error)
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

func (r *postgresURLRepository) Delete(ctx context.Context, shortCode string, userID uuid.UUID) error {
	query := `
		UPDATE urls
		SET deleted_at = NOW()
		WHERE short_code = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	cmdTag, err := r.pool.Exec(ctx, query, shortCode, userID)
	if err != nil {
		return fmt.Errorf("failed to delete url: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return url.ErrNotFound
	}

	return nil
}

func (r *postgresURLRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*url.URL, error) {
	query := `
		SELECT id, user_id, short_code, original_url, expires_at, created_at, updated_at, deleted_at
		FROM urls
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list urls: %w", err)
	}
	defer rows.Close()

	var urls []*url.URL
	for rows.Next() {
		var u url.URL
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan url: %w", err)
		}
		urls = append(urls, &u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating urls: %w", err)
	}

	return urls, nil
}
