package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID               uuid.UUID  `json:"id"`
	ChannelID        uuid.UUID  `json:"channel_id"`
	SenderID         uuid.UUID  `json:"sender_id"`
	Content          *string    `json:"content,omitempty"`
	ContentEncrypted []byte     `json:"-"`
	Nonce            []byte     `json:"-"`
	Type             string     `json:"type"`
	ParentID         *uuid.UUID `json:"parent_id,omitempty"`
	EditedAt         *time.Time `json:"edited_at,omitempty"`
	DeletedAt        *time.Time `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	// Joined fields
	SenderUsername    string `json:"sender_username,omitempty"`
	SenderDisplayName string `json:"sender_display_name,omitempty"`
}
