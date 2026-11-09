import { apiRequest } from "./client";
import type { StoredFile, StoredFilePage } from "./types";

export function listFiles(
  params: {
    page?: number;
    pageSize?: number;
    keyword?: string;
    groupId?: string;
  } = {},
): Promise<StoredFilePage> {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.groupId) search.set("groupId", params.groupId);
  const suffix = search.toString();
  return apiRequest<StoredFilePage>(`/api/v1/files${suffix ? `?${suffix}` : ""}`);
}

export function uploadFile(
  file: File,
  groupId?: string,
): Promise<{ file: StoredFile; url: string }> {
  const form = new FormData();
  form.append("file", file);
  if (groupId) form.append("groupId", groupId);
  return apiRequest<{ file: StoredFile; url: string }>("/api/v1/files", {
    method: "POST",
    body: form,
  });
}

export function getFile(id: string): Promise<StoredFile> {
  return apiRequest<StoredFile>(`/api/v1/files/${encodeURIComponent(id)}`);
}

export function deleteFile(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/files/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function fileDownloadUrl(id: string): string {
  return `/api/v1/files/${encodeURIComponent(id)}/download`;
}
