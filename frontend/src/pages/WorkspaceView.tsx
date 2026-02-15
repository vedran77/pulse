import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api, type Workspace, type Channel } from "../lib/api";

export default function WorkspaceView() {
  const { workspaceId } = useParams();
  const navigate = useNavigate();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [activeChannel, setActiveChannel] = useState<Channel | null>(null);
  const [loading, setLoading] = useState(true);
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [newChannelName, setNewChannelName] = useState("");

  useEffect(() => {
    if (!workspaceId) return;

    Promise.all([
      api.getWorkspace(workspaceId),
      api.listChannels(workspaceId),
    ])
      .then(([ws, chs]) => {
        setWorkspace(ws);
        setChannels(chs);
        if (chs.length > 0) setActiveChannel(chs[0]);
      })
      .catch(() => navigate("/"))
      .finally(() => setLoading(false));
  }, [workspaceId, navigate]);

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

  if (loading) {
    return <div className="workspace-loading">Loading...</div>;
  }

  return (
    <div className="workspace-layout">
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="sidebar-header">
          <button className="sidebar-workspace-btn" onClick={() => navigate("/")}>
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
            <form onSubmit={handleCreateChannel} className="sidebar-create-form">
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
              <div className="messages-empty">
                <p>No messages yet in #{activeChannel.name}</p>
                <p className="messages-empty-sub">
                  Be the first to send a message!
                </p>
              </div>
            </div>
            <div className="message-input-area">
              <input
                type="text"
                className="message-input"
                placeholder={`Message #${activeChannel.name}`}
                disabled
              />
            </div>
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
