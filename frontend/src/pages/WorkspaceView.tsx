import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api, type Workspace, type Channel, type Message } from "../lib/api";
import { pulseWS } from "../lib/ws";

export default function WorkspaceView() {
  const { workspaceId } = useParams();
  const navigate = useNavigate();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [activeChannel, setActiveChannel] = useState<Channel | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [messageInput, setMessageInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [newChannelName, setNewChannelName] = useState("");
  const [typingUsers, setTypingUsers] = useState<Map<string, string>>(
    new Map()
  );
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const activeChannelRef = useRef<Channel | null>(null);
  const typingTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map()
  );
  const currentUser = JSON.parse(localStorage.getItem("user") || "{}");

  // Keep ref in sync for WS callbacks
  useEffect(() => {
    activeChannelRef.current = activeChannel;
  }, [activeChannel]);

  // Load workspace + channels
  useEffect(() => {
    if (!workspaceId) return;

    Promise.all([api.getWorkspace(workspaceId), api.listChannels(workspaceId)])
      .then(([ws, chs]) => {
        setWorkspace(ws);
        setChannels(chs);
        if (chs.length > 0) setActiveChannel(chs[0]);
      })
      .catch(() => navigate("/"))
      .finally(() => setLoading(false));
  }, [workspaceId, navigate]);

  // Load messages when channel changes
  useEffect(() => {
    if (!activeChannel) return;
    setMessages([]);
    setTypingUsers(new Map());

    api
      .listMessages(activeChannel.id)
      .then((res) => setMessages(res.messages))
      .catch(() => {});
  }, [activeChannel]);

  // Scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // WS callbacks
  const handleWSMessage = useCallback((msg: Message) => {
    const current = activeChannelRef.current;
    if (!current || msg.channel_id !== current.id) return;
    setMessages((prev) => {
      // Avoid duplicates (e.g. from optimistic update)
      if (prev.some((m) => m.id === msg.id)) return prev;
      return [...prev, msg];
    });
  }, []);

  const handleWSMessageEdited = useCallback((msg: Message) => {
    const current = activeChannelRef.current;
    if (!current || msg.channel_id !== current.id) return;
    setMessages((prev) => prev.map((m) => (m.id === msg.id ? msg : m)));
  }, []);

  const handleWSMessageDeleted = useCallback(
    (channelId: string, messageId: string) => {
      const current = activeChannelRef.current;
      if (!current || channelId !== current.id) return;
      setMessages((prev) => prev.filter((m) => m.id !== messageId));
    },
    []
  );

  const handleWSTyping = useCallback(
    (channelId: string, payload: { user_id: string; display_name: string }) => {
      const current = activeChannelRef.current;
      if (!current || channelId !== current.id) return;
      if (payload.user_id === currentUser.id) return;

      const name = payload.display_name || payload.user_id.slice(0, 8);

      setTypingUsers((prev) => {
        const next = new Map(prev);
        next.set(payload.user_id, name);
        return next;
      });

      // Clear after 3s
      const existing = typingTimers.current.get(payload.user_id);
      if (existing) clearTimeout(existing);
      typingTimers.current.set(
        payload.user_id,
        setTimeout(() => {
          setTypingUsers((prev) => {
            const next = new Map(prev);
            next.delete(payload.user_id);
            return next;
          });
          typingTimers.current.delete(payload.user_id);
        }, 3000)
      );
    },
    [currentUser.id]
  );

  // Connect WebSocket
  useEffect(() => {
    pulseWS.connect({
      onMessage: handleWSMessage,
      onMessageEdited: handleWSMessageEdited,
      onMessageDeleted: handleWSMessageDeleted,
      onTyping: handleWSTyping,
    });

    return () => {
      pulseWS.disconnect();
    };
  }, [handleWSMessage, handleWSMessageEdited, handleWSMessageDeleted, handleWSTyping]);

  // Subscribe/unsubscribe on channel change
  useEffect(() => {
    if (!activeChannel) return;

    pulseWS.subscribe(activeChannel.id);

    return () => {
      pulseWS.unsubscribe(activeChannel.id);
    };
  }, [activeChannel]);

  async function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (!activeChannel || !messageInput.trim() || sending) return;

    setSending(true);
    try {
      const msg = await api.sendMessage(activeChannel.id, messageInput.trim());
      setMessages((prev) => {
        if (prev.some((m) => m.id === msg.id)) return prev;
        return [...prev, msg];
      });
      setMessageInput("");
    } catch {
      // TODO: show error
    } finally {
      setSending(false);
    }
  }

  async function handleCreateChannel(e: React.FormEvent) {
    e.preventDefault();
    if (!workspaceId || !newChannelName.trim()) return;

    try {
      const ch = await api.createChannel(workspaceId, {
        name: newChannelName.toLowerCase().replace(/\s+/g, "-"),
        type: "public",
      });
      setChannels([...channels, ch]);
      setActiveChannel(ch);
      setNewChannelName("");
      setShowCreateChannel(false);
    } catch {
      // TODO: show error
    }
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    setMessageInput(e.target.value);
    if (activeChannel && e.target.value.trim()) {
      pulseWS.sendTyping(activeChannel.id);
    }
  }

  function formatTime(dateStr: string) {
    return new Date(dateStr).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  const typingText =
    typingUsers.size > 0
      ? Array.from(typingUsers.values()).join(", ") + " is typing..."
      : null;

  if (loading) {
    return <div className="workspace-loading">Loading...</div>;
  }

  return (
    <div className="workspace-layout">
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="sidebar-header">
          <button
            className="sidebar-workspace-btn"
            onClick={() => navigate("/")}
          >
            <span className="sidebar-workspace-avatar">
              {workspace?.name.charAt(0).toUpperCase()}
            </span>
            <span className="sidebar-workspace-name">{workspace?.name}</span>
          </button>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-header">
            <span>Channels</span>
            <button
              className="sidebar-add-btn"
              onClick={() => setShowCreateChannel(!showCreateChannel)}
            >
              +
            </button>
          </div>

          {showCreateChannel && (
            <form
              onSubmit={handleCreateChannel}
              className="sidebar-create-form"
            >
              <input
                type="text"
                value={newChannelName}
                onChange={(e) => setNewChannelName(e.target.value)}
                placeholder="channel-name"
                autoFocus
              />
            </form>
          )}

          <div className="channel-list">
            {channels.map((ch) => (
              <button
                key={ch.id}
                className={`channel-item ${activeChannel?.id === ch.id ? "active" : ""}`}
                onClick={() => setActiveChannel(ch)}
              >
                <span className="channel-hash">#</span>
                {ch.name}
              </button>
            ))}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="main-content">
        {activeChannel ? (
          <>
            <div className="main-header">
              <h2>
                <span className="channel-hash">#</span> {activeChannel.name}
              </h2>
              {activeChannel.description && (
                <p className="main-header-desc">{activeChannel.description}</p>
              )}
            </div>

            <div className="messages-area">
              {messages.length === 0 ? (
                <div className="messages-empty">
                  <p>No messages yet in #{activeChannel.name}</p>
                  <p className="messages-empty-sub">
                    Be the first to send a message!
                  </p>
                </div>
              ) : (
                <div className="messages-list">
                  {messages.map((msg) => (
                    <div
                      key={msg.id}
                      className={`message ${msg.sender_id === currentUser.id ? "message-own" : ""}`}
                    >
                      <div className="message-avatar">
                        {msg.sender_display_name?.charAt(0).toUpperCase()}
                      </div>
                      <div className="message-body">
                        <div className="message-meta">
                          <span className="message-sender">
                            {msg.sender_display_name}
                          </span>
                          <span className="message-time">
                            {formatTime(msg.created_at)}
                          </span>
                          {msg.edited_at && (
                            <span className="message-edited">(edited)</span>
                          )}
                        </div>
                        <p className="message-text">{msg.content}</p>
                      </div>
                    </div>
                  ))}
                  <div ref={messagesEndRef} />
                </div>
              )}
            </div>

            {typingText && (
              <div className="typing-indicator">{typingText}</div>
            )}

            <form onSubmit={handleSend} className="message-input-area">
              <input
                type="text"
                className="message-input"
                placeholder={`Message #${activeChannel.name}`}
                value={messageInput}
                onChange={handleInputChange}
                disabled={sending}
              />
            </form>
          </>
        ) : (
          <div className="no-channel">
            <p>Select a channel to start chatting</p>
          </div>
        )}
      </main>
    </div>
  );
}
