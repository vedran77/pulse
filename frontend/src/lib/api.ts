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

export interface DMConversation {
  id: string;
  user1_id: string;
  user2_id: string;
  created_at: string;
  other_user_id: string;
  other_username: string;
  other_display_name: string;
}

export interface DMMessage {
  id: string;
  conversation_id: string;
  sender_id: string;
  content?: string;
  edited_at?: string;
  created_at: string;
  sender_username: string;
  sender_display_name: string;
}

export interface DMMessageListResponse {
  messages: DMMessage[];
  has_more: boolean;
}

export interface PulsemateRequest {
  id: string;
  sender_id: string;
  receiver_id: string;
  status: string;
  created_at: string;
  sender_username?: string;
  sender_display_name?: string;
  receiver_username?: string;
  receiver_display_name?: string;
}

export interface Pulsemate {
  id: string;
  user1_id: string;
  user2_id: string;
  created_at: string;
  other_user_id: string;
  other_username: string;
  other_display_name: string;
  other_status: string;
}

export interface WorkspaceMember {
  workspace_id: string;
  user_id: string;
  role: string;
  joined_at: string;
  username: string;
  display_name: string;
}

export interface WorkspaceInvite {
  id: string;
  workspace_id: string;
  email: string;
  token: string;
  invited_by: string;
  created_at: string;
  expires_at: string;
  accepted_at?: string;
  accepted_by?: string;
}

export interface InviteInfo {
  workspace_name: string;
  email: string;
  expires_at: string;
  accepted: boolean;
}

export interface CreateInviteResponse {
  invite: WorkspaceInvite;
  link: string;
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

  listWorkspaceMembers(workspaceId: string) {
    return request<WorkspaceMember[]>(`/workspaces/${workspaceId}/members`);
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

  // Invites
  createInvite(workspaceId: string, email: string) {
    return request<CreateInviteResponse>(`/workspaces/${workspaceId}/invites`, {
      method: "POST",
      body: JSON.stringify({ email }),
    });
  },

  listInvites(workspaceId: string) {
    return request<WorkspaceInvite[]>(`/workspaces/${workspaceId}/invites`);
  },

  revokeInvite(workspaceId: string, inviteId: string) {
    return request<void>(`/workspaces/${workspaceId}/invites/${inviteId}`, {
      method: "DELETE",
    });
  },

  getInviteInfo(token: string) {
    return request<InviteInfo>(`/invites/${token}`);
  },

  acceptInvite(token: string) {
    return request<{ workspace_id: string }>(`/invites/${token}/accept`, {
      method: "POST",
    });
  },

  // Direct Messages
  getOrCreateDM(userId: string) {
    return request<DMConversation>("/dm/conversations", {
      method: "POST",
      body: JSON.stringify({ user_id: userId }),
    });
  },

  listDMConversations() {
    return request<DMConversation[]>("/dm/conversations");
  },

  listDMMessages(conversationId: string, before?: string, limit = 50) {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set("before", before);
    return request<DMMessageListResponse>(
      `/dm/conversations/${conversationId}/messages?${params}`
    );
  },

  sendDMMessage(conversationId: string, content: string) {
    return request<DMMessage>(`/dm/conversations/${conversationId}/messages`, {
      method: "POST",
      body: JSON.stringify({ content }),
    });
  },

  editDMMessage(messageId: string, content: string) {
    return request<DMMessage>(`/dm/messages/${messageId}`, {
      method: "PATCH",
      body: JSON.stringify({ content }),
    });
  },

  deleteDMMessage(messageId: string) {
    return request<void>(`/dm/messages/${messageId}`, { method: "DELETE" });
  },

  // Pulsemates
  sendPulsemateRequest(username: string) {
    return request<PulsemateRequest | { status: string }>("/pulsemates/requests", {
      method: "POST",
      body: JSON.stringify({ username }),
    });
  },

  listPulsemates() {
    return request<Pulsemate[]>("/pulsemates");
  },

  listIncomingRequests() {
    return request<PulsemateRequest[]>("/pulsemates/requests/incoming");
  },

  listOutgoingRequests() {
    return request<PulsemateRequest[]>("/pulsemates/requests/outgoing");
  },

  acceptPulsemateRequest(id: string) {
    return request<void>(`/pulsemates/requests/${id}/accept`, { method: "POST" });
  },

  rejectPulsemateRequest(id: string) {
    return request<void>(`/pulsemates/requests/${id}/reject`, { method: "POST" });
  },

  cancelPulsemateRequest(id: string) {
    return request<void>(`/pulsemates/requests/${id}`, { method: "DELETE" });
  },

  removePulsemate(userId: string) {
    return request<void>(`/pulsemates/${userId}`, { method: "DELETE" });
  },
};
