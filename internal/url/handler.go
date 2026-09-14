package url

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/logger"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service Service
}

// NewHandler creates a new HTTP handler for URLs.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /api/v1/urls.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid JSON payload")
		return
	}

	res, err := h.service.CreateURL(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidURL):
			h.writeError(w, r, http.StatusBadRequest, "INVALID_URL", "Original URL must be a valid HTTP/HTTPS URL")
		case errors.Is(err, ErrInvalidAlias):
			h.writeError(w, r, http.StatusBadRequest, "INVALID_ALIAS", "Custom alias must be 3-64 characters and contain only letters, numbers, hyphens, and underscores")
		case errors.Is(err, ErrReservedAlias):
			h.writeError(w, r, http.StatusBadRequest, "RESERVED_ALIAS", "The specified alias is a reserved system keyword")
		case errors.Is(err, ErrExpirationInPast):
			h.writeError(w, r, http.StatusBadRequest, "INVALID_EXPIRATION", "Expiration date must be in the future")
		case errors.Is(err, ErrAliasAlreadyExists):
			h.writeError(w, r, http.StatusConflict, "ALIAS_ALREADY_EXISTS", "The custom alias is already in use")
		case errors.Is(err, ErrGenerationCollision):
			h.writeError(w, r, http.StatusInternalServerError, "GENERATION_FAILED", "Failed to generate unique short code. Please retry")
		default:
			h.writeError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An internal error occurred")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

// Redirect handles GET /{short_code}.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "short_code")
	if shortCode == "" {
		h.writeError(w, r, http.StatusBadRequest, "INVALID_SHORT_CODE", "Short code is required")
		return
	}

	req := ResolveRequest{
		ShortCode: shortCode,
		UserAgent: r.UserAgent(),
		IPCountry: r.Header.Get("CF-IPCountry"),
		Referer:   r.Referer(),
	}

	targetURL, err := h.service.ResolveURL(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			h.writeError(w, r, http.StatusNotFound, "URL_NOT_FOUND", "The requested URL does not exist")
		case errors.Is(err, ErrExpired):
			h.writeError(w, r, http.StatusGone, "URL_EXPIRED", "The requested URL has expired")
		default:
			h.writeError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An internal error occurred")
		}
		return
	}

	// 307 Temporary Redirect preserves HTTP method and indicates temporary location
	http.Redirect(w, r, targetURL, http.StatusTemporaryRedirect)
}

// Delete handles requests to delete a short URL.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "short_code")
	if shortCode == "" {
		h.writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Short code is required")
		return
	}

	err := h.service.DeleteURL(r.Context(), shortCode)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			h.writeError(w, r, http.StatusNotFound, "URL_NOT_FOUND", "The requested URL does not exist or you do not have permission to delete it")
		case errors.Is(err, auth.ErrUnauthorized):
			h.writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "You must be logged in to delete URLs")
		default:
			h.writeError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An internal error occurred")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := APIErrorResponse{
		Error: APIErrorDetail{
			Code:    code,
			Message: message,
		},
		RequestID: logger.GetRequestID(r.Context()),
	}

	_ = json.NewEncoder(w).Encode(resp)
}
