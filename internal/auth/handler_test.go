package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DanielCahya/url-shortener/internal/auth"
)

type mockAuthService struct {
	RegisterFunc func(ctx context.Context, req auth.RegisterRequest) (*auth.User, error)
}

func (m *mockAuthService) Register(ctx context.Context, req auth.RegisterRequest) (*auth.User, error) {
	return m.RegisterFunc(ctx, req)
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
