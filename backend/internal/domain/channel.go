package domain

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Type        string     `json:"type"`
	IsEncrypted bool       `json:"is_encrypted"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

type ChannelMember struct {
	ChannelID    uuid.UUID  `json:"channel_id"`
	UserID       uuid.UUID  `json:"user_id"`
	Role         string     `json:"role"`
	EncryptedKey []byte     `json:"-"`
	LastReadMsgID *uuid.UUID `json:"last_read_msg_id,omitempty"`
	JoinedAt     time.Time  `json:"joined_at"`
}
