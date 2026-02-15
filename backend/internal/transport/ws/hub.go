package ws

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
)

// Hub manages all active WebSocket clients and routes messages.
type Hub struct {
	// clients maps userID → client.
	clients map[uuid.UUID]*Client

	register   chan *Client
	unregister chan *Client
	broadcast  chan *broadcastMsg
}

type broadcastMsg struct {
	channelID uuid.UUID
	data      []byte
	excludeID *uuid.UUID // optional: skip this user (e.g. sender)
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *broadcastMsg, 256),
	}
}

// Run starts the Hub's main event loop. Call this in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.userID] = client
			log.Printf("ws hub: user %s connected (%d total)", client.userID, len(h.clients))

			// Broadcast presence online
			h.broadcastPresence(client.userID, "online")

		case client := <-h.unregister:
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
				close(client.done)
				log.Printf("ws hub: user %s disconnected (%d total)", client.userID, len(h.clients))

				// Broadcast presence offline
				h.broadcastPresence(client.userID, "offline")
			}

		case msg := <-h.broadcast:
			for _, client := range h.clients {
				// Skip excluded user
				if msg.excludeID != nil && client.userID == *msg.excludeID {
					continue
				}
				// Only send to clients subscribed to this channel
				if !client.IsSubscribed(msg.channelID) {
					continue
				}
				select {
				case client.send <- msg.data:
				default:
					// Client buffer full - disconnect
					delete(h.clients, client.userID)
					close(client.send)
					close(client.done)
				}
			}
		}
	}
}

// BroadcastToChannel sends an event to all subscribers of a channel.
func (h *Hub) BroadcastToChannel(channelID uuid.UUID, event *Event, excludeUserID *uuid.UUID) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("ws hub: marshal error: %v", err)
		return
	}
	h.broadcast <- &broadcastMsg{
		channelID: channelID,
		data:      data,
		excludeID: excludeUserID,
	}
}

// BroadcastToUser sends an event directly to a specific user.
func (h *Hub) BroadcastToUser(userID uuid.UUID, event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	if client, ok := h.clients[userID]; ok {
		select {
		case client.send <- data:
		default:
		}
	}
}

// HandleTyping broadcasts typing events to channel subscribers (excluding sender).
func (h *Hub) HandleTyping(sender *Client, event *Event) {
	channelID := *event.ChannelID

	var eventType string
	if event.Type == EventTypeTypingStart {
		eventType = EventTypeTyping
	} else {
		return // typing.stop doesn't need broadcast, frontend uses timeout
	}

	evt, err := NewEvent(eventType, &channelID, TypingPayload{
		UserID:   sender.userID,
		Username: "", // Hub doesn't have user info - frontend can resolve from cache
	})
	if err != nil {
		return
	}

	h.BroadcastToChannel(channelID, evt, &sender.userID)
}

// broadcastPresence sends online/offline to all connected clients.
func (h *Hub) broadcastPresence(userID uuid.UUID, status string) {
	evt, err := NewEvent(EventTypePresence, nil, PresencePayload{
		UserID: userID,
		Status: status,
	})
	if err != nil {
		return
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	for _, client := range h.clients {
		if client.userID == userID {
			continue
		}
		select {
		case client.send <- data:
		default:
		}
	}
}
