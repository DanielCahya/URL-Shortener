package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/DanielCahya/url-shortener/internal/auth"
)

func TestTokenService(t *testing.T) {
	cfg := auth.JWTConfig{
		SecretKey:  "supersecretkey",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		Issuer:     "url-shortener",
	}

	service := auth.NewTokenService(cfg)
	userID := uuid.New()

	t.Run("GenerateTokenPair", func(t *testing.T) {
		pair, err := service.GenerateTokenPair(userID)
		if err != nil {
			t.Fatalf("unexpected error generating tokens: %v", err)
		}
		if pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatalf("expected non-empty tokens, got access: %q, refresh: %q", pair.AccessToken, pair.RefreshToken)
		}

		// Validate the generated access token
		parsedUserID, err := service.ValidateAccessToken(pair.AccessToken)
		if err != nil {
			t.Fatalf("unexpected error validating access token: %v", err)
		}
		if parsedUserID != userID {
			t.Fatalf("expected user ID %v, got %v", userID, parsedUserID)
		}
	})

	t.Run("InvalidAccessToken", func(t *testing.T) {
		_, err := service.ValidateAccessToken("invalid.token.string")
		if err == nil {
			t.Fatalf("expected error validating invalid access token, got nil")
		}
	})

	t.Run("HashRefreshToken", func(t *testing.T) {
		token := "my-opaque-token"
		hash1 := auth.HashRefreshToken(token)
		hash2 := auth.HashRefreshToken(token)
		
		if hash1 != hash2 {
			t.Fatalf("expected deterministic hash, got %q and %q", hash1, hash2)
		}
		if hash1 == token {
			t.Fatalf("expected hash to be different from token, but they are identical")
		}
	})
}
