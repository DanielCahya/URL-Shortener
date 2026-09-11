package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DanielCahya/url-shortener/internal/auth"
)

type mockAuthService struct {
	RegisterFunc func(ctx context.Context, req auth.RegisterRequest) (*auth.User, error)
	LoginFunc    func(ctx context.Context, req auth.LoginRequest) (*auth.TokenResponse, error)
	RefreshFunc  func(ctx context.Context, req auth.RefreshRequest) (*auth.TokenResponse, error)
	LogoutFunc   func(ctx context.Context, req auth.LogoutRequest) error
	GetProfileFunc func(ctx context.Context, userID uuid.UUID) (*auth.UserProfileResponse, error)
}

func (m *mockAuthService) Register(ctx context.Context, req auth.RegisterRequest) (*auth.User, error) {
	return m.RegisterFunc(ctx, req)
}

func (m *mockAuthService) Login(ctx context.Context, req auth.LoginRequest) (*auth.TokenResponse, error) {
	return m.LoginFunc(ctx, req)
}

func (m *mockAuthService) RefreshToken(ctx context.Context, req auth.RefreshRequest) (*auth.TokenResponse, error) {
	return m.RefreshFunc(ctx, req)
}

func (m *mockAuthService) Logout(ctx context.Context, req auth.LogoutRequest) error {
	return m.LogoutFunc(ctx, req)
}

func (m *mockAuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*auth.UserProfileResponse, error) {
	return m.GetProfileFunc(ctx, userID)
}

func TestAuthHandler_RegisterUser(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := auth.NewAuthHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		mockSvc.RegisterFunc = func(ctx context.Context, req auth.RegisterRequest) (*auth.User, error) {
			return &auth.User{Email: req.Email}, nil
		}

		body := []byte(`{"email": "test@example.com", "password": "securepassword"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		
		var user auth.User
		err := json.NewDecoder(rr.Body).Decode(&user)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer([]byte(`{invalid json`)))
		rr := httptest.NewRecorder()

		handler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
	
	t.Run("Service Error - Invalid Email", func(t *testing.T) {
		mockSvc.RegisterFunc = func(ctx context.Context, req auth.RegisterRequest) (*auth.User, error) {
			return nil, auth.ErrInvalidEmail
		}

		body := []byte(`{"email": "invalid", "password": "securepassword"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestAuthHandler_LoginUser(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := auth.NewAuthHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		mockSvc.LoginFunc = func(ctx context.Context, req auth.LoginRequest) (*auth.TokenResponse, error) {
			return &auth.TokenResponse{AccessToken: "access", RefreshToken: "refresh"}, nil
		}

		body := []byte(`{"email": "test@example.com", "password": "securepassword"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.LoginUser(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp auth.TokenResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		assert.Equal(t, "access", resp.AccessToken)
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		mockSvc.LoginFunc = func(ctx context.Context, req auth.LoginRequest) (*auth.TokenResponse, error) {
			return nil, errors.New("invalid email or password")
		}

		body := []byte(`{"email": "test@example.com", "password": "wrongpassword"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.LoginUser(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthHandler_RefreshTokens(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := auth.NewAuthHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		mockSvc.RefreshFunc = func(ctx context.Context, req auth.RefreshRequest) (*auth.TokenResponse, error) {
			return &auth.TokenResponse{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil
		}

		body := []byte(`{"refresh_token": "valid_refresh"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.RefreshTokens(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp auth.TokenResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		assert.Equal(t, "new-access", resp.AccessToken)
		assert.Equal(t, "new-refresh", resp.RefreshToken)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		mockSvc.RefreshFunc = func(ctx context.Context, req auth.RefreshRequest) (*auth.TokenResponse, error) {
			return nil, errors.New("invalid refresh token")
		}

		body := []byte(`{"refresh_token": "invalid_refresh"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.RefreshTokens(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthHandler_LogoutUser(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := auth.NewAuthHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		mockSvc.LogoutFunc = func(ctx context.Context, req auth.LogoutRequest) error {
			return nil
		}

		body := []byte(`{"refresh_token": "valid_refresh"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.LogoutUser(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("Missing Token", func(t *testing.T) {
		mockSvc.LogoutFunc = func(ctx context.Context, req auth.LogoutRequest) error {
			return errors.New("missing refresh token")
		}

		body := []byte(`{"refresh_token": ""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.LogoutUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestAuthHandler_GetProfile(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := auth.NewAuthHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		mockSvc.GetProfileFunc = func(ctx context.Context, uid uuid.UUID) (*auth.UserProfileResponse, error) {
			return &auth.UserProfileResponse{ID: uid, Email: "test@example.com"}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetProfile(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp auth.UserProfileResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		assert.Equal(t, "test@example.com", resp.Email)
	})

	t.Run("Unauthorized Context Missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rr := httptest.NewRecorder()
		handler.GetProfile(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
