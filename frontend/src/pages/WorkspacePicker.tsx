import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, type Workspace } from "../lib/api";

export default function WorkspacePicker() {
  const navigate = useNavigate();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .listWorkspaces()
      .then((ws) => setWorkspaces(ws))
      .catch(() => navigate("/login"))
      .finally(() => setLoading(false));
  }, [navigate]);

  if (loading) {
    return (
      <div className="picker-container">
        <div className="picker-card">
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="picker-container">
      <div className="picker-card">
        <img src="/logo.png" alt="Pulse" className="picker-logo" />
        <h1>Pulse</h1>

        {workspaces.length > 0 ? (
          <>
            <p className="picker-subtitle">Choose a workspace</p>
            <div className="workspace-list">
              {workspaces.map((ws) => (
                <button
                  key={ws.id}
                  className="workspace-item"
                  onClick={() => navigate(`/w/${ws.id}`)}
                >
                  <span className="workspace-avatar">
                    {ws.name.charAt(0).toUpperCase()}
                  </span>
                  <div className="workspace-info">
                    <span className="workspace-name">{ws.name}</span>
                    <span className="workspace-slug">/{ws.slug}</span>
                  </div>
                </button>
              ))}
            </div>
            <div className="picker-divider">
              <span>or</span>
            </div>
          </>
        ) : (
          <p className="picker-subtitle">
            You are not part of any workspace yet
          </p>
        )}

        <Link to="/create-workspace" className="auth-button picker-create-btn">
          Create a new workspace
        </Link>

        <button
          className="logout-button picker-logout"
          onClick={() => {
            localStorage.clear();
            navigate("/login");
          }}
        >
          Sign Out
        </button>
      </div>
    </div>
  );
}
