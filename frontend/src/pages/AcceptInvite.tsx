import { useEffect, useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { api, type InviteInfo } from "../lib/api";

export default function AcceptInvite() {
  const { token } = useParams();
  const navigate = useNavigate();
  const [invite, setInvite] = useState<InviteInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [accepting, setAccepting] = useState(false);
  const [error, setError] = useState("");

  const isLoggedIn = !!localStorage.getItem("access_token");

  useEffect(() => {
    if (!token) return;
    api
      .getInviteInfo(token)
      .then(setInvite)
      .catch((err) => setError(err instanceof Error ? err.message : "Invite not found"))
      .finally(() => setLoading(false));
  }, [token]);

  async function handleAccept() {
    if (!token) return;
    setAccepting(true);
    setError("");
    try {
      const res = await api.acceptInvite(token);
      navigate(`/w/${res.workspace_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setAccepting(false);
    }
  }

  if (loading) {
    return <div className="auth-container"><div className="auth-card">Loading...</div></div>;
  }

  if (error && !invite) {
    return (
      <div className="auth-container">
        <div className="auth-card">
          <h1>Invalid Invite</h1>
          <p className="auth-subtitle">{error}</p>
          <Link to="/" className="auth-button" style={{ display: "block", textAlign: "center", textDecoration: "none" }}>
            Go Home
          </Link>
        </div>
      </div>
    );
  }

  if (invite?.accepted) {
    return (
      <div className="auth-container">
        <div className="auth-card">
          <h1>Invite Already Used</h1>
          <p className="auth-subtitle">This invite has already been accepted.</p>
          <Link to="/" className="auth-button" style={{ display: "block", textAlign: "center", textDecoration: "none" }}>
            Go Home
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <img src="/logo.png" alt="Pulse" className="auth-logo" />
        <h1>You're Invited!</h1>
        <p className="auth-subtitle">
          You've been invited to join <strong>{invite?.workspace_name}</strong>
        </p>

        {error && <div className="auth-error">{error}</div>}

        {isLoggedIn ? (
          <button
            className="auth-button"
            onClick={handleAccept}
            disabled={accepting}
          >
            {accepting ? "Joining..." : "Accept Invite"}
          </button>
        ) : (
          <div>
            <p className="auth-subtitle">Sign in or create an account to join</p>
            <div style={{ display: "flex", gap: "0.5rem" }}>
              <Link
                to={`/login?redirect=/invite/${token}`}
                className="auth-button"
                style={{ flex: 1, textAlign: "center", textDecoration: "none" }}
              >
                Sign In
              </Link>
              <Link
                to={`/register?redirect=/invite/${token}`}
                className="auth-button"
                style={{ flex: 1, textAlign: "center", textDecoration: "none", background: "transparent", border: "1px solid var(--primary)", color: "var(--primary)" }}
              >
                Register
              </Link>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
