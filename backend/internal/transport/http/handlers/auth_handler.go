package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/vedran77/pulse/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if input.Email == "" || input.Username == "" || input.Password == "" || input.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "All fields are required")
		return
	}

	if len(input.Password) < 8 {
		writeError(w, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters")
		return
	}

	resp, err := h.authService.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "EMAIL_TAKEN", "Email is already registered")
		case errors.Is(err, service.ErrUsernameTaken):
			writeError(w, http.StatusConflict, "USERNAME_TAKEN", "Username is already taken")
		default:
			log.Printf("ERROR register: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "Email and password are required")
		return
	}

	resp, err := h.authService.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		} else {
			log.Printf("ERROR login: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
