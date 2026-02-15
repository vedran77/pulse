package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/service"
	"github.com/vedran77/pulse/internal/transport/http/middleware"
)

type PulsemateHandler struct {
	pmService *service.PulsemateService
}

func NewPulsemateHandler(pmService *service.PulsemateService) *PulsemateHandler {
	return &PulsemateHandler{pmService: pmService}
}

func (h *PulsemateHandler) SendRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}
	if input.Username == "" {
		writeError(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required")
		return
	}

	req, err := h.pmService.SendRequest(r.Context(), userID, input.Username)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCannotRequestSelf):
			writeError(w, http.StatusBadRequest, "CANNOT_REQUEST_SELF", "Cannot send a request to yourself")
		case errors.Is(err, service.ErrUserNotFoundForRequest):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		case errors.Is(err, service.ErrRequestAlreadyExists):
			writeError(w, http.StatusConflict, "ALREADY_EXISTS", "A pending request already exists")
		case errors.Is(err, service.ErrAlreadyPulsemates):
			writeError(w, http.StatusConflict, "ALREADY_PULSEMATES", "You are already pulsemates")
		default:
			log.Printf("ERROR send pulsemate request: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	if req == nil {
		// Auto-accepted
		writeJSON(w, http.StatusOK, map[string]string{"status": "auto_accepted"})
		return
	}

	writeJSON(w, http.StatusCreated, req)
}

func (h *PulsemateHandler) ListPulsemates(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	pms, err := h.pmService.ListPulsemates(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR list pulsemates: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	writeJSON(w, http.StatusOK, pms)
}

func (h *PulsemateHandler) ListIncomingRequests(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	reqs, err := h.pmService.ListIncomingRequests(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR list incoming requests: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	writeJSON(w, http.StatusOK, reqs)
}

func (h *PulsemateHandler) ListOutgoingRequests(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	reqs, err := h.pmService.ListOutgoingRequests(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR list outgoing requests: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	writeJSON(w, http.StatusOK, reqs)
}

func (h *PulsemateHandler) AcceptRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	requestID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid request ID")
		return
	}

	if err := h.pmService.AcceptRequest(r.Context(), userID, requestID); err != nil {
		switch {
		case errors.Is(err, service.ErrRequestNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Request not found")
		case errors.Is(err, service.ErrNotRequestReceiver):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the receiver can accept this request")
		default:
			log.Printf("ERROR accept pulsemate request: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PulsemateHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	requestID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid request ID")
		return
	}

	if err := h.pmService.RejectRequest(r.Context(), userID, requestID); err != nil {
		switch {
		case errors.Is(err, service.ErrRequestNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Request not found")
		case errors.Is(err, service.ErrNotRequestReceiver):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the receiver can reject this request")
		default:
			log.Printf("ERROR reject pulsemate request: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PulsemateHandler) CancelRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	requestID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid request ID")
		return
	}

	if err := h.pmService.CancelRequest(r.Context(), userID, requestID); err != nil {
		switch {
		case errors.Is(err, service.ErrRequestNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Request not found")
		case errors.Is(err, service.ErrNotRequestSender):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the sender can cancel this request")
		default:
			log.Printf("ERROR cancel pulsemate request: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PulsemateHandler) RemovePulsemate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	otherUserID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.pmService.RemovePulsemate(r.Context(), userID, otherUserID); err != nil {
		log.Printf("ERROR remove pulsemate: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
