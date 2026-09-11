package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err := h.service.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidEmail) || errors.Is(err, ErrPasswordTooShort) {
			h.respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrUserAlreadyExists) {
			h.respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		
		// Internal Error
		h.respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	tokenResponse, err := h.service.Login(r.Context(), req)
	if err != nil {
		if err.Error() == "invalid email or password" {
			h.respondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		// Internal Error
		h.respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResponse)
}

func (h *AuthHandler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	tokenResponse, err := h.service.RefreshToken(r.Context(), req)
	if err != nil {
		if err.Error() == "invalid refresh token" || err.Error() == "missing refresh token" {
			h.respondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResponse)
}

func (h *AuthHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err := h.service.Logout(r.Context(), req)
	if err != nil {
		if err.Error() == "missing refresh token" {
			h.respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
