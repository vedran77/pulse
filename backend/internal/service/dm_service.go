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
	ErrDMConversationNotFound = errors.New("dm conversation not found")
	ErrDMNotParticipant       = errors.New("you are not a participant of this conversation")
	ErrDMMessageNotFound      = errors.New("dm message not found")
	ErrNotDMMessageOwner      = errors.New("only the message sender can perform this action")
	ErrCannotDMSelf           = errors.New("cannot start a conversation with yourself")
	ErrUserNotFound           = errors.New("user not found")
)

type DMService struct {
	dmRepo   repository.DMRepository
	userRepo repository.UserRepository
	notifier Notifier
}

func NewDMService(dmRepo repository.DMRepository, userRepo repository.UserRepository) *DMService {
	return &DMService{
		dmRepo:   dmRepo,
		userRepo: userRepo,
	}
}

func (s *DMService) SetNotifier(n Notifier) {
	s.notifier = n
}

type DMMessageListResponse struct {
	Messages []domain.DMMessage `json:"messages"`
	HasMore  bool               `json:"has_more"`
}

// GetOrCreateConversation finds or creates a DM conversation between two users.
func (s *DMService) GetOrCreateConversation(ctx context.Context, userID, otherUserID uuid.UUID) (*domain.DMConversation, error) {
	if userID == otherUserID {
		return nil, ErrCannotDMSelf
	}

	// Validate other user exists
	other, err := s.userRepo.GetByID(ctx, otherUserID)
	if err != nil {
		return nil, err
	}
	if other == nil {
		return nil, ErrUserNotFound
	}

	// Sort IDs so user1 < user2 (canonical order for CHECK constraint)
	u1, u2 := userID, otherUserID
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}

	// Check if exists
	conv, err := s.dmRepo.GetConversationByUsers(ctx, u1, u2)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		// Fill in other user info
		conv.OtherUserID = otherUserID
		conv.OtherUserUsername = other.Username
		conv.OtherUserDisplayName = other.DisplayName
		return conv, nil
	}

	// Create new
	conv = &domain.DMConversation{
		ID:        uuid.New(),
		User1ID:   u1,
		User2ID:   u2,
		CreatedAt: time.Now(),
		// Fill in other user info
		OtherUserID:          otherUserID,
		OtherUserUsername:    other.Username,
		OtherUserDisplayName: other.DisplayName,
	}

	if err := s.dmRepo.CreateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("creating dm conversation: %w", err)
	}

	return conv, nil
}

// ListConversations returns all DM conversations for a user.
func (s *DMService) ListConversations(ctx context.Context, userID uuid.UUID) ([]domain.DMConversation, error) {
	convs, err := s.dmRepo.ListConversations(ctx, userID)
	if err != nil {
		return nil, err
	}
	if convs == nil {
		convs = []domain.DMConversation{}
	}
	return convs, nil
}

// SendMessage sends a DM message.
func (s *DMService) SendMessage(ctx context.Context, userID, conversationID uuid.UUID, content string) (*domain.DMMessage, error) {
	if err := s.checkParticipant(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	msg := &domain.DMMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       userID,
		Content:        &content,
		CreatedAt:      time.Now(),
	}

	if err := s.dmRepo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("creating dm message: %w", err)
	}

	full, err := s.dmRepo.GetMessageByID(ctx, msg.ID)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyNewDM(full)
	}

	return full, nil
}

// ListMessages returns paginated DM messages.
func (s *DMService) ListMessages(ctx context.Context, userID, conversationID uuid.UUID, before *uuid.UUID, limit int) (*DMMessageListResponse, error) {
	if err := s.checkParticipant(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := s.dmRepo.ListMessages(ctx, conversationID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[len(messages)-limit:]
	}

	if messages == nil {
		messages = []domain.DMMessage{}
	}

	return &DMMessageListResponse{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}

// EditMessage edits a DM message.
func (s *DMService) EditMessage(ctx context.Context, userID, messageID uuid.UUID, content string) (*domain.DMMessage, error) {
	msg, err := s.dmRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, ErrDMMessageNotFound
	}
	if msg.SenderID != userID {
		return nil, ErrNotDMMessageOwner
	}

	msg.Content = &content
	if err := s.dmRepo.UpdateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("updating dm message: %w", err)
	}

	updated, err := s.dmRepo.GetMessageByID(ctx, msg.ID)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyEditedDM(updated)
	}

	return updated, nil
}

// DeleteMessage soft-deletes a DM message.
func (s *DMService) DeleteMessage(ctx context.Context, userID, messageID uuid.UUID) error {
	msg, err := s.dmRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrDMMessageNotFound
	}
	if msg.SenderID != userID {
		return ErrNotDMMessageOwner
	}

	if err := s.dmRepo.SoftDeleteMessage(ctx, messageID); err != nil {
		return err
	}

	if s.notifier != nil {
		s.notifier.NotifyDeletedDM(msg.ConversationID, messageID)
	}

	return nil
}

func (s *DMService) checkParticipant(ctx context.Context, userID, conversationID uuid.UUID) error {
	conv, err := s.dmRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return err
	}
	if conv == nil {
		return ErrDMConversationNotFound
	}
	if conv.User1ID != userID && conv.User2ID != userID {
		return ErrDMNotParticipant
	}
	return nil
}
