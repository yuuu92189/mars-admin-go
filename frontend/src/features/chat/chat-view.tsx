"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { Wifi, WifiOff } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import { PageContainer } from "@/components/layout/page-container";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { Field, TextInput } from "@/components/ui/input";
import { Avatar, Card, EmptyState, PageHeader, Tag } from "@/components/ui/surface";
import { ApiError } from "@/lib/api/client";
import {
  listChatUsers,
  listDirectMessages,
  markDirectMessagesRead,
  sendDirectMessage,
} from "@/lib/api/messaging";
import type { ChatMessage, ChatUser } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";
import styles from "./chat-view.module.css";
import { isDirectMessageForPeer, mergeDirectMessageHistory } from "./realtime";

type RealtimeEvent = {
  type?: string;
  message?: ChatMessage;
};

function messageTime(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", hour12: false });
}

function websocketUrl(): string {
  const configured = process.env.NEXT_PUBLIC_REALTIME_URL;
  if (configured) {
    const base = configured.replace(/^http/, "ws").replace(/\/+$/, "");
    return `${base}/ws/message`;
  }
  if (typeof window === "undefined") return "";
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws/message`;
}

export function ChatView() {
  const { user } = useAuth();
  const { pushToast } = useToast();
  const [users, setUsers] = useState<ChatUser[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [content, setContent] = useState("");
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const selectedIdRef = useRef<string | null>(null);
  const messagesPeerIdRef = useRef<string | null>(null);
  const selected = users.find((item) => item.id === selectedId) ?? null;
  const currentUserId = user?.user?.id;
  const visibleUsers = useMemo(() => {
    const normalized = query.trim().toLocaleLowerCase();
    if (!normalized) return users;
    return users.filter((item) =>
      [item.displayName, item.username].some((value) =>
        value.toLocaleLowerCase().includes(normalized),
      ),
    );
  }, [query, users]);

  const loadUsers = useCallback(async () => {
    try {
      const result = await listChatUsers();
      setUsers(result);
      setSelectedId((current) => current ?? result[0]?.id ?? null);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : "联系人加载失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, []);

  const loadMessages = useCallback(
    async (peerId: string) => {
      try {
        const result = await listDirectMessages(peerId, { limit: 50 });
        const samePeer = messagesPeerIdRef.current === peerId;
        setMessages((current) =>
          samePeer ? mergeDirectMessageHistory(current, result, currentUserId, peerId) : result,
        );
        messagesPeerIdRef.current = peerId;
        await markDirectMessagesRead(peerId).catch(() => undefined);
      } catch (cause) {
        pushToast({
          title: "消息加载失败",
          message: cause instanceof ApiError ? cause.message : "请稍后重试",
          tone: "danger",
        });
      }
    },
    [currentUserId, pushToast],
  );

  useDeferredEffect(loadUsers);

  const loadSelectedMessages = useCallback(() => {
    if (selectedId) return loadMessages(selectedId);
    return Promise.resolve();
  }, [loadMessages, selectedId]);

  useDeferredEffect(loadSelectedMessages);

  useEffect(() => {
    selectedIdRef.current = selectedId;
  }, [selectedId]);

  useEffect(() => {
    if (!selectedId) return;
    const timer = window.setInterval(() => {
      void loadMessages(selectedId);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [loadMessages, selectedId]);

  useEffect(() => {
    if (!currentUserId) return;
    const url = websocketUrl();
    if (!url) return;
    let closed = false;
    let retryTimer: number | null = null;
    let retryDelay = 1000;

    const connect = () => {
      if (closed) return;
      const socket = new WebSocket(url);
      socketRef.current = socket;
      socket.onopen = () => {
        retryDelay = 1000;
        setConnected(true);
        socket.send(JSON.stringify({ type: "ping" }));
      };
      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(String(event.data)) as RealtimeEvent;
          if (
            payload.type === "chat" &&
            payload.message &&
            isDirectMessageForPeer(payload.message, currentUserId, selectedIdRef.current)
          ) {
            const incoming = payload.message;
            setMessages((current) => {
              if (current.some((message) => message.id === incoming.id)) return current;
              return [incoming, ...current];
            });
          }
        } catch {
          // Polling remains the fallback for malformed or unknown events.
        }
      };
      socket.onerror = () => setConnected(false);
      socket.onclose = () => {
        setConnected(false);
        if (!closed) {
          retryTimer = window.setTimeout(connect, retryDelay);
          retryDelay = Math.min(retryDelay * 2, 15000);
        }
      };
    };

    connect();
    return () => {
      closed = true;
      if (retryTimer !== null) window.clearTimeout(retryTimer);
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [currentUserId]);

  async function send(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected || !content.trim() || sending) return;
    setSending(true);
    try {
      const created = await sendDirectMessage({
        receiverId: selected.id,
        content: content.trim(),
      });
      setMessages((current) =>
        current.some((message) => message.id === created.id) ? current : [created, ...current],
      );
      setContent("");
    } catch (cause) {
      pushToast({
        title: "消息发送失败",
        message: cause instanceof ApiError ? cause.message : "请稍后重试",
        tone: "danger",
      });
    } finally {
      setSending(false);
    }
  }

  return (
    <PageContainer>
      <PageHeader
        title="即时聊天"
        description="私聊消息由 Go Messaging API 持久化；同源 WebSocket 可用时实时推送，否则自动轮询。"
      >
        <Tag tone={connected ? "ok" : "neutral"} dot>
          {connected ? <Wifi size={13} /> : <WifiOff size={13} />}
          {connected ? "实时连接" : "轮询模式"}
        </Tag>
      </PageHeader>
      <Card className={styles.chatShell}>
        <aside className={styles.contacts} aria-label="联系人">
          <div className={styles.contactHead}>
            <span className="section-label">联系人</span>
            <span className="faint">{users.length}</span>
          </div>
          <TextInput
            icon="search"
            placeholder="搜索联系人"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <div className={styles.contactList}>
            {visibleUsers.map((item) => (
              <button
                type="button"
                key={item.id}
                className={`${styles.contact} ${item.id === selectedId ? styles.active : ""}`}
                onClick={() => setSelectedId(item.id)}
              >
                <span className={styles.avatarWrap}>
                  {item.avatarUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={item.avatarUrl} alt="" />
                  ) : (
                    <Avatar name={item.displayName || item.username} size={38} />
                  )}
                  <i className={item.online ? styles.online : styles.offline} />
                </span>
                <span className={styles.contactCopy}>
                  <strong>{item.displayName || item.username}</strong>
                  <small>@{item.username}</small>
                </span>
              </button>
            ))}
          </div>
        </aside>
        <section className={styles.conversation} aria-live="polite">
          {selected ? (
            <>
              <div className={styles.conversationHead}>
                <div>
                  <strong>{selected.displayName || selected.username}</strong>
                  <span>@{selected.username}</span>
                </div>
                <Tag tone={selected.online ? "ok" : "neutral"} dot>
                  {selected.online ? "在线" : "离线"}
                </Tag>
              </div>
              <div className={styles.messageList}>
                {messages.length ? (
                  [...messages].reverse().map((message) => {
                    const own = message.senderId === currentUserId;
                    return (
                      <div
                        className={`${styles.messageRow} ${own ? styles.own : ""}`}
                        key={message.id}
                      >
                        {!own ? (
                          <Avatar
                            name={message.senderName || selected.displayName || selected.username}
                            size={28}
                          />
                        ) : null}
                        <div className={styles.bubble}>
                          <p>{message.content}</p>
                          <span>{messageTime(message.createdAt)}</span>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <EmptyState
                    title="还没有消息"
                    description="发送第一条消息开始对话。"
                    icon="message"
                  />
                )}
              </div>
              <form className={styles.composer} onSubmit={send}>
                <Field label="消息">
                  <TextInput
                    value={content}
                    placeholder="输入消息，按 Enter 发送"
                    onChange={(event) => setContent(event.target.value)}
                  />
                </Field>
                <Button
                  type="submit"
                  variant="primary"
                  icon="send"
                  loading={sending}
                  disabled={!content.trim()}
                >
                  发送
                </Button>
              </form>
            </>
          ) : (
            <EmptyState
              title={loading ? "正在读取联系人" : "选择一个联系人"}
              description={error ?? "从左侧联系人列表开始聊天。"}
              icon="message"
            />
          )}
        </section>
      </Card>
    </PageContainer>
  );
}
