package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
	"github.com/vedran77/pulse/internal/repository"
)

var (
	ErrChannelNotFound  = errors.New("channel not found")
	ErrChannelNameTaken = errors.New("channel name already exists in this workspace")
	ErrNotChannelAdmin  = errors.New("only channel admin can perform this action")
	ErrNotChannelMember = errors.New("user is not a member of this channel")
)

type ChannelService struct {
	channelRepo   repository.ChannelRepository
	workspaceRepo repository.WorkspaceRepository
}

func NewChannelService(channelRepo repository.ChannelRepository, workspaceRepo repository.WorkspaceRepository) *ChannelService {
	return &ChannelService{
		channelRepo:   channelRepo,
		workspaceRepo: workspaceRepo,
	}
}

type CreateChannelInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type UpdateChannelInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (s *ChannelService) Create(ctx context.Context, userID, workspaceID uuid.UUID, input CreateChannelInput) (*domain.Channel, error) {
	// Provjeri da je user member workspace-a
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	chType := input.Type
	if chType == "" {
		chType = "public"
	}

	var desc *string
	if input.Description != "" {
		desc = &input.Description
	}

	ch := &domain.Channel{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        input.Name,
		Description: desc,
		Type:        chType,
		IsEncrypted: false,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}

	if err := s.channelRepo.Create(ctx, ch); err != nil {
		// Unique constraint violation = duplicate name
		if isDuplicateError(err) {
			return nil, ErrChannelNameTaken
		}
		return nil, fmt.Errorf("creating channel: %w", err)
	}

	// Dodaj creatora kao admin membera
	cm := &domain.ChannelMember{
		ChannelID: ch.ID,
		UserID:    userID,
		Role:      "admin",
		JoinedAt:  time.Now(),
	}
	if err := s.channelRepo.AddMember(ctx, cm); err != nil {
		return nil, fmt.Errorf("adding creator as member: %w", err)
	}

	return ch, nil
}

func (s *ChannelService) GetByID(ctx context.Context, userID, channelID uuid.UUID) (*domain.Channel, error) {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}

	// Public kanali su vidljivi svim memberima workspace-a
	if ch.Type == "public" {
		member, err := s.workspaceRepo.GetMember(ctx, ch.WorkspaceID, userID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, ErrNotMember
		}
	} else {
		// Private/DM kanali zahtijevaju channel membership
		cm, err := s.channelRepo.GetMember(ctx, channelID, userID)
		if err != nil {
			return nil, err
		}
		if cm == nil {
			return nil, ErrNotChannelMember
		}
	}

	return ch, nil
}

func (s *ChannelService) ListByWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) ([]domain.Channel, error) {
	// Provjeri membership
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	return s.channelRepo.ListByWorkspace(ctx, workspaceID)
}

func (s *ChannelService) Update(ctx context.Context, userID, channelID uuid.UUID, input UpdateChannelInput) (*domain.Channel, error) {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}

	// Samo admin ili creator
	cm, err := s.channelRepo.GetMember(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}
	if cm == nil || (cm.Role != "admin" && ch.CreatedBy != userID) {
		return nil, ErrNotChannelAdmin
	}

	if input.Name != nil {
		ch.Name = *input.Name
	}
	if input.Description != nil {
		ch.Description = input.Description
	}

	if err := s.channelRepo.Update(ctx, ch); err != nil {
		if isDuplicateError(err) {
			return nil, ErrChannelNameTaken
		}
		return nil, fmt.Errorf("updating channel: %w", err)
	}

	return ch, nil
}

func (s *ChannelService) Archive(ctx context.Context, userID, channelID uuid.UUID) error {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch == nil {
		return ErrChannelNotFound
	}

	// Samo workspace owner ili channel creator
	wsMember, err := s.workspaceRepo.GetMember(ctx, ch.WorkspaceID, userID)
	if err != nil {
		return err
	}
	if wsMember == nil || (wsMember.Role != "owner" && ch.CreatedBy != userID) {
		return ErrNotChannelAdmin
	}

	return s.channelRepo.Archive(ctx, channelID)
}

func (s *ChannelService) AddMember(ctx context.Context, requesterID, channelID, userID uuid.UUID) error {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch == nil {
		return ErrChannelNotFound
	}

	// Za public kanale, bilo koji workspace member može joinati
	if ch.Type == "public" {
		wsMember, err := s.workspaceRepo.GetMember(ctx, ch.WorkspaceID, userID)
		if err != nil {
			return err
		}
		if wsMember == nil {
			return ErrNotMember
		}
	} else {
		// Za private, samo admin može dodati
		cm, err := s.channelRepo.GetMember(ctx, channelID, requesterID)
		if err != nil {
			return err
		}
		if cm == nil || cm.Role != "admin" {
			return ErrNotChannelAdmin
		}
	}

	// Provjeri da nije već member
	existing, err := s.channelRepo.GetMember(ctx, channelID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAlreadyMember
	}

	member := &domain.ChannelMember{
		ChannelID: channelID,
		UserID:    userID,
		Role:      "member",
		JoinedAt:  time.Now(),
	}
	return s.channelRepo.AddMember(ctx, member)
}

func (s *ChannelService) RemoveMember(ctx context.Context, requesterID, channelID, userID uuid.UUID) error {
	cm, err := s.channelRepo.GetMember(ctx, channelID, requesterID)
	if err != nil {
		return err
	}
	if cm == nil || (cm.Role != "admin" && requesterID != userID) {
		return ErrNotChannelAdmin
	}

	return s.channelRepo.RemoveMember(ctx, channelID, userID)
}

func (s *ChannelService) ListMembers(ctx context.Context, userID, channelID uuid.UUID) ([]domain.ChannelMember, error) {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}

	// Provjeri pristup
	wsMember, err := s.workspaceRepo.GetMember(ctx, ch.WorkspaceID, userID)
	if err != nil {
		return nil, err
	}
	if wsMember == nil {
		return nil, ErrNotMember
	}

	return s.channelRepo.ListMembers(ctx, channelID)
}

// Helper za detekciju duplicate key errora iz pgx
func isDuplicateError(err error) bool {
	return err != nil && (errors.Is(err, errors.New("unique_violation")) ||
		// pgx vraća error string sa "SQLSTATE 23505"
		fmt.Sprintf("%v", err) != "" && contains(fmt.Sprintf("%v", err), "23505"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
