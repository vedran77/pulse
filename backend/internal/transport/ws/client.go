package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = 30 * time.Second
	maxMessageSize = 4096
	sendBufSize    = 256
)

// Client represents a single WebSocket connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID uuid.UUID

	// subscribedChannels tracks which channels this client listens to.
	subscribedChannels map[uuid.UUID]struct{}
	mu                 sync.RWMutex

	send chan []byte
	done chan struct{}
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID) *Client {
	return &Client{
		hub:                hub,
		conn:               conn,
		userID:             userID,
		subscribedChannels: make(map[uuid.UUID]struct{}),
		send:               make(chan []byte, sendBufSize),
		done:               make(chan struct{}),
	}
}

// IsSubscribed checks if this client is subscribed to a channel.
func (c *Client) IsSubscribed(channelID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subscribedChannels[channelID]
	return ok
}

// Subscribe adds a channel subscription.
func (c *Client) Subscribe(channelID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribedChannels[channelID] = struct{}{}
}

// Unsubscribe removes a channel subscription.
func (c *Client) Unsubscribe(channelID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribedChannels, channelID)
}

// ReadPump reads messages from the WebSocket and routes them to the Hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var event Event
		err := wsjson.Read(context.Background(), c.conn, &event)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Printf("ws: client %s disconnected", c.userID)
			} else {
				log.Printf("ws: read error from %s: %v", c.userID, err)
			}
			return
		}

		c.handleEvent(&event)
	}
}

// WritePump writes messages from the send channel to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Write(ctx, websocket.MessageText, message)
			cancel()
			if err != nil {
				log.Printf("ws: write error to %s: %v", c.userID, err)
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				log.Printf("ws: ping error to %s: %v", c.userID, err)
				return
			}

		case <-c.done:
			return
		}
	}
}

// handleEvent routes an incoming client event.
func (c *Client) handleEvent(event *Event) {
	switch event.Type {
	case EventTypeChannelSubscribe:
		var p ChannelPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			c.sendError("INVALID_PAYLOAD", "invalid channel_subscribe payload")
			return
		}
		c.Subscribe(p.ChannelID)
		log.Printf("ws: %s subscribed to channel %s", c.userID, p.ChannelID)

	case EventTypeChannelUnsubscribe:
		var p ChannelPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			c.sendError("INVALID_PAYLOAD", "invalid channel_unsubscribe payload")
			return
		}
		c.Unsubscribe(p.ChannelID)
		log.Printf("ws: %s unsubscribed from channel %s", c.userID, p.ChannelID)

	case EventTypeTypingStart, EventTypeTypingStop:
		if event.ChannelID == nil {
			c.sendError("INVALID_PAYLOAD", "channel_id required for typing events")
			return
		}
		c.hub.HandleTyping(c, event)

	case EventTypePing:
		c.sendPong()

	default:
		c.sendError("UNKNOWN_EVENT", "unknown event type: "+event.Type)
	}
}

func (c *Client) sendPong() {
	data, _ := json.Marshal(Event{Type: EventTypePong})
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) sendError(code, message string) {
	evt, err := NewEvent(EventTypeError, nil, ErrorPayload{Code: code, Message: message})
	if err != nil {
		return
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}
