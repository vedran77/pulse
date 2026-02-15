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

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	var input service.SendMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if input.Content == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CONTENT", "Message content is required")
		return
	}

	msg, err := h.messageService.Send(r.Context(), userID, channelID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrChannelNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
		case errors.Is(err, service.ErrNotMember), errors.Is(err, service.ErrNotChannelMember):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this channel")
		default:
			log.Printf("ERROR send message: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	// Parse query params
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

	resp, err := h.messageService.List(r.Context(), userID, channelID, before, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrChannelNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
		case errors.Is(err, service.ErrNotMember), errors.Is(err, service.ErrNotChannelMember):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this channel")
		default:
			log.Printf("ERROR list messages: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	messageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid message ID")
		return
	}

	var input service.EditMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if input.Content == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CONTENT", "Message content is required")
		return
	}

	msg, err := h.messageService.Edit(r.Context(), userID, messageID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
		case errors.Is(err, service.ErrNotMessageOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You can only edit your own messages")
		default:
			log.Printf("ERROR edit message: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	messageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid message ID")
		return
	}

	if err := h.messageService.Delete(r.Context(), userID, messageID); err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
		case errors.Is(err, service.ErrNotMessageOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You can only delete your own messages")
		default:
			log.Printf("ERROR delete message: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
