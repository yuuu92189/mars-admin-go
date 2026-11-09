import type { ChatMessage } from "@/lib/api/types";

export function isDirectMessageForPeer(
  message: ChatMessage,
  currentUserId: string | undefined,
  peerId: string | null,
): boolean {
  if (!currentUserId || !peerId || message.groupId) return false;
  return (
    (message.senderId === peerId && message.receiverId === currentUserId) ||
    (message.senderId === currentUserId && message.receiverId === peerId)
  );
}

export function mergeDirectMessageHistory(
  current: ChatMessage[],
  history: ChatMessage[],
  currentUserId: string | undefined,
  peerId: string | null,
): ChatMessage[] {
  const historyIDs = new Set(history.flatMap((message) => (message.id ? [message.id] : [])));
  const pending = current.filter(
    (message) =>
      message.id &&
      !historyIDs.has(message.id) &&
      isDirectMessageForPeer(message, currentUserId, peerId),
  );
  return [...pending, ...history];
}
