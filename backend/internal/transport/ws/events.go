package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
)

// Event types - Client → Server
const (
	EventTypeMessageSend       = "message.send"
	EventTypeTypingStart       = "typing.start"
	EventTypeTypingStop        = "typing.stop"
	EventTypeChannelSubscribe  = "channel.subscribe"
	EventTypeChannelUnsubscribe = "channel.unsubscribe"
	EventTypePing              = "ping"
)

// Event types - Server → Client
const (
	EventTypeMessageNew     = "message.new"
	EventTypeMessageEdited  = "message.edited"
	EventTypeMessageDeleted = "message.deleted"
	EventTypeTyping         = "typing"
	EventTypePresence       = "presence"
	EventTypePong           = "pong"
	EventTypeError          = "error"
)

// Event is the base envelope for all WebSocket messages.
type Event struct {
	Type      string          `json:"type"`
	ChannelID *uuid.UUID      `json:"channel_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"ts,omitempty"`
}

// --- Client → Server payloads ---

type MessageSendPayload struct {
	Content string `json:"content"`
	Nonce   string `json:"nonce,omitempty"`
}

type ChannelPayload struct {
	ChannelID uuid.UUID `json:"channel_id"`
}

// --- Server → Client payloads ---

type MessagePayload struct {
	domain.Message
}

type MessageDeletedPayload struct {
	ID uuid.UUID `json:"id"`
}

type TypingPayload struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
}

type PresencePayload struct {
	UserID uuid.UUID `json:"user_id"`
	Status string    `json:"status"` // "online" | "offline"
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewEvent creates a server→client event with the current timestamp.
func NewEvent(eventType string, channelID *uuid.UUID, payload any) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		Type:      eventType,
		ChannelID: channelID,
		Payload:   data,
		Timestamp: time.Now().Unix(),
	}, nil
}
