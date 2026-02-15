package domain

import (
	"time"

	"github.com/google/uuid"
)

type PulsemateRequest struct {
	ID         uuid.UUID `json:"id"`
	SenderID   uuid.UUID `json:"sender_id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	// Joined fields
	SenderUsername      string `json:"sender_username,omitempty"`
	SenderDisplayName   string `json:"sender_display_name,omitempty"`
	ReceiverUsername    string `json:"receiver_username,omitempty"`
	ReceiverDisplayName string `json:"receiver_display_name,omitempty"`
}

type Pulsemate struct {
	ID        uuid.UUID `json:"id"`
	User1ID   uuid.UUID `json:"user1_id"`
	User2ID   uuid.UUID `json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`
	// Joined fields
	OtherUserID          uuid.UUID `json:"other_user_id"`
	OtherUsername        string    `json:"other_username"`
	OtherDisplayName     string    `json:"other_display_name"`
	OtherStatus          string    `json:"other_status"`
}
