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
	ErrMessageNotFound = errors.New("message not found")
	ErrNotMessageOwner = errors.New("only the message sender can perform this action")
)

// Notifier broadcasts real-time events to connected clients.
type Notifier interface {
	NotifyNewMessage(msg *domain.Message)
	NotifyEditedMessage(msg *domain.Message)
	NotifyDeletedMessage(channelID, messageID uuid.UUID)
	// DM notifications
	NotifyNewDM(msg *domain.DMMessage)
	NotifyEditedDM(msg *domain.DMMessage)
	NotifyDeletedDM(conversationID, messageID uuid.UUID)
}

type MessageService struct {
	messageRepo   repository.MessageRepository
	channelRepo   repository.ChannelRepository
	workspaceRepo repository.WorkspaceRepository
	notifier      Notifier
}

func NewMessageService(
	messageRepo repository.MessageRepository,
	channelRepo repository.ChannelRepository,
	workspaceRepo repository.WorkspaceRepository,
) *MessageService {
	return &MessageService{
		messageRepo:   messageRepo,
		channelRepo:   channelRepo,
		workspaceRepo: workspaceRepo,
	}
}

// SetNotifier sets the real-time notifier (optional dependency).
func (s *MessageService) SetNotifier(n Notifier) {
	s.notifier = n
}

type SendMessageInput struct {
	Content  string     `json:"content"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

type EditMessageInput struct {
	Content string `json:"content"`
}

type MessageListResponse struct {
	Messages []domain.Message `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

func (s *MessageService) Send(ctx context.Context, userID, channelID uuid.UUID, input SendMessageInput) (*domain.Message, error) {
	// Provjeri pristup kanalu
	if err := s.checkChannelAccess(ctx, userID, channelID); err != nil {
		return nil, err
	}

	content := input.Content
	msg := &domain.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		SenderID:  userID,
		Content:   &content,
		Type:      "text",
		ParentID:  input.ParentID,
		CreatedAt: time.Now(),
	}

	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}

	// Dohvati sa sender info
	full, err := s.messageRepo.GetByID(ctx, msg.ID)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyNewMessage(full)
	}

	return full, nil
}

func (s *MessageService) List(ctx context.Context, userID, channelID uuid.UUID, before *uuid.UUID, limit int) (*MessageListResponse, error) {
	if err := s.checkChannelAccess(ctx, userID, channelID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Dohvati limit+1 da znamo ima li jos
	messages, err := s.messageRepo.ListByChannel(ctx, channelID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[len(messages)-limit:] // zadrzi zadnjih "limit" (najnovije)
	}

	if messages == nil {
		messages = []domain.Message{}
	}

	return &MessageListResponse{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}

func (s *MessageService) Edit(ctx context.Context, userID, messageID uuid.UUID, input EditMessageInput) (*domain.Message, error) {
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, ErrMessageNotFound
	}
	if msg.SenderID != userID {
		return nil, ErrNotMessageOwner
	}

	msg.Content = &input.Content
	if err := s.messageRepo.Update(ctx, msg); err != nil {
		return nil, fmt.Errorf("updating message: %w", err)
	}

	updated, err := s.messageRepo.GetByID(ctx, msg.ID)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyEditedMessage(updated)
	}

	return updated, nil
}

func (s *MessageService) Delete(ctx context.Context, userID, messageID uuid.UUID) error {
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrMessageNotFound
	}
	if msg.SenderID != userID {
		return ErrNotMessageOwner
	}

	if err := s.messageRepo.SoftDelete(ctx, messageID); err != nil {
		return err
	}

	if s.notifier != nil {
		s.notifier.NotifyDeletedMessage(msg.ChannelID, messageID)
	}

	return nil
}

func (s *MessageService) checkChannelAccess(ctx context.Context, userID, channelID uuid.UUID) error {
	ch, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch == nil {
		return ErrChannelNotFound
	}

	// Za public kanale, workspace membership je dovoljan
	if ch.Type == "public" {
		member, err := s.workspaceRepo.GetMember(ctx, ch.WorkspaceID, userID)
		if err != nil {
			return err
		}
		if member == nil {
			return ErrNotMember
		}
		return nil
	}

	// Za private kanale, treba channel membership
	cm, err := s.channelRepo.GetMember(ctx, channelID, userID)
	if err != nil {
		return err
	}
	if cm == nil {
		return ErrNotChannelMember
	}
	return nil
}
