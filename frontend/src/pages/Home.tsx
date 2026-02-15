import { useNavigate } from "react-router-dom";
import { User } from "../lib/api";

export default function Home() {
  const navigate = useNavigate();
  const userJson = localStorage.getItem("user");
  const user: User | null = userJson ? JSON.parse(userJson) : null;

  if (!user) {
    navigate("/login");
    return null;
  }

  function handleLogout() {
    localStorage.removeItem("access_token");
    localStorage.removeItem("user");
    navigate("/login");
  }

  return (
    <div className="home-container">
      <div className="home-card">
        <h1>Welcome, {user.display_name}!</h1>
        <p>You're signed in as <strong>@{user.username}</strong></p>
        <p className="home-email">{user.email}</p>
        <button onClick={handleLogout} className="logout-button">
          Sign Out
        </button>
      </div>
    </div>
  );
}
