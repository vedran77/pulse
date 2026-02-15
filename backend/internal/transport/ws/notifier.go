package ws

import (
	"log"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
)

// HubNotifier implements service.Notifier using the WebSocket Hub.
type HubNotifier struct {
	hub *Hub
}

func NewHubNotifier(hub *Hub) *HubNotifier {
	return &HubNotifier{hub: hub}
}

func (n *HubNotifier) NotifyNewMessage(msg *domain.Message) {
	evt, err := NewEvent(EventTypeMessageNew, &msg.ChannelID, MessagePayload{Message: *msg})
	if err != nil {
		log.Printf("ws notifier: marshal error: %v", err)
		return
	}
	n.hub.BroadcastToChannel(msg.ChannelID, evt, nil)
}

func (n *HubNotifier) NotifyEditedMessage(msg *domain.Message) {
	evt, err := NewEvent(EventTypeMessageEdited, &msg.ChannelID, MessagePayload{Message: *msg})
	if err != nil {
		log.Printf("ws notifier: marshal error: %v", err)
		return
	}
	n.hub.BroadcastToChannel(msg.ChannelID, evt, nil)
}

func (n *HubNotifier) NotifyDeletedMessage(channelID, messageID uuid.UUID) {
	evt, err := NewEvent(EventTypeMessageDeleted, &channelID, MessageDeletedPayload{ID: messageID})
	if err != nil {
		log.Printf("ws notifier: marshal error: %v", err)
		return
	}
	n.hub.BroadcastToChannel(channelID, evt, nil)
}
