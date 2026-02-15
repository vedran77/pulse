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

type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
}

func NewWorkspaceHandler(workspaceService *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: workspaceService}
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.CreateWorkspaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if errs := validator.ValidateWorkspace(input.Name, input.Slug); errs.HasErrors() {
		writeValidationErrors(w, errs)
		return
	}

	ws, err := h.workspaceService.Create(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, service.ErrSlugTaken) {
			writeError(w, http.StatusConflict, "SLUG_TAKEN", "Workspace slug is already taken")
		} else {
			log.Printf("ERROR create workspace: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusCreated, ws)
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	workspaces, err := h.workspaceService.ListByUser(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR list workspaces: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	if workspaces == nil {
		workspaces = []domain.Workspace{}
	}

	writeJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	ws, err := h.workspaceService.GetByID(r.Context(), userID, workspaceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWorkspaceNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Workspace not found")
		case errors.Is(err, service.ErrNotMember):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
		default:
			log.Printf("ERROR get workspace: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, ws)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	var input service.UpdateWorkspaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	ws, err := h.workspaceService.Update(r.Context(), userID, workspaceID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWorkspaceNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Workspace not found")
		case errors.Is(err, service.ErrNotWorkspaceOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the workspace owner can update it")
		case errors.Is(err, service.ErrSlugTaken):
			writeError(w, http.StatusConflict, "SLUG_TAKEN", "Workspace slug is already taken")
		default:
			log.Printf("ERROR update workspace: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, ws)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	if err := h.workspaceService.Delete(r.Context(), userID, workspaceID); err != nil {
		switch {
		case errors.Is(err, service.ErrWorkspaceNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Workspace not found")
		case errors.Is(err, service.ErrNotWorkspaceOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the workspace owner can delete it")
		default:
			log.Printf("ERROR delete workspace: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.workspaceService.AddMember(r.Context(), requesterID, workspaceID, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotWorkspaceOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only owner or admin can add members")
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, "ALREADY_MEMBER", "User is already a member")
		default:
			log.Printf("ERROR add member: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	userID, err := uuid.Parse(r.PathValue("uid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.workspaceService.RemoveMember(r.Context(), requesterID, workspaceID, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotWorkspaceOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only owner or admin can remove members")
		default:
			log.Printf("ERROR remove member: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}
	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email is required")
		return
	}

	invite, err := h.workspaceService.CreateInvite(r.Context(), requesterID, workspaceID, body.Email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotWorkspaceOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only owner or admin can create invites")
		default:
			log.Printf("ERROR create invite: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"invite": invite,
		"link":   "/invite/" + invite.Token,
	})
}

func (h *WorkspaceHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	invites, err := h.workspaceService.ListInvites(r.Context(), requesterID, workspaceID)
	if err != nil {
		if errors.Is(err, service.ErrNotMember) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
		} else {
			log.Printf("ERROR list invites: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	if invites == nil {
		invites = []domain.WorkspaceInvite{}
	}

	writeJSON(w, http.StatusOK, invites)
}

func (h *WorkspaceHandler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	inviteID, err := uuid.Parse(r.PathValue("inviteId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid invite ID")
		return
	}

	if err := h.workspaceService.RevokeInvite(r.Context(), requesterID, workspaceID, inviteID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotWorkspaceOwner):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Only owner or admin can revoke invites")
		default:
			log.Printf("ERROR revoke invite: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) GetInviteInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "Token is required")
		return
	}

	invite, err := h.workspaceService.GetInviteInfo(r.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrInviteNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Invite not found")
		} else {
			log.Printf("ERROR get invite info: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_name": invite.WorkspaceName,
		"email":          invite.Email,
		"expires_at":     invite.ExpiresAt,
		"accepted":       invite.AcceptedAt != nil,
	})
}

func (h *WorkspaceHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	token := r.PathValue("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "Token is required")
		return
	}

	invite, err := h.workspaceService.AcceptInvite(r.Context(), userID, token)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInviteNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Invite not found")
		case errors.Is(err, service.ErrInviteExpired):
			writeError(w, http.StatusGone, "EXPIRED", "Invite has expired")
		case errors.Is(err, service.ErrInviteUsed):
			writeError(w, http.StatusConflict, "ALREADY_USED", "Invite has already been used")
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, "ALREADY_MEMBER", "You are already a member of this workspace")
		default:
			log.Printf("ERROR accept invite: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id": invite.WorkspaceID,
	})
}

func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	workspaceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid workspace ID")
		return
	}

	members, err := h.workspaceService.ListMembers(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, service.ErrNotMember) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
		} else {
			log.Printf("ERROR list members: %v", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, members)
}
