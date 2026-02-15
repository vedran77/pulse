package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
	"github.com/vedran77/pulse/internal/service"
	"github.com/vedran77/pulse/internal/transport/http/middleware"
	"github.com/vedran77/pulse/pkg/validator"
)

type ChannelHandler struct {
	channelService *service.ChannelService
}

func NewChannelHandler(channelService *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{channelService: channelService}
}

func (h *ChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("wid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	var input service.CreateChannelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if errs := validator.ValidateChannel(input.Name, input.Type); errs.HasErrors() {
		writeValidationErrors(w, errs)
		return
	}

	ch, err := h.channelService.Create(r.Context(), userID, workspaceID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotMember):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
		case errors.Is(err, service.ErrChannelNameTaken):
			writeError(w, http.StatusConflict, "NAME_TAKEN", "Channel name already exists in this workspace")
		default:
			log.Printf("ERROR create channel: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusCreated, ch)
}

func (h *ChannelHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("wid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	channels, err := h.channelService.ListByWorkspace(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, service.ErrNotMember) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
		} else {
			log.Printf("ERROR list channels: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	if channels == nil {
		channels = []domain.Channel{}
	}

	writeJSON(w, http.StatusOK, channels)
}

func (h *ChannelHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	ch, err := h.channelService.GetByID(r.Context(), userID, channelID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrChannelNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
		case errors.Is(err, service.ErrNotMember), errors.Is(err, service.ErrNotChannelMember):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this channel")
		default:
			log.Printf("ERROR get channel: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *ChannelHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	var input service.UpdateChannelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	ch, err := h.channelService.Update(r.Context(), userID, channelID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrChannelNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
		case errors.Is(err, service.ErrNotChannelAdmin):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only channel admin can update it")
		case errors.Is(err, service.ErrChannelNameTaken):
			writeError(w, http.StatusConflict, "NAME_TAKEN", "Channel name already exists")
		default:
			log.Printf("ERROR update channel: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *ChannelHandler) Archive(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	if err := h.channelService.Archive(r.Context(), userID, channelID); err != nil {
		switch {
		case errors.Is(err, service.ErrChannelNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
		case errors.Is(err, service.ErrNotChannelAdmin):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only workspace owner or channel creator can archive")
		default:
			log.Printf("ERROR archive channel: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChannelHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	if err := h.channelService.AddMember(r.Context(), userID, channelID, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotMember):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, "ALREADY_MEMBER", "You are already a member of this channel")
		default:
			log.Printf("ERROR join channel: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChannelHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	targetID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.channelService.AddMember(r.Context(), requesterID, channelID, targetID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotChannelAdmin):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only channel admin can add members")
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, "ALREADY_MEMBER", "User is already a member")
		default:
			log.Printf("ERROR add channel member: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChannelHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	targetID, err := uuid.Parse(r.PathValue("uid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.channelService.RemoveMember(r.Context(), requesterID, channelID, targetID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotChannelAdmin):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only channel admin can remove members")
		default:
			log.Printf("ERROR remove channel member: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChannelHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	channelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid channel ID")
		return
	}

	members, err := h.channelService.ListMembers(r.Context(), userID, channelID)
	if err != nil {
		if errors.Is(err, service.ErrNotMember) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access")
		} else {
			log.Printf("ERROR list channel members: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, members)
}
