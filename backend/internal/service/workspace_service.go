package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
	"github.com/vedran77/pulse/internal/repository"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrSlugTaken         = errors.New("workspace slug already taken")
	ErrNotWorkspaceOwner = errors.New("only workspace owner can perform this action")
	ErrNotMember         = errors.New("user is not a member of this workspace")
	ErrAlreadyMember     = errors.New("user is already a member")
)

type WorkspaceService struct {
	workspaceRepo repository.WorkspaceRepository
	userRepo      repository.UserRepository
}

func NewWorkspaceService(workspaceRepo repository.WorkspaceRepository, userRepo repository.UserRepository) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepo: workspaceRepo,
		userRepo:      userRepo,
	}
}

type CreateWorkspaceInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type UpdateWorkspaceInput struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

func (s *WorkspaceService) Create(ctx context.Context, userID uuid.UUID, input CreateWorkspaceInput) (*domain.Workspace, error) {
	slug := slugify(input.Slug)
	if slug == "" {
		slug = slugify(input.Name)
	}

	existing, err := s.workspaceRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugTaken
	}

	var desc *string
	if input.Description != "" {
		desc = &input.Description
	}

	ws := &domain.Workspace{
		ID:          uuid.New(),
		Name:        input.Name,
		Slug:        slug,
		Description: desc,
		OwnerID:     userID,
		CreatedAt:   time.Now(),
	}

	if err := s.workspaceRepo.Create(ctx, ws); err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	// Dodaj owner-a kao member sa ulogom "owner"
	member := &domain.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      userID,
		Role:        "owner",
		JoinedAt:    time.Now(),
	}
	if err := s.workspaceRepo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("adding owner as member: %w", err)
	}

	return ws, nil
}

func (s *WorkspaceService) GetByID(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.Workspace, error) {
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	ws, err := s.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrWorkspaceNotFound
	}

	return ws, nil
}

func (s *WorkspaceService) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	return s.workspaceRepo.ListByUser(ctx, userID)
}

func (s *WorkspaceService) Update(ctx context.Context, userID, workspaceID uuid.UUID, input UpdateWorkspaceInput) (*domain.Workspace, error) {
	ws, err := s.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrWorkspaceNotFound
	}
	if ws.OwnerID != userID {
		return nil, ErrNotWorkspaceOwner
	}

	if input.Name != nil {
		ws.Name = *input.Name
	}
	if input.Slug != nil {
		newSlug := slugify(*input.Slug)
		existing, err := s.workspaceRepo.GetBySlug(ctx, newSlug)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != ws.ID {
			return nil, ErrSlugTaken
		}
		ws.Slug = newSlug
	}
	if input.Description != nil {
		ws.Description = input.Description
	}

	if err := s.workspaceRepo.Update(ctx, ws); err != nil {
		return nil, fmt.Errorf("updating workspace: %w", err)
	}

	return ws, nil
}

func (s *WorkspaceService) Delete(ctx context.Context, userID, workspaceID uuid.UUID) error {
	ws, err := s.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return err
	}
	if ws == nil {
		return ErrWorkspaceNotFound
	}
	if ws.OwnerID != userID {
		return ErrNotWorkspaceOwner
	}

	return s.workspaceRepo.Delete(ctx, workspaceID)
}

func (s *WorkspaceService) AddMember(ctx context.Context, requesterID, workspaceID, userID uuid.UUID) error {
	// Provjeri da requester ima pristup
	requester, err := s.workspaceRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return err
	}
	if requester == nil || (requester.Role != "owner" && requester.Role != "admin") {
		return ErrNotWorkspaceOwner
	}

	// Provjeri da user postoji
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Provjeri da nije već member
	existing, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAlreadyMember
	}

	member := &domain.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        "member",
		JoinedAt:    time.Now(),
	}
	return s.workspaceRepo.AddMember(ctx, member)
}

func (s *WorkspaceService) RemoveMember(ctx context.Context, requesterID, workspaceID, userID uuid.UUID) error {
	requester, err := s.workspaceRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return err
	}
	if requester == nil || (requester.Role != "owner" && requester.Role != "admin") {
		return ErrNotWorkspaceOwner
	}

	return s.workspaceRepo.RemoveMember(ctx, workspaceID, userID)
}

func (s *WorkspaceService) ListMembers(ctx context.Context, userID, workspaceID uuid.UUID) ([]domain.WorkspaceMember, error) {
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	return s.workspaceRepo.ListMembers(ctx, workspaceID)
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]`)
var multiDash = regexp.MustCompile(`-{2,}`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
