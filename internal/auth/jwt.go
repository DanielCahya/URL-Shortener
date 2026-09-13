package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrUnauthorized = errors.New("unauthorized access")
)

// JWTConfig holds configuration for generating and validating tokens.
type JWTConfig struct {
	SecretKey  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
}

// TokenPair contains the generated access and refresh tokens.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// TokenService handles generation and validation of JWTs and Refresh Tokens.
type TokenService struct {
	config JWTConfig
}

func NewTokenService(cfg JWTConfig) *TokenService {
	return &TokenService{
		config: cfg,
	}
}

// GenerateTokenPair generates a new JWT access token and an opaque refresh token.
func (s *TokenService) GenerateTokenPair(userID uuid.UUID) (TokenPair, error) {
	now := time.Now()

	// 1. Generate Access Token (JWT)
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iss": s.config.Issuer,
		"iat": now.Unix(),
		"exp": now.Add(s.config.AccessTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return TokenPair{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Generate Refresh Token (Opaque)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return TokenPair{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken := base64.URLEncoding.EncodeToString(b)

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateAccessToken parses and validates a JWT access token, returning the User ID (sub claim).
func (s *TokenService) ValidateAccessToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	// Verify Issuer
	iss, ok := claims["iss"].(string)
	if !ok || iss != s.config.Issuer {
		return uuid.Nil, ErrInvalidToken
	}

	subStr, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(subStr)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}

// HashRefreshToken generates a SHA-256 hash of the opaque refresh token for database storage.
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}
