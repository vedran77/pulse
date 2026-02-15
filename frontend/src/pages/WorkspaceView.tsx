import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  api,
  type Workspace,
  type Channel,
  type Message,
  type DMConversation,
  type DMMessage,
  type Pulsemate,
  type PulsemateRequest,
} from "../lib/api";
import { pulseWS } from "../lib/ws";

type ActiveView =
  | { type: "channel"; channel: Channel }
  | { type: "dm"; conversation: DMConversation };

function shouldGroupWithPrevious(
  prev: Message | DMMessage | undefined,
  curr: Message | DMMessage
): boolean {
  if (!prev) return false;
  return (
    prev.sender_id === curr.sender_id &&
    new Date(curr.created_at).getTime() - new Date(prev.created_at).getTime() <
      7 * 60 * 1000
  );
}

export default function WorkspaceView() {
  const { workspaceId } = useParams();
  const navigate = useNavigate();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
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
  const [channelsCollapsed, setChannelsCollapsed] = useState(false);
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
  const [editContent, setEditContent] = useState("");
  const [wsDropdownOpen, setWsDropdownOpen] = useState(false);
  const [pulsemates, setPulsemates] = useState<Pulsemate[]>([]);
  const [pulsematesCollapsed, setPulsematesCollapsed] = useState(false);
  const [showAddPulsemate, setShowAddPulsemate] = useState(false);
  const [pulsemateUsername, setPulsemateUsername] = useState("");
  const [pulsemateError, setPulsemateError] = useState("");
  const [incomingRequests, setIncomingRequests] = useState<PulsemateRequest[]>([]);
  const [showRequestsPanel, setShowRequestsPanel] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const activeViewRef = useRef<ActiveView | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const wsDropdownRef = useRef<HTMLDivElement>(null);
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

  // Close workspace dropdown on outside click
  useEffect(() => {
    if (!wsDropdownOpen) return;
    function handleClick(e: MouseEvent) {
      if (
        wsDropdownRef.current &&
        !wsDropdownRef.current.contains(e.target as Node)
      ) {
        setWsDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [wsDropdownOpen]);

  // Load workspace + channels + DM conversations + all workspaces
  useEffect(() => {
    if (!workspaceId) return;

    Promise.all([
      api.getWorkspace(workspaceId),
      api.listChannels(workspaceId),
      api.listDMConversations(),
      api.listWorkspaces(),
      api.listPulsemates(),
      api.listIncomingRequests(),
    ])
      .then(([ws, chs, dms, allWs, pms, incoming]) => {
        setWorkspace(ws);
        setChannels(chs);
        setDmConversations(dms);
        setWorkspaces(allWs);
        setPulsemates(pms);
        setIncomingRequests(incoming);
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
    setEditingMessageId(null);

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

  async function handleSend() {
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
      // Reset textarea height
      if (textareaRef.current) {
        textareaRef.current.style.height = "auto";
      }
    } catch {
      // TODO: show error
    } finally {
      setSending(false);
      textareaRef.current?.focus();
    }
  }

  function handleTextareaKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleTextareaChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setMessageInput(e.target.value);
    // Auto-grow
    e.target.style.height = "auto";
    e.target.style.height = e.target.scrollHeight + "px";
    // Send typing indicator
    if (activeView && e.target.value.trim()) {
      const id =
        activeView.type === "channel"
          ? activeView.channel.id
          : activeView.conversation.id;
      pulseWS.sendTyping(id);
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

  async function handleAddPulsemate(e: React.FormEvent) {
    e.preventDefault();
    if (!pulsemateUsername.trim()) return;
    setPulsemateError("");

    try {
      await api.sendPulsemateRequest(pulsemateUsername.trim());
      setPulsemateUsername("");
      setShowAddPulsemate(false);
      // Refresh lists
      const [pms, incoming] = await Promise.all([
        api.listPulsemates(),
        api.listIncomingRequests(),
      ]);
      setPulsemates(pms);
      setIncomingRequests(incoming);
    } catch (err) {
      setPulsemateError(
        err instanceof Error ? err.message : "Failed to send request"
      );
    }
  }

  async function handleAcceptRequest(requestId: string) {
    try {
      await api.acceptPulsemateRequest(requestId);
      const [pms, incoming] = await Promise.all([
        api.listPulsemates(),
        api.listIncomingRequests(),
      ]);
      setPulsemates(pms);
      setIncomingRequests(incoming);
    } catch {
      // TODO: show error
    }
  }

  async function handleRejectRequest(requestId: string) {
    try {
      await api.rejectPulsemateRequest(requestId);
      setIncomingRequests((prev) => prev.filter((r) => r.id !== requestId));
    } catch {
      // TODO: show error
    }
  }

  async function handlePulsemateClick(pm: Pulsemate) {
    try {
      const conv = await api.getOrCreateDM(pm.other_user_id);
      setDmConversations((prev) => {
        if (prev.some((c) => c.id === conv.id)) return prev;
        return [conv, ...prev];
      });
      setActiveView({ type: "dm", conversation: conv });
    } catch {
      // TODO: show error
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

  function startEdit(msg: Message | DMMessage) {
    setEditingMessageId(msg.id);
    setEditContent(msg.content || "");
  }

  async function saveEdit() {
    if (!editingMessageId || !editContent.trim() || !activeView) return;

    try {
      if (activeView.type === "channel") {
        const updated = await api.editMessage(editingMessageId, editContent.trim());
        setMessages((prev) => prev.map((m) => (m.id === updated.id ? updated : m)));
      } else {
        const updated = await api.editDMMessage(editingMessageId, editContent.trim());
        setMessages((prev) => prev.map((m) => (m.id === updated.id ? updated : m)));
      }
    } catch {
      // TODO: show error
    }
    setEditingMessageId(null);
    setEditContent("");
  }

  function cancelEdit() {
    setEditingMessageId(null);
    setEditContent("");
  }

  function handleEditKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      saveEdit();
    } else if (e.key === "Escape") {
      cancelEdit();
    }
  }

  async function handleDelete(msg: Message | DMMessage) {
    if (!activeView) return;

    try {
      if (activeView.type === "channel") {
        await api.deleteMessage(msg.id);
      } else {
        await api.deleteDMMessage(msg.id);
      }
      // Optimistic removal
      setMessages((prev) => prev.filter((m) => m.id !== msg.id));
    } catch {
      // TODO: show error
    }
  }

  function formatTime(dateStr: string) {
    return new Date(dateStr).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function formatTimestamp(dateStr: string) {
    const date = new Date(dateStr);
    const today = new Date();
    const isToday = date.toDateString() === today.toDateString();
    const time = date.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
    return isToday ? `Today at ${time}` : `${date.toLocaleDateString()} ${time}`;
  }

  const typingText =
    typingUsers.size > 0
      ? Array.from(typingUsers.values()).join(", ") + " is typing..."
      : null;

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
        <div className="sidebar-header" ref={wsDropdownRef}>
          <div className="sidebar-header-top">
            <button
              className="ws-dropdown-trigger"
              onClick={() => setWsDropdownOpen(!wsDropdownOpen)}
            >
              <img src="/logo.png" alt="Pulse" className="sidebar-logo" />
              <span className="sidebar-header-name">{workspace?.name}</span>
              <span className={`ws-dropdown-chevron ${wsDropdownOpen ? "open" : ""}`}>
                &#9662;
              </span>
            </button>
            <div className="sidebar-header-actions">
              <button
                className="sidebar-settings-btn"
                onClick={() => {
                  setShowInvite(!showInvite);
                  setInviteLink("");
                  setInviteError("");
                }}
                title="Invite People"
              >
                &#9881;
              </button>
            </div>
          </div>

          {wsDropdownOpen && (
            <div className="ws-dropdown">
              {workspaces.map((ws) => (
                <button
                  key={ws.id}
                  className={`ws-dropdown-item ${ws.id === workspaceId ? "active" : ""}`}
                  onClick={() => {
                    navigate(`/w/${ws.id}`);
                    setWsDropdownOpen(false);
                  }}
                >
                  <span className="ws-dropdown-avatar">
                    {ws.name.charAt(0).toUpperCase()}
                  </span>
                  <span className="ws-dropdown-name">{ws.name}</span>
                </button>
              ))}
              <div className="ws-dropdown-separator" />
              <button
                className="ws-dropdown-create"
                onClick={() => {
                  navigate("/create-workspace");
                  setWsDropdownOpen(false);
                }}
              >
                + Create workspace
              </button>
            </div>
          )}

          {showInvite && (
            <div className="invite-dropdown">
              <form onSubmit={handleInvite}>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="email@example.com"
                  autoFocus
                />
              </form>
              {inviteError && <p className="invite-error">{inviteError}</p>}
              {inviteLink && (
                <div className="invite-link-box">
                  <p className="invite-link-label">Share this link:</p>
                  <div className="invite-link-row">
                    <input
                      type="text"
                      value={inviteLink}
                      readOnly
                      onClick={(e) => (e.target as HTMLInputElement).select()}
                    />
                    <button
                      type="button"
                      onClick={() => navigator.clipboard.writeText(inviteLink)}
                    >
                      Copy
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Search placeholder */}
        <div className="sidebar-search">
          <input placeholder="Search..." disabled />
        </div>

        <div className="sidebar-content">
          {/* Channels category */}
          <div className="sidebar-category">
            <div className="sidebar-category-header">
              <span
                className="sidebar-category-label"
                onClick={() => setChannelsCollapsed(!channelsCollapsed)}
              >
                <span
                  className={`sidebar-category-arrow ${channelsCollapsed ? "collapsed" : ""}`}
                >
                  &#9662;
                </span>
                Channels
              </span>
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

            {!channelsCollapsed && (
              <div className="channel-list">
                {channels.map((ch) => (
                  <button
                    key={ch.id}
                    className={`channel-item ${activeView?.type === "channel" && activeView.channel.id === ch.id ? "active" : ""}`}
                    onClick={() =>
                      setActiveView({ type: "channel", channel: ch })
                    }
                  >
                    <span className="channel-dot" />
                    {ch.name}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Separator */}
          <div className="sidebar-separator" />

          {/* Pulsemates category */}
          <div className="sidebar-category">
            <div className="sidebar-category-header">
              <span
                className="sidebar-category-label"
                onClick={() => setPulsematesCollapsed(!pulsematesCollapsed)}
              >
                <span
                  className={`sidebar-category-arrow ${pulsematesCollapsed ? "collapsed" : ""}`}
                >
                  &#9662;
                </span>
                Pulsemates
                {incomingRequests.length > 0 && (
                  <span
                    className="request-badge"
                    onClick={(e) => {
                      e.stopPropagation();
                      setShowRequestsPanel(!showRequestsPanel);
                    }}
                  >
                    {incomingRequests.length}
                  </span>
                )}
              </span>
              <button
                className="sidebar-add-btn"
                onClick={() => {
                  setShowAddPulsemate(!showAddPulsemate);
                  setPulsemateError("");
                }}
              >
                +
              </button>
            </div>

            {showAddPulsemate && (
              <div style={{ padding: "0 12px" }}>
                <form onSubmit={handleAddPulsemate} className="sidebar-create-form">
                  <input
                    type="text"
                    value={pulsemateUsername}
                    onChange={(e) => setPulsemateUsername(e.target.value)}
                    placeholder="username"
                    autoFocus
                  />
                </form>
                {pulsemateError && (
                  <p className="invite-error" style={{ padding: "0 12px" }}>
                    {pulsemateError}
                  </p>
                )}
              </div>
            )}

            {showRequestsPanel && incomingRequests.length > 0 && (
              <div className="pulsemate-requests">
                {incomingRequests.map((req) => (
                  <div key={req.id} className="request-item">
                    <span className="request-item-name">
                      {req.sender_display_name || req.sender_username}
                    </span>
                    <div className="request-item-actions">
                      <button
                        className="request-btn accept"
                        onClick={() => handleAcceptRequest(req.id)}
                        title="Accept"
                      >
                        &#10003;
                      </button>
                      <button
                        className="request-btn reject"
                        onClick={() => handleRejectRequest(req.id)}
                        title="Reject"
                      >
                        &#10005;
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {!pulsematesCollapsed && (
              <div className="channel-list">
                {pulsemates.length === 0 ? (
                  <div className="empty-pulsemates">No pulsemates yet</div>
                ) : (
                  pulsemates.map((pm) => (
                    <button
                      key={pm.id}
                      className="channel-item"
                      onClick={() => handlePulsemateClick(pm)}
                    >
                      <span className="dm-avatar">
                        {pm.other_display_name?.charAt(0).toUpperCase()}
                      </span>
                      <span className={`pulsemate-status ${pm.other_status === "online" ? "online" : ""}`} />
                      {pm.other_display_name}
                    </button>
                  ))
                )}
              </div>
            )}
          </div>
        </div>

        {/* User Area */}
        <div className="user-area">
          <div className="user-area-avatar">
            {currentUser.display_name?.charAt(0).toUpperCase() ||
              currentUser.username?.charAt(0).toUpperCase() ||
              "?"}
            <span className="status-dot" />
          </div>
          <div className="user-area-info">
            <div className="user-area-name">
              {currentUser.display_name || currentUser.username}
            </div>
            <div className="user-area-status">Online</div>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="main-content">
        {activeView ? (
          <>
            <div className="main-header">
              <h2>{headerName}</h2>
              {headerDesc && (
                <span className="main-header-desc">{headerDesc}</span>
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
                  {messages.map((msg, i) => {
                    const prev = i > 0 ? messages[i - 1] : undefined;
                    const grouped = shouldGroupWithPrevious(prev, msg);
                    const isOwn = msg.sender_id === currentUser.id;
                    const isEditing = editingMessageId === msg.id;

                    if (grouped) {
                      return (
                        <div
                          key={msg.id}
                          className="message message-continuation"
                        >
                          <span className="message-hover-time">
                            {formatTime(msg.created_at)}
                          </span>
                          {isEditing ? (
                            <div className="message-edit-container">
                              <textarea
                                className="message-edit-textarea"
                                value={editContent}
                                onChange={(e) => setEditContent(e.target.value)}
                                onKeyDown={handleEditKeyDown}
                                autoFocus
                              />
                              <div className="message-edit-actions">
                                <span>
                                  <kbd>Escape</kbd> to cancel
                                </span>
                                <span>
                                  <kbd>Enter</kbd> to save
                                </span>
                              </div>
                            </div>
                          ) : (
                            <p className="message-text">
                              {msg.content}
                              {msg.edited_at && (
                                <span className="message-edited">
                                  (edited)
                                </span>
                              )}
                            </p>
                          )}
                          {isOwn && !isEditing && (
                            <div className="message-hover-actions">
                              <button
                                className="message-action-btn"
                                onClick={() => startEdit(msg)}
                                title="Edit"
                              >
                                &#9998;
                              </button>
                              <button
                                className="message-action-btn danger"
                                onClick={() => handleDelete(msg)}
                                title="Delete"
                              >
                                &#10006;
                              </button>
                            </div>
                          )}
                        </div>
                      );
                    }

                    return (
                      <div
                        key={msg.id}
                        className="message message-group-start"
                      >
                        <div
                          className={`message-group-avatar ${isOwn ? "own-avatar" : ""}`}
                        >
                          {msg.sender_display_name?.charAt(0).toUpperCase()}
                        </div>
                        <div className="message-group-header">
                          <span className="message-sender">
                            {msg.sender_display_name}
                          </span>
                          <span className="message-timestamp">
                            {formatTimestamp(msg.created_at)}
                          </span>
                        </div>
                        {isEditing ? (
                          <div className="message-edit-container">
                            <textarea
                              className="message-edit-textarea"
                              value={editContent}
                              onChange={(e) => setEditContent(e.target.value)}
                              onKeyDown={handleEditKeyDown}
                              autoFocus
                            />
                            <div className="message-edit-actions">
                              <span>
                                <kbd>Escape</kbd> to cancel
                              </span>
                              <span>
                                <kbd>Enter</kbd> to save
                              </span>
                            </div>
                          </div>
                        ) : (
                          <p className="message-text">
                            {msg.content}
                            {msg.edited_at && (
                              <span className="message-edited">(edited)</span>
                            )}
                          </p>
                        )}
                        {isOwn && !isEditing && (
                          <div className="message-hover-actions">
                            <button
                              className="message-action-btn"
                              onClick={() => startEdit(msg)}
                              title="Edit"
                            >
                              &#9998;
                            </button>
                            <button
                              className="message-action-btn danger"
                              onClick={() => handleDelete(msg)}
                              title="Delete"
                            >
                              &#10006;
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                  <div ref={messagesEndRef} />
                </div>
              )}
            </div>

            {typingText && (
              <div className="typing-indicator">{typingText}</div>
            )}

            <div className="message-input-area">
              <div className="message-input-container">
                <textarea
                  ref={textareaRef}
                  className="message-input"
                  placeholder={inputPlaceholder}
                  value={messageInput}
                  onChange={handleTextareaChange}
                  onKeyDown={handleTextareaKeyDown}
                  rows={1}
                />
                <button
                  className="message-send-btn"
                  onClick={handleSend}
                  disabled={sending || !messageInput.trim()}
                  title="Send message"
                >
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                    <path d="M5.25 2.5L21 12L5.25 21.5V14.25L16.5 12L5.25 9.75V2.5Z" fill="currentColor"/>
                  </svg>
                </button>
              </div>
            </div>
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
