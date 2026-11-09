import { apiRequest } from "./client";
import type { ChatGroup, ChatMessage, ChatUser } from "./types";

export function listChatUsers(): Promise<ChatUser[]> {
  return apiRequest<ChatUser[]>("/api/v1/messages/users");
}

export function listDirectMessages(
  peerId: string,
  options: { before?: number; limit?: number } = {},
): Promise<ChatMessage[]> {
  const params = new URLSearchParams();
  if (options.before) params.set("before", String(options.before));
  if (options.limit) params.set("limit", String(options.limit));
  const suffix = params.toString();
  return apiRequest<ChatMessage[]>(
    `/api/v1/messages/${encodeURIComponent(peerId)}${suffix ? `?${suffix}` : ""}`,
  );
}

export function sendDirectMessage(input: {
  receiverId: string;
  content: string;
  messageType?: string;
  clientMessageId?: string;
}): Promise<ChatMessage> {
  return apiRequest<ChatMessage>("/api/v1/messages", {
    method: "POST",
    body: {
      receiverId: input.receiverId,
      content: input.content,
      messageType: input.messageType ?? "text",
      clientMessageId: input.clientMessageId ?? crypto.randomUUID(),
    },
  });
}

export function markDirectMessagesRead(peerId: string): Promise<void> {
  return apiRequest<void>(`/api/v1/messages/${encodeURIComponent(peerId)}/read`, {
    method: "POST",
  });
}

export function listChatGroups(): Promise<ChatGroup[]> {
  return apiRequest<ChatGroup[]>("/api/v1/groups");
}

export function listGroupMessages(groupId: string): Promise<ChatMessage[]> {
  return apiRequest<ChatMessage[]>(`/api/v1/groups/${encodeURIComponent(groupId)}/messages`);
}
