package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*User, error)
	Login(ctx context.Context, req LoginRequest) (*TokenResponse, error)
	RefreshToken(ctx context.Context, req RefreshRequest) (*TokenResponse, error)
	Logout(ctx context.Context, req LogoutRequest) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error)
}

type authService struct {
	repo         AuthRepository
	tokenService *TokenService
}

func NewAuthService(repo AuthRepository, tokenService *TokenService) AuthService {
	return &authService{
		repo:         repo,
		tokenService: tokenService,
	}
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*User, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return nil, ErrInvalidEmail
	}

	if len(req.Password) < 8 {
		return nil, ErrPasswordTooShort
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Email:        req.Email,
		PasswordHash: hash,
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return nil, ErrInvalidEmail
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, errors.New("invalid email or password") // Obfuscate error
		}
		return nil, err
	}

	if !CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	tokenPair, err := s.tokenService.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashRefreshToken(tokenPair.RefreshToken),
		ExpiresAt: time.Now().Add(s.tokenService.config.RefreshTTL),
	}

	err = s.repo.CreateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, req RefreshRequest) (*TokenResponse, error) {
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		return nil, errors.New("missing refresh token")
	}

	hash := HashRefreshToken(req.RefreshToken)
	token, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, err
	}

	// Check if revoked or expired
	if token.RevokedAt != nil || token.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("invalid refresh token")
	}

	// Revoke the old token (rotation)
	if err := s.repo.RevokeRefreshToken(ctx, hash); err != nil {
		return nil, err
	}

	// Generate new token pair
	tokenPair, err := s.tokenService.GenerateTokenPair(token.UserID)
	if err != nil {
		return nil, err
	}

	newRefreshToken := &RefreshToken{
		UserID:    token.UserID,
		TokenHash: HashRefreshToken(tokenPair.RefreshToken),
		ExpiresAt: time.Now().Add(s.tokenService.config.RefreshTTL),
	}

	if err := s.repo.CreateRefreshToken(ctx, newRefreshToken); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

func (s *authService) Logout(ctx context.Context, req LogoutRequest) error {
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		return errors.New("missing refresh token")
	}

	hash := HashRefreshToken(req.RefreshToken)
	err := s.repo.RevokeRefreshToken(ctx, hash)
	if err != nil && !errors.Is(err, ErrTokenNotFound) {
		return err
	}

	return nil
}

func (s *authService) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}
