package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/repository"
)

func TestAuthRepository(t *testing.T) {
	// Assumes test DB is running (via docker-compose test setup)
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test, could not connect to database: %v", err)
	}
	defer pool.Close()

	// Clean up tables
	_, _ = pool.Exec(context.Background(), "TRUNCATE TABLE refresh_tokens CASCADE")
	_, _ = pool.Exec(context.Background(), "TRUNCATE TABLE users CASCADE")

	repo := repository.NewPostgresAuthRepository(pool)
	ctx := context.Background()

	t.Run("Create and Get User", func(t *testing.T) {
		user := &auth.User{
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		}

		err := repo.CreateUser(ctx, user)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)

		// Test Duplicate Email
		err = repo.CreateUser(ctx, &auth.User{Email: "test@example.com", PasswordHash: "other"})
		assert.ErrorIs(t, err, auth.ErrUserAlreadyExists)

		// Get User
		fetchedUser, err := repo.GetUserByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, user.ID, fetchedUser.ID)
		assert.Equal(t, "hashed_password", fetchedUser.PasswordHash)

		// Get Non-Existent
		_, err = repo.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.ErrorIs(t, err, auth.ErrUserNotFound)
	})

	t.Run("Refresh Tokens", func(t *testing.T) {
		// Needs a valid user first
		user := &auth.User{Email: "tokenuser@example.com", PasswordHash: "hash"}
		err := repo.CreateUser(ctx, user)
		require.NoError(t, err)

		token := &auth.RefreshToken{
			UserID:    user.ID,
			TokenHash: "opaque_hash_123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repo.CreateRefreshToken(ctx, token)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, token.ID)

		// Get Token
		fetchedToken, err := repo.GetRefreshTokenByHash(ctx, "opaque_hash_123")
		require.NoError(t, err)
		assert.Equal(t, token.ID, fetchedToken.ID)
		assert.Equal(t, user.ID, fetchedToken.UserID)
		assert.Nil(t, fetchedToken.RevokedAt)

		// Revoke Token
		err = repo.RevokeRefreshToken(ctx, "opaque_hash_123")
		require.NoError(t, err)

		// Get Token Again (should show revoked)
		fetchedToken, err = repo.GetRefreshTokenByHash(ctx, "opaque_hash_123")
		require.NoError(t, err)
		assert.NotNil(t, fetchedToken.RevokedAt)

		// Revoke Non-Existent Token
		err = repo.RevokeRefreshToken(ctx, "invalid_hash")
		assert.ErrorIs(t, err, auth.ErrTokenNotFound)
	})
}
