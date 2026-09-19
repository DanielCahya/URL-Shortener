package url

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound            = errors.New("url not found")
	ErrExpired             = errors.New("url has expired or max accesses reached")
	ErrAliasAlreadyExists  = errors.New("custom alias already exists")
	ErrInvalidURL          = errors.New("invalid url format")
	ErrInvalidAlias        = errors.New("invalid custom alias")
	ErrReservedAlias       = errors.New("custom alias is a reserved system keyword")
	ErrExpirationInPast    = errors.New("expiration time must be in the future")
	ErrGenerationCollision = errors.New("failed to generate unique short code")
	ErrCacheMiss           = errors.New("cache miss")
	ErrPasswordRequired    = errors.New("password required to access this url")
	ErrUnauthorized        = errors.New("invalid password")
)

// URL represents the core URL entity.
type URL struct {
	ID           string     `json:"id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	ShortCode    string     `json:"short_code"`
	OriginalURL  string     `json:"original_url"`
	ExpiresAt    *time.Time `json:"expires_at"`
	MaxAccesses  *int       `json:"max_accesses"`
	AccessCount  int        `json:"access_count"`
	PasswordHash *string    `json:"-"`
	IsEnabled    bool       `json:"is_enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// CreateURLRequest represents the incoming payload to create a shortened URL.
type CreateURLRequest struct {
	OriginalURL string     `json:"original_url"`
	CustomAlias *string    `json:"custom_alias"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxAccesses *int       `json:"max_accesses"`
	Password    *string    `json:"password"`
}

// URLResponse represents the public API representation of a shortened URL.
type URLResponse struct {
	ID          string     `json:"id"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsEnabled   bool       `json:"is_enabled"`
	CreatedAt   time.Time  `json:"created_at"`
}

// APIErrorDetail holds the structured error fields.
type APIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIErrorResponse represents standard error payload across the API.
type APIErrorResponse struct {
	Error     APIErrorDetail `json:"error"`
	RequestID string         `json:"request_id"`
}
