package auth

import (
	"context"
	"strings"
)

var (
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*User, error)
	Login(ctx context.Context, req LoginRequest) (*TokenResponse, error)
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
