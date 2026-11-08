import { apiRequest } from "./client";
import type { UserPage } from "./types";

export function searchUsers(keyword: string, signal?: AbortSignal): Promise<UserPage> {
  const query = new URLSearchParams({
    keyword: keyword.trim(),
    page: "1",
    pageSize: "5",
  });
  return apiRequest<UserPage>(`/api/v1/admin/users?${query.toString()}`, { signal });
}
