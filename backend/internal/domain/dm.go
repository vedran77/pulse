package domain

import (
	"time"

	"github.com/google/uuid"
)

type DMConversation struct {
	ID        uuid.UUID `json:"id"`
	User1ID   uuid.UUID `json:"user1_id"`
	User2ID   uuid.UUID `json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`
	// Joined fields for frontend
	OtherUserID          uuid.UUID `json:"other_user_id"`
	OtherUserUsername    string    `json:"other_username"`
	OtherUserDisplayName string    `json:"other_display_name"`
}

type DMMessage struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	Content        *string    `json:"content,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	DeletedAt      *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	// Joined fields
	SenderUsername    string `json:"sender_username,omitempty"`
	SenderDisplayName string `json:"sender_display_name,omitempty"`
}
