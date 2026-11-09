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
