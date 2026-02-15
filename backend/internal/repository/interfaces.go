package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *domain.Workspace) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Workspace, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error)
	Update(ctx context.Context, workspace *domain.Workspace) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, member *domain.WorkspaceMember) error
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error)
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.WorkspaceMember, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, channel *domain.Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Channel, error)
	Update(ctx context.Context, channel *domain.Channel) error
	Archive(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, member *domain.ChannelMember) error
	RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error
	GetMember(ctx context.Context, channelID, userID uuid.UUID) (*domain.ChannelMember, error)
	ListMembers(ctx context.Context, channelID uuid.UUID) ([]domain.ChannelMember, error)
}

type InviteRepository interface {
	Create(ctx context.Context, invite *domain.WorkspaceInvite) error
	GetByToken(ctx context.Context, token string) (*domain.WorkspaceInvite, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.WorkspaceInvite, error)
	MarkAccepted(ctx context.Context, id, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *domain.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error)
	ListByChannel(ctx context.Context, channelID uuid.UUID, before *uuid.UUID, limit int) ([]domain.Message, error)
	Update(ctx context.Context, msg *domain.Message) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type DMRepository interface {
	CreateConversation(ctx context.Context, conv *domain.DMConversation) error
	GetConversationByUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.DMConversation, error)
	GetConversationByID(ctx context.Context, id uuid.UUID) (*domain.DMConversation, error)
	ListConversations(ctx context.Context, userID uuid.UUID) ([]domain.DMConversation, error)
	CreateMessage(ctx context.Context, msg *domain.DMMessage) error
	GetMessageByID(ctx context.Context, id uuid.UUID) (*domain.DMMessage, error)
	ListMessages(ctx context.Context, conversationID uuid.UUID, before *uuid.UUID, limit int) ([]domain.DMMessage, error)
	UpdateMessage(ctx context.Context, msg *domain.DMMessage) error
	SoftDeleteMessage(ctx context.Context, id uuid.UUID) error
}
