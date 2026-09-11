package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DanielCahya/url-shortener/internal/auth"
)

type mockAuthRepo struct {
	users  map[string]*auth.User
	tokens map[string]*auth.RefreshToken
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, user *auth.User) error {
	if _, exists := m.users[user.Email]; exists {
		return auth.ErrUserAlreadyExists
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return nil
}

func (m *mockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, auth.ErrUserNotFound
}

func (m *mockAuthRepo) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockAuthRepo) GetRefreshTokenByHash(ctx context.Context, hash string) (*auth.RefreshToken, error) {
	if token, exists := m.tokens[hash]; exists {
		return token, nil
	}
	return nil, auth.ErrTokenNotFound
}

func (m *mockAuthRepo) RevokeRefreshToken(ctx context.Context, hash string) error {
	if token, exists := m.tokens[hash]; exists {
		now := time.Now()
		token.RevokedAt = &now
		return nil
	}
	return auth.ErrTokenNotFound
}

func TestAuthService_Register(t *testing.T) {
	repo := &mockAuthRepo{
		users:  make(map[string]*auth.User),
		tokens: make(map[string]*auth.RefreshToken),
	}
	svc := auth.NewAuthService(repo, nil)

	ctx := context.Background()

	t.Run("Valid Registration", func(t *testing.T) {
		req := auth.RegisterRequest{Email: "test@example.com", Password: "securepassword"}
		user, err := svc.Register(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.NotEmpty(t, user.PasswordHash)
	})

	t.Run("Invalid Email", func(t *testing.T) {
		req := auth.RegisterRequest{Email: "invalid", Password: "securepassword"}
		_, err := svc.Register(ctx, req)
		assert.ErrorIs(t, err, auth.ErrInvalidEmail)
	})

	t.Run("Short Password", func(t *testing.T) {
		req := auth.RegisterRequest{Email: "test2@example.com", Password: "short"}
		_, err := svc.Register(ctx, req)
		assert.ErrorIs(t, err, auth.ErrPasswordTooShort)
	})

	t.Run("Duplicate Email", func(t *testing.T) {
		req := auth.RegisterRequest{Email: "test@example.com", Password: "anotherpassword"}
		_, err := svc.Register(ctx, req)
		assert.ErrorIs(t, err, auth.ErrUserAlreadyExists)
	})
}

func TestAuthService_Login(t *testing.T) {
	repo := &mockAuthRepo{
		users:  make(map[string]*auth.User),
		tokens: make(map[string]*auth.RefreshToken),
	}
	tokenService := auth.NewTokenService(auth.JWTConfig{
		SecretKey:  "secret",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
		Issuer:     "test",
	})
	svc := auth.NewAuthService(repo, tokenService)

	ctx := context.Background()

	// Seed user
	hash, _ := auth.HashPassword("securepassword")
	user := &auth.User{ID: uuid.New(), Email: "test@example.com", PasswordHash: hash}
	repo.users[user.Email] = user

	t.Run("Valid Login", func(t *testing.T) {
		req := auth.LoginRequest{Email: "test@example.com", Password: "securepassword"}
		resp, err := svc.Login(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("Invalid Password", func(t *testing.T) {
		req := auth.LoginRequest{Email: "test@example.com", Password: "wrongpassword"}
		_, err := svc.Login(ctx, req)
		assert.ErrorContains(t, err, "invalid email or password")
	})

	t.Run("User Not Found", func(t *testing.T) {
		req := auth.LoginRequest{Email: "notfound@example.com", Password: "securepassword"}
		_, err := svc.Login(ctx, req)
		assert.ErrorContains(t, err, "invalid email or password")
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	repo := &mockAuthRepo{
		users:  make(map[string]*auth.User),
		tokens: make(map[string]*auth.RefreshToken),
	}
	tokenService := auth.NewTokenService(auth.JWTConfig{
		SecretKey:  "secret",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
		Issuer:     "test",
	})
	svc := auth.NewAuthService(repo, tokenService)

	ctx := context.Background()
	userID := uuid.New()

	// Create valid token
	tokenPair, _ := tokenService.GenerateTokenPair(userID)
	hash := auth.HashRefreshToken(tokenPair.RefreshToken)
	repo.tokens[hash] = &auth.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Create revoked token
	revokedTokenPair, _ := tokenService.GenerateTokenPair(userID)
	revokedHash := auth.HashRefreshToken(revokedTokenPair.RefreshToken)
	now := time.Now()
	repo.tokens[revokedHash] = &auth.RefreshToken{
		UserID:    userID,
		TokenHash: revokedHash,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &now,
	}

	t.Run("Valid Refresh", func(t *testing.T) {
		req := auth.RefreshRequest{RefreshToken: tokenPair.RefreshToken}
		resp, err := svc.RefreshToken(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.NotEqual(t, tokenPair.RefreshToken, resp.RefreshToken)
	})

	t.Run("Revoked Token", func(t *testing.T) {
		req := auth.RefreshRequest{RefreshToken: revokedTokenPair.RefreshToken}
		_, err := svc.RefreshToken(ctx, req)
		assert.ErrorContains(t, err, "invalid refresh token")
	})

	t.Run("Unknown Token", func(t *testing.T) {
		req := auth.RefreshRequest{RefreshToken: "unknown_token"}
		_, err := svc.RefreshToken(ctx, req)
		assert.ErrorContains(t, err, "invalid refresh token")
	})
}
