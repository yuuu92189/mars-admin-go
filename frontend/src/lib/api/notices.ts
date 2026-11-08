import { apiRequest } from "./client";
import type { Notice, NoticeInput, NoticePage } from "./types";

export type NoticeQuery = {
  page?: number;
  pageSize?: number;
  status?: string;
};

function noticeQueryString(query: NoticeQuery): string {
  const params = new URLSearchParams();
  params.set("page", String(query.page ?? 1));
  params.set("pageSize", String(query.pageSize ?? 20));
  if (query.status) params.set("status", query.status);
  return params.toString();
}

export function listNotices(query: NoticeQuery = {}): Promise<NoticePage> {
  return apiRequest<NoticePage>(`/api/v1/notices?${noticeQueryString(query)}`);
}

export function listMyNotices(query: NoticeQuery = {}): Promise<NoticePage> {
  return apiRequest<NoticePage>(`/api/v1/notices/my?${noticeQueryString(query)}`);
}

export function getNotice(id: string): Promise<Notice> {
  return apiRequest<Notice>(`/api/v1/notices/${encodeURIComponent(id)}`);
}

export function createNotice(input: NoticeInput): Promise<Notice> {
  return apiRequest<Notice>("/api/v1/notices", { method: "POST", body: input });
}

export function updateNotice(id: string, input: NoticeInput): Promise<Notice> {
  return apiRequest<Notice>(`/api/v1/notices/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteNotice(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/notices/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function markNoticeRead(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/notices/${encodeURIComponent(id)}/read`, { method: "POST" });
}

export function publishNotice(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/notices/${encodeURIComponent(id)}/publish`, {
    method: "POST",
  });
}

export function withdrawNotice(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/notices/${encodeURIComponent(id)}/withdraw`, {
    method: "POST",
  });
}

export function unreadNoticeCount(): Promise<number> {
  return apiRequest<Record<string, number>>("/api/v1/notices/unread").then(
    (result) => result.unread ?? result.count ?? 0,
  );
}
