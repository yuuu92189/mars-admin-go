import { describe, expect, it } from "vitest";
import type { ChatMessage } from "@/lib/api/types";
import { isDirectMessageForPeer } from "./realtime";

const currentUserId = "019b0000-0000-7000-8000-000000000001";
const selectedPeerId = "019b0000-0000-7000-8000-000000000002";
const otherPeerId = "019b0000-0000-7000-8000-000000000003";

function message(overrides: Partial<ChatMessage>): ChatMessage {
  return {
    id: "019b0000-0000-7000-8000-000000000010",
    senderId: selectedPeerId,
    receiverId: currentUserId,
    content: "hello",
    ...overrides,
  };
}

describe("isDirectMessageForPeer", () => {
  it("accepts incoming and outgoing messages for the selected conversation", () => {
    expect(isDirectMessageForPeer(message({}), currentUserId, selectedPeerId)).toBe(true);
    expect(
      isDirectMessageForPeer(
        message({ senderId: currentUserId, receiverId: selectedPeerId }),
        currentUserId,
        selectedPeerId,
      ),
    ).toBe(true);
  });

  it("rejects messages from other contacts and group conversations", () => {
    expect(
      isDirectMessageForPeer(
        message({ senderId: otherPeerId, receiverId: currentUserId }),
        currentUserId,
        selectedPeerId,
      ),
    ).toBe(false);
    expect(
      isDirectMessageForPeer(
        message({ groupId: "019b0000-0000-7000-8000-000000000020", receiverId: undefined }),
        currentUserId,
        selectedPeerId,
      ),
    ).toBe(false);
  });
});
