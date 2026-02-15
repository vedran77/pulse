import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../lib/api";

export default function CreateWorkspace() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  function handleNameChange(value: string) {
    setName(value);
    // Auto-generate slug from name
    setSlug(
      value
        .toLowerCase()
        .trim()
        .replace(/[^a-z0-9\s-]/g, "")
        .replace(/\s+/g, "-")
        .replace(/-+/g, "-")
    );
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const ws = await api.createWorkspace({ name, slug, description });

      // Automatski kreiraj #general kanal
      await api.createChannel(ws.id, {
        name: "general",
        description: "General discussion",
        type: "public",
      });

      navigate(`/w/${ws.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <img src="/logo.png" alt="Pulse" className="auth-logo" />
        <h1>Pulse</h1>
        <p className="auth-subtitle">Create a new workspace</p>

        {error && <div className="auth-error">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="name">Workspace Name</label>
            <input
              id="name"
              type="text"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="My Team"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="slug">URL Slug</label>
            <input
              id="slug"
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="my-team"
              required
            />
            <span className="field-hint">pulse.app/{slug || "..."}</span>
          </div>

          <div className="form-group">
            <label htmlFor="description">Description (optional)</label>
            <input
              id="description"
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this workspace for?"
            />
          </div>

          <button type="submit" className="auth-button" disabled={loading}>
            {loading ? "Creating..." : "Create Workspace"}
          </button>
        </form>

        <p className="auth-link">
          <Link to="/">Back to workspaces</Link>
        </p>
      </div>
    </div>
  );
}
