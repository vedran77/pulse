package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/service"
	"github.com/vedran77/pulse/internal/transport/http/middleware"
)

type DMHandler struct {
	dmService *service.DMService
}

func NewDMHandler(dmService *service.DMService) *DMHandler {
	return &DMHandler{dmService: dmService}
}

func (h *DMHandler) GetOrCreateConversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input struct {
		UserID uuid.UUID `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}
	if input.UserID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "MISSING_USER_ID", "user_id is required")
		return
	}

	conv, err := h.dmService.GetOrCreateConversation(r.Context(), userID, input.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCannotDMSelf):
			writeError(w, http.StatusBadRequest, "CANNOT_DM_SELF", "Cannot start a conversation with yourself")
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		default:
			log.Printf("ERROR get or create dm: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, conv)
}

func (h *DMHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	convs, err := h.dmService.ListConversations(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR list dm conversations: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	writeJSON(w, http.StatusOK, convs)
}

func (h *DMHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	convID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	var input struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}
	if input.Content == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CONTENT", "Message content is required")
		return
	}

	msg, err := h.dmService.SendMessage(r.Context(), userID, convID, input.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDMConversationNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Conversation not found")
		case errors.Is(err, service.ErrDMNotParticipant):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a participant of this conversation")
		default:
			log.Printf("ERROR send dm message: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *DMHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	convID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	var before *uuid.UUID
	if beforeStr := r.URL.Query().Get("before"); beforeStr != "" {
		id, err := uuid.Parse(beforeStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid before cursor")
			return
		}
		before = &id
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	resp, err := h.dmService.ListMessages(r.Context(), userID, convID, before, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDMConversationNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Conversation not found")
		case errors.Is(err, service.ErrDMNotParticipant):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a participant of this conversation")
		default:
			log.Printf("ERROR list dm messages: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *DMHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	messageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid message ID")
		return
	}

	var input struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}
	if input.Content == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CONTENT", "Message content is required")
		return
	}

	msg, err := h.dmService.EditMessage(r.Context(), userID, messageID, input.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDMMessageNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
		case errors.Is(err, service.ErrNotDMMessageOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You can only edit your own messages")
		default:
			log.Printf("ERROR edit dm message: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

func (h *DMHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	messageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid message ID")
		return
	}

	if err := h.dmService.DeleteMessage(r.Context(), userID, messageID); err != nil {
		switch {
		case errors.Is(err, service.ErrDMMessageNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
		case errors.Is(err, service.ErrNotDMMessageOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You can only delete your own messages")
		default:
			log.Printf("ERROR delete dm message: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
