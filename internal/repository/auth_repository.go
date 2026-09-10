package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	
	"github.com/DanielCahya/url-shortener/internal/auth"
)

var (
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrTokenNotFound     = errors.New("refresh token not found")
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *auth.User) error
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*auth.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, hash string) error
}

type postgresAuthRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuthRepository(pool *pgxpool.Pool) AuthRepository {
	return &postgresAuthRepository{pool: pool}
}

func (r *postgresAuthRepository) CreateUser(ctx context.Context, user *auth.User) error {
	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query, user.Email, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
		
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrUserAlreadyExists
		}
		return err
	}
	return nil
}

func (r *postgresAuthRepository) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user auth.User
	err := r.pool.QueryRow(ctx, query, email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
		
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *postgresAuthRepository) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
	return err
}

func (r *postgresAuthRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*auth.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var token auth.RefreshToken
	err := r.pool.QueryRow(ctx, query, hash).
		Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
		
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

func (r *postgresAuthRepository) RevokeRefreshToken(ctx context.Context, hash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	cmdTag, err := r.pool.Exec(ctx, query, hash)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}
