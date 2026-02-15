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

  // 204 No Content
  if (res.status === 204) return undefined as T;

  return res.json();
}

// Types

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

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  description?: string;
  owner_id: string;
  created_at: string;
}

export interface Channel {
  id: string;
  workspace_id: string;
  name: string;
  description?: string;
  type: string;
  is_encrypted: boolean;
  created_by: string;
  created_at: string;
}

export interface Message {
  id: string;
  channel_id: string;
  sender_id: string;
  content?: string;
  type: string;
  parent_id?: string;
  edited_at?: string;
  created_at: string;
  sender_username: string;
  sender_display_name: string;
}

export interface MessageListResponse {
  messages: Message[];
  has_more: boolean;
}

// API

export const api = {
  // Auth
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

  // Workspaces
  listWorkspaces() {
    return request<Workspace[]>("/workspaces");
  },

  createWorkspace(data: { name: string; slug: string; description?: string }) {
    return request<Workspace>("/workspaces", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getWorkspace(id: string) {
    return request<Workspace>(`/workspaces/${id}`);
  },

  // Channels
  listChannels(workspaceId: string) {
    return request<Channel[]>(`/workspaces/${workspaceId}/channels`);
  },

  createChannel(
    workspaceId: string,
    data: { name: string; description?: string; type?: string }
  ) {
    return request<Channel>(`/workspaces/${workspaceId}/channels`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  joinChannel(channelId: string) {
    return request<void>(`/channels/${channelId}/join`, { method: "POST" });
  },

  // Messages
  listMessages(channelId: string, before?: string, limit = 50) {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set("before", before);
    return request<MessageListResponse>(
      `/channels/${channelId}/messages?${params}`
    );
  },

  sendMessage(channelId: string, content: string) {
    return request<Message>(`/channels/${channelId}/messages`, {
      method: "POST",
      body: JSON.stringify({ content }),
    });
  },

  editMessage(messageId: string, content: string) {
    return request<Message>(`/messages/${messageId}`, {
      method: "PATCH",
      body: JSON.stringify({ content }),
    });
  },

  deleteMessage(messageId: string) {
    return request<void>(`/messages/${messageId}`, { method: "DELETE" });
  },
};
