import type { Message, DMMessage } from "./api";

// Event types matching backend
type EventType =
  | "message.new"
  | "message.edited"
  | "message.deleted"
  | "dm.new"
  | "dm.edited"
  | "dm.deleted"
  | "typing"
  | "presence"
  | "pong"
  | "error";

interface WSEvent {
  type: EventType;
  channel_id?: string;
  payload: unknown;
  ts?: number;
}

interface TypingPayload {
  user_id: string;
  username: string;
  display_name: string;
}

interface PresencePayload {
  user_id: string;
  status: "online" | "offline";
}

interface MessageDeletedPayload {
  id: string;
}

interface ErrorPayload {
  code: string;
  message: string;
}

export type WSCallbacks = {
  onMessage?: (msg: Message) => void;
  onMessageEdited?: (msg: Message) => void;
  onMessageDeleted?: (channelId: string, messageId: string) => void;
  onDMMessage?: (msg: DMMessage) => void;
  onDMMessageEdited?: (msg: DMMessage) => void;
  onDMMessageDeleted?: (conversationId: string, messageId: string) => void;
  onTyping?: (channelId: string, payload: TypingPayload) => void;
  onPresence?: (payload: PresencePayload) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (payload: ErrorPayload) => void;
};

const WS_BASE = "ws://localhost:8080/ws";
const MAX_RECONNECT_DELAY = 30000;

export class PulseWebSocket {
  private ws: WebSocket | null = null;
  private callbacks: WSCallbacks = {};
  private reconnectDelay = 1000;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldReconnect = true;
  private typingThrottles = new Map<string, number>();
  private subscribedChannels = new Set<string>();

  connect(callbacks: WSCallbacks) {
    // Close any existing connection first
    if (this.ws) {
      this.shouldReconnect = false;
      this.ws.close();
      this.ws = null;
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.callbacks = callbacks;
    this.shouldReconnect = true;
    this.doConnect();
  }

  private doConnect() {
    const token = localStorage.getItem("access_token");
    if (!token) return;

    this.ws = new WebSocket(`${WS_BASE}?token=${token}`);

    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      // Re-subscribe to any pending channels
      for (const chId of this.subscribedChannels) {
        this.send({
          type: "channel.subscribe",
          payload: { channel_id: chId },
        });
      }
      this.callbacks.onConnect?.();
    };

    this.ws.onclose = () => {
      this.callbacks.onDisconnect?.();
      if (this.shouldReconnect) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      // onclose will fire after this
    };

    this.ws.onmessage = (e) => {
      try {
        const event: WSEvent = JSON.parse(e.data);
        this.handleEvent(event);
      } catch {
        // ignore malformed messages
      }
    };
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.doConnect();
    }, this.reconnectDelay);
    this.reconnectDelay = Math.min(
      this.reconnectDelay * 2,
      MAX_RECONNECT_DELAY
    );
  }

  private handleEvent(event: WSEvent) {
    switch (event.type) {
      case "message.new":
        this.callbacks.onMessage?.(event.payload as Message);
        break;
      case "message.edited":
        this.callbacks.onMessageEdited?.(event.payload as Message);
        break;
      case "message.deleted": {
        const p = event.payload as MessageDeletedPayload;
        this.callbacks.onMessageDeleted?.(event.channel_id!, p.id);
        break;
      }
      case "dm.new":
        this.callbacks.onDMMessage?.(event.payload as DMMessage);
        break;
      case "dm.edited":
        this.callbacks.onDMMessageEdited?.(event.payload as DMMessage);
        break;
      case "dm.deleted": {
        const p = event.payload as MessageDeletedPayload;
        this.callbacks.onDMMessageDeleted?.(event.channel_id!, p.id);
        break;
      }
      case "typing":
        this.callbacks.onTyping?.(
          event.channel_id!,
          event.payload as TypingPayload
        );
        break;
      case "presence":
        this.callbacks.onPresence?.(event.payload as PresencePayload);
        break;
      case "error":
        this.callbacks.onError?.(event.payload as ErrorPayload);
        break;
      case "pong":
        break;
    }
  }

  subscribe(channelId: string) {
    this.subscribedChannels.add(channelId);
    this.send({
      type: "channel.subscribe",
      payload: { channel_id: channelId },
    });
  }

  unsubscribe(channelId: string) {
    this.subscribedChannels.delete(channelId);
    this.send({
      type: "channel.unsubscribe",
      payload: { channel_id: channelId },
    });
  }

  sendTyping(channelId: string) {
    const now = Date.now();
    const lastSent = this.typingThrottles.get(channelId) ?? 0;
    if (now - lastSent < 3000) return;

    this.typingThrottles.set(channelId, now);
    this.send({
      type: "typing.start",
      channel_id: channelId,
    });
  }

  private send(data: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  disconnect() {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.typingThrottles.clear();
    this.subscribedChannels.clear();
    this.ws?.close();
    this.ws = null;
  }
}

// Singleton instance
export const pulseWS = new PulseWebSocket();
