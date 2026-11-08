import { apiRequest } from "./client";
import type { CreateUser, UpdateUser, User, UserPage } from "./types";

export type UserQuery = {
  page?: number;
  pageSize?: number;
  keyword?: string;
  status?: string;
  userType?: string;
  sort?: string;
  departmentId?: string | null;
  positionId?: string | null;
};

function userQueryString(query: UserQuery): string {
  const params = new URLSearchParams();
  params.set("page", String(query.page ?? 1));
  params.set("pageSize", String(query.pageSize ?? 20));
  if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
  if (query.status) params.set("status", query.status);
  if (query.userType) params.set("userType", query.userType);
  if (query.sort) params.set("sort", query.sort);
  if (query.departmentId) params.set("departmentId", query.departmentId);
  if (query.positionId) params.set("positionId", query.positionId);
  return params.toString();
}

export function listUsers(query: UserQuery = {}, signal?: AbortSignal): Promise<UserPage> {
  return apiRequest<UserPage>(`/api/v1/admin/users?${userQueryString(query)}`, { signal });
}

export function searchUsers(keyword: string, signal?: AbortSignal): Promise<UserPage> {
  return listUsers({ keyword, page: 1, pageSize: 5 }, signal);
}

export function getUser(id: string): Promise<User> {
  return apiRequest<User>(`/api/v1/admin/users/${encodeURIComponent(id)}`);
}

export function createUser(input: CreateUser): Promise<User> {
  return apiRequest<User>("/api/v1/admin/users", { method: "POST", body: input });
}

export function updateUser(id: string, input: UpdateUser): Promise<User> {
  return apiRequest<User>(`/api/v1/admin/users/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteUser(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/users/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function resetUserPassword(id: string): Promise<{ temporaryPassword: string }> {
  return apiRequest<{ temporaryPassword: string }>(
    `/api/v1/admin/users/${encodeURIComponent(id)}/reset-password`,
    { method: "POST" },
  );
}

export function revokeUserSessions(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/users/${encodeURIComponent(id)}/revoke-sessions`, {
    method: "POST",
  });
}
