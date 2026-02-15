import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  api,
  type Workspace,
  type Channel,
  type Message,
  type DMConversation,
  type DMMessage,
} from "../lib/api";
import { pulseWS } from "../lib/ws";

type ActiveView =
  | { type: "channel"; channel: Channel }
  | { type: "dm"; conversation: DMConversation };

export default function WorkspaceView() {
  const { workspaceId } = useParams();
  const navigate = useNavigate();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [activeView, setActiveView] = useState<ActiveView | null>(null);
  const [messages, setMessages] = useState<(Message | DMMessage)[]>([]);
  const [messageInput, setMessageInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [newChannelName, setNewChannelName] = useState("");
  const [showInvite, setShowInvite] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [inviteError, setInviteError] = useState("");
  const [typingUsers, setTypingUsers] = useState<Map<string, string>>(
    new Map()
  );
  const [dmConversations, setDmConversations] = useState<DMConversation[]>([]);
  const [showNewDM, setShowNewDM] = useState(false);
  const [newDMUsername, setNewDMUsername] = useState("");
  const [dmError, setDmError] = useState("");
  const [unreadDMs, setUnreadDMs] = useState<Set<string>>(new Set());

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const activeViewRef = useRef<ActiveView | null>(null);
  const typingTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map()
  );
  const [currentUser] = useState(() =>
    JSON.parse(localStorage.getItem("user") || "{}")
  );

  // Keep ref in sync for WS callbacks
  useEffect(() => {
    activeViewRef.current = activeView;
  }, [activeView]);

  // Load workspace + channels + DM conversations
  useEffect(() => {
    if (!workspaceId) return;

    Promise.all([
      api.getWorkspace(workspaceId),
      api.listChannels(workspaceId),
      api.listDMConversations(),
    ])
      .then(([ws, chs, dms]) => {
        setWorkspace(ws);
        setChannels(chs);
        setDmConversations(dms);
        if (chs.length > 0) setActiveView({ type: "channel", channel: chs[0] });
      })
      .catch(() => navigate("/"))
      .finally(() => setLoading(false));
  }, [workspaceId, navigate]);

  // Load messages when active view changes
  useEffect(() => {
    if (!activeView) return;
    setMessages([]);
    setTypingUsers(new Map());

    if (activeView.type === "channel") {
      api
        .listMessages(activeView.channel.id)
        .then((res) => setMessages(res.messages))
        .catch(() => {});
    } else {
      api
        .listDMMessages(activeView.conversation.id)
        .then((res) => setMessages(res.messages))
        .catch(() => {});
      // Clear unread for this conversation
      setUnreadDMs((prev) => {
        const next = new Set(prev);
        next.delete(activeView.conversation.id);
        return next;
      });
    }
  }, [activeView]);

  // Scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // --- WS callbacks for channel messages ---
  const handleWSMessage = useCallback((msg: Message) => {
    const current = activeViewRef.current;
    if (!current || current.type !== "channel" || msg.channel_id !== current.channel.id)
      return;
    setMessages((prev) => {
      if (prev.some((m) => m.id === msg.id)) return prev;
      return [...prev, msg];
    });
  }, []);

  const handleWSMessageEdited = useCallback((msg: Message) => {
    const current = activeViewRef.current;
    if (!current || current.type !== "channel" || msg.channel_id !== current.channel.id)
      return;
    setMessages((prev) => prev.map((m) => (m.id === msg.id ? msg : m)));
  }, []);

  const handleWSMessageDeleted = useCallback(
    (channelId: string, messageId: string) => {
      const current = activeViewRef.current;
      if (!current || current.type !== "channel" || channelId !== current.channel.id)
        return;
      setMessages((prev) => prev.filter((m) => m.id !== messageId));
    },
    []
  );

  // --- WS callbacks for DM messages ---
  const handleWSDMMessage = useCallback((msg: DMMessage) => {
    const current = activeViewRef.current;
    if (
      current?.type === "dm" &&
      msg.conversation_id === current.conversation.id
    ) {
      setMessages((prev) => {
        if (prev.some((m) => m.id === msg.id)) return prev;
        return [...prev, msg];
      });
    } else {
      // Mark as unread if not viewing this conversation
      setUnreadDMs((prev) => {
        const next = new Set(prev);
        next.add(msg.conversation_id);
        return next;
      });
    }
  }, []);

  const handleWSDMMessageEdited = useCallback((msg: DMMessage) => {
    const current = activeViewRef.current;
    if (
      current?.type === "dm" &&
      msg.conversation_id === current.conversation.id
    ) {
      setMessages((prev) => prev.map((m) => (m.id === msg.id ? msg : m)));
    }
  }, []);

  const handleWSDMMessageDeleted = useCallback(
    (conversationId: string, messageId: string) => {
      const current = activeViewRef.current;
      if (
        current?.type === "dm" &&
        conversationId === current.conversation.id
      ) {
        setMessages((prev) => prev.filter((m) => m.id !== messageId));
      }
    },
    []
  );

  const handleWSTyping = useCallback(
    (channelId: string, payload: { user_id: string; display_name: string }) => {
      const current = activeViewRef.current;
      const activeId =
        current?.type === "channel"
          ? current.channel.id
          : current?.type === "dm"
            ? current.conversation.id
            : null;
      if (!activeId || channelId !== activeId) return;
      if (payload.user_id === currentUser.id) return;

      const name = payload.display_name || payload.user_id.slice(0, 8);

      setTypingUsers((prev) => {
        const next = new Map(prev);
        next.set(payload.user_id, name);
        return next;
      });

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
      onDMMessage: handleWSDMMessage,
      onDMMessageEdited: handleWSDMMessageEdited,
      onDMMessageDeleted: handleWSDMMessageDeleted,
      onTyping: handleWSTyping,
    });

    return () => {
      pulseWS.disconnect();
    };
  }, [
    handleWSMessage,
    handleWSMessageEdited,
    handleWSMessageDeleted,
    handleWSDMMessage,
    handleWSDMMessageEdited,
    handleWSDMMessageDeleted,
    handleWSTyping,
  ]);

  // Subscribe/unsubscribe on active view change
  useEffect(() => {
    if (!activeView) return;
    const id =
      activeView.type === "channel"
        ? activeView.channel.id
        : activeView.conversation.id;

    pulseWS.subscribe(id);
    return () => {
      pulseWS.unsubscribe(id);
    };
  }, [activeView]);

  // Subscribe to all DM conversations for real-time notifications
  useEffect(() => {
    for (const conv of dmConversations) {
      pulseWS.subscribe(conv.id);
    }
    return () => {
      for (const conv of dmConversations) {
        pulseWS.unsubscribe(conv.id);
      }
    };
  }, [dmConversations]);

  async function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (!activeView || !messageInput.trim() || sending) return;

    setSending(true);
    try {
      if (activeView.type === "channel") {
        const msg = await api.sendMessage(
          activeView.channel.id,
          messageInput.trim()
        );
        setMessages((prev) => {
          if (prev.some((m) => m.id === msg.id)) return prev;
          return [...prev, msg];
        });
      } else {
        const msg = await api.sendDMMessage(
          activeView.conversation.id,
          messageInput.trim()
        );
        setMessages((prev) => {
          if (prev.some((m) => m.id === msg.id)) return prev;
          return [...prev, msg];
        });
      }
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
      setActiveView({ type: "channel", channel: ch });
      setNewChannelName("");
      setShowCreateChannel(false);
    } catch {
      // TODO: show error
    }
  }

  async function handleNewDM(e: React.FormEvent) {
    e.preventDefault();
    if (!newDMUsername.trim()) return;
    setDmError("");

    try {
      // First find user by username, then create DM
      // The backend getOrCreateDM takes a user_id, so we use the username lookup
      // For now, we'll use the getOrCreateDM which accepts user_id
      // We need a user search - let's use the workspace members as a source
      const members = await api.listWorkspaceMembers(workspaceId!);
      const target = members.find(
        (m) => m.username.toLowerCase() === newDMUsername.trim().toLowerCase()
      );
      if (!target) {
        setDmError("User not found in this workspace");
        return;
      }

      const conv = await api.getOrCreateDM(target.user_id);
      // Add to list if not already there
      setDmConversations((prev) => {
        if (prev.some((c) => c.id === conv.id)) return prev;
        return [conv, ...prev];
      });
      setActiveView({ type: "dm", conversation: conv });
      setNewDMUsername("");
      setShowNewDM(false);
    } catch (err) {
      setDmError(
        err instanceof Error ? err.message : "Failed to create conversation"
      );
    }
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    if (!workspaceId || !inviteEmail.trim()) return;
    setInviteError("");
    setInviteLink("");

    try {
      const res = await api.createInvite(workspaceId, inviteEmail.trim());
      setInviteLink(window.location.origin + res.link);
      setInviteEmail("");
    } catch (err) {
      setInviteError(
        err instanceof Error ? err.message : "Failed to create invite"
      );
    }
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    setMessageInput(e.target.value);
    if (activeView && e.target.value.trim()) {
      const id =
        activeView.type === "channel"
          ? activeView.channel.id
          : activeView.conversation.id;
      pulseWS.sendTyping(id);
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

  // Derive active header info
  const headerName =
    activeView?.type === "channel"
      ? `# ${activeView.channel.name}`
      : activeView?.type === "dm"
        ? activeView.conversation.other_display_name
        : "";
  const headerDesc =
    activeView?.type === "channel" ? activeView.channel.description : undefined;
  const emptyText =
    activeView?.type === "channel"
      ? `No messages yet in #${activeView.channel.name}`
      : activeView?.type === "dm"
        ? `No messages yet with ${activeView.conversation.other_display_name}`
        : "";
  const inputPlaceholder =
    activeView?.type === "channel"
      ? `Message #${activeView.channel.name}`
      : activeView?.type === "dm"
        ? `Message ${activeView.conversation.other_display_name}`
        : "";

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
                className={`channel-item ${activeView?.type === "channel" && activeView.channel.id === ch.id ? "active" : ""}`}
                onClick={() => setActiveView({ type: "channel", channel: ch })}
              >
                <span className="channel-hash">#</span>
                {ch.name}
              </button>
            ))}
          </div>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-header">
            <span>Direct Messages</span>
            <button
              className="sidebar-add-btn"
              onClick={() => {
                setShowNewDM(!showNewDM);
                setDmError("");
              }}
            >
              +
            </button>
          </div>

          {showNewDM && (
            <div style={{ padding: "0 0.75rem 0.5rem" }}>
              <form onSubmit={handleNewDM} className="sidebar-create-form">
                <input
                  type="text"
                  value={newDMUsername}
                  onChange={(e) => setNewDMUsername(e.target.value)}
                  placeholder="username"
                  autoFocus
                />
              </form>
              {dmError && (
                <p
                  style={{
                    color: "var(--error, #e74c3c)",
                    fontSize: "0.75rem",
                    margin: "0.25rem 0",
                  }}
                >
                  {dmError}
                </p>
              )}
            </div>
          )}

          <div className="channel-list">
            {dmConversations.map((conv) => (
              <button
                key={conv.id}
                className={`channel-item ${activeView?.type === "dm" && activeView.conversation.id === conv.id ? "active" : ""}`}
                onClick={() => setActiveView({ type: "dm", conversation: conv })}
                style={{
                  fontWeight: unreadDMs.has(conv.id) ? "bold" : "normal",
                }}
              >
                <span
                  className="message-avatar"
                  style={{
                    width: "1.25rem",
                    height: "1.25rem",
                    fontSize: "0.65rem",
                    flexShrink: 0,
                  }}
                >
                  {conv.other_display_name?.charAt(0).toUpperCase()}
                </span>
                {conv.other_display_name}
              </button>
            ))}
          </div>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-section-header">
            <span>Invite People</span>
            <button
              className="sidebar-add-btn"
              onClick={() => {
                setShowInvite(!showInvite);
                setInviteLink("");
                setInviteError("");
              }}
            >
              +
            </button>
          </div>

          {showInvite && (
            <div style={{ padding: "0 0.75rem 0.5rem" }}>
              <form onSubmit={handleInvite} className="sidebar-create-form">
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="email@example.com"
                  autoFocus
                />
              </form>
              {inviteError && (
                <p
                  style={{
                    color: "var(--error, #e74c3c)",
                    fontSize: "0.75rem",
                    margin: "0.25rem 0",
                  }}
                >
                  {inviteError}
                </p>
              )}
              {inviteLink && (
                <div style={{ marginTop: "0.5rem" }}>
                  <p
                    style={{
                      fontSize: "0.75rem",
                      color: "var(--text-secondary, #999)",
                      margin: "0 0 0.25rem",
                    }}
                  >
                    Share this link:
                  </p>
                  <div style={{ display: "flex", gap: "0.25rem" }}>
                    <input
                      type="text"
                      value={inviteLink}
                      readOnly
                      style={{
                        flex: 1,
                        fontSize: "0.7rem",
                        padding: "0.25rem",
                      }}
                      onClick={(e) =>
                        (e.target as HTMLInputElement).select()
                      }
                    />
                    <button
                      type="button"
                      style={{
                        fontSize: "0.7rem",
                        padding: "0.25rem 0.5rem",
                        cursor: "pointer",
                      }}
                      onClick={() =>
                        navigator.clipboard.writeText(inviteLink)
                      }
                    >
                      Copy
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </aside>

      {/* Main Content */}
      <main className="main-content">
        {activeView ? (
          <>
            <div className="main-header">
              <h2>{headerName}</h2>
              {headerDesc && (
                <p className="main-header-desc">{headerDesc}</p>
              )}
            </div>

            <div className="messages-area">
              {messages.length === 0 ? (
                <div className="messages-empty">
                  <p>{emptyText}</p>
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
                placeholder={inputPlaceholder}
                value={messageInput}
                onChange={handleInputChange}
                disabled={sending}
              />
            </form>
          </>
        ) : (
          <div className="no-channel">
            <p>Select a channel or conversation to start chatting</p>
          </div>
        )}
      </main>
    </div>
  );
}
