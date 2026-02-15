const API_BASE = "http://localhost:8080/api/v1";

interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

async function request<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const token = localStorage.getItem("access_token");

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const err: ApiError = await res.json();
    throw new Error(err.error.message);
  }

  return res.json();
}

export interface User {
  id: string;
  email: string;
  username: string;
  display_name: string;
  status: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
}

export const api = {
  register(data: {
    email: string;
    username: string;
    display_name: string;
    password: string;
  }) {
    return request<AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  login(data: { email: string; password: string }) {
    return request<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },
};
