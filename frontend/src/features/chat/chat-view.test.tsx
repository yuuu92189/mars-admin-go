import { act, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/components/providers/toast-provider";
import { ChatView } from "./chat-view";

const mocks = vi.hoisted(() => ({
  listChatUsers: vi.fn(),
  listDirectMessages: vi.fn(),
  markDirectMessagesRead: vi.fn(),
  sendDirectMessage: vi.fn(),
}));

vi.mock("@/components/auth/auth-provider", () => ({
  useAuth: () => ({
    user: {
      user: { id: "me" },
    },
  }),
}));

vi.mock("@/lib/api/messaging", () => ({
  listChatUsers: mocks.listChatUsers,
  listDirectMessages: mocks.listDirectMessages,
  markDirectMessagesRead: mocks.markDirectMessagesRead,
  sendDirectMessage: mocks.sendDirectMessage,
}));

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  constructor(public url: string) {
    FakeWebSocket.instances.push(this);
  }

  emit(payload: unknown) {
    this.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent);
  }
}

describe("ChatView realtime routing", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    mocks.listChatUsers.mockReset();
    mocks.listDirectMessages.mockReset();
    mocks.markDirectMessagesRead.mockReset();
    mocks.sendDirectMessage.mockReset();
    mocks.listChatUsers.mockResolvedValue([
      {
        id: "peer-a",
        username: "alice",
        displayName: "Alice",
        online: true,
      },
      {
        id: "peer-b",
        username: "bob",
        displayName: "Bob",
        online: false,
      },
    ]);
    mocks.listDirectMessages.mockResolvedValue([]);
    mocks.markDirectMessagesRead.mockResolvedValue(undefined);
    vi.stubGlobal("WebSocket", FakeWebSocket);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not append another contact's realtime message to the selected conversation", async () => {
    render(
      <ToastProvider>
        <ChatView />
      </ToastProvider>,
    );

    expect(await screen.findByRole("button", { name: /Alice/ })).toBeVisible();
    await waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    const socket = FakeWebSocket.instances[0]!;

    act(() => {
      socket.emit({
        type: "chat",
        message: {
          id: "foreign-message",
          senderId: "peer-b",
          receiverId: "me",
          content: "other conversation",
          messageType: "text",
        },
      });
      socket.emit({
        type: "chat",
        message: {
          id: "selected-message",
          senderId: "peer-a",
          receiverId: "me",
          content: "selected conversation",
          messageType: "text",
        },
      });
    });

    expect(await screen.findByText("selected conversation")).toBeVisible();
    expect(screen.queryByText("other conversation")).not.toBeInTheDocument();
  });
});
