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
