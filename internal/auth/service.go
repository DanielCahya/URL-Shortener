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
	// We'll add Login here in later steps
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
