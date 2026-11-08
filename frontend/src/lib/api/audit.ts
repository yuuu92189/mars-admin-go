import { apiRequest } from "./client";
import type { AuditEventPage, LoginEventPage } from "./types";

export type AuditQuery = {
  page?: number;
  pageSize?: number;
  action?: string;
  resourceType?: string;
  from?: string;
  to?: string;
};

function queryString(query: AuditQuery): string {
  const params = new URLSearchParams();
  params.set("page", String(query.page ?? 1));
  params.set("pageSize", String(query.pageSize ?? 20));
  if (query.action?.trim()) params.set("action", query.action.trim());
  if (query.resourceType?.trim()) params.set("resourceType", query.resourceType.trim());
  if (query.from) params.set("from", query.from);
  if (query.to) params.set("to", query.to);
  return params.toString();
}

export function listAuditEvents(query: AuditQuery = {}): Promise<AuditEventPage> {
  return apiRequest<AuditEventPage>(`/api/v1/admin/audit/events?${queryString(query)}`);
}

export function listLoginEvents(page = 1, pageSize = 20): Promise<LoginEventPage> {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  return apiRequest<LoginEventPage>(`/api/v1/admin/audit/logins?${params.toString()}`);
}
