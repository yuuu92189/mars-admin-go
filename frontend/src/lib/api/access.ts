import { apiRequest } from "./client";
import type { CreateRole, Menu, MenuInput, Permission, Role, RolePage, UpdateRole } from "./types";

export type RoleQuery = {
  page?: number;
  pageSize?: number;
  keyword?: string;
};

function roleQueryString(query: RoleQuery): string {
  const params = new URLSearchParams();
  params.set("page", String(query.page ?? 1));
  params.set("pageSize", String(query.pageSize ?? 20));
  if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
  return params.toString();
}

export function listRoles(query: RoleQuery = {}): Promise<RolePage> {
  return apiRequest<RolePage>(`/api/v1/admin/access/roles?${roleQueryString(query)}`);
}

export function createRole(input: CreateRole): Promise<Role> {
  return apiRequest<Role>("/api/v1/admin/access/roles", { method: "POST", body: input });
}

export function updateRole(id: string, input: UpdateRole): Promise<Role> {
  return apiRequest<Role>(`/api/v1/admin/access/roles/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteRole(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/access/roles/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function listPermissions(): Promise<Permission[]> {
  return apiRequest<Permission[]>("/api/v1/admin/access/permissions");
}

export function listMenus(): Promise<Menu[]> {
  return apiRequest<Menu[]>("/api/v1/admin/access/menus");
}

export function getRoleMenus(id: string): Promise<string[]> {
  return apiRequest<string[]>(`/api/v1/admin/access/roles/${encodeURIComponent(id)}/menus`);
}

export function setRoleMenus(id: string, menuIds: string[]): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/access/roles/${encodeURIComponent(id)}/menus`, {
    method: "PUT",
    body: { menuIds },
  });
}

export function setRolePermissions(id: string, permissions: string[]): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/access/roles/${encodeURIComponent(id)}/permissions`, {
    method: "PUT",
    body: { permissions },
  });
}

export function createMenu(input: MenuInput): Promise<Menu> {
  return apiRequest<Menu>("/api/v1/admin/access/menus", { method: "POST", body: input });
}

export function updateMenu(id: string, input: MenuInput): Promise<Menu> {
  return apiRequest<Menu>(`/api/v1/admin/access/menus/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteMenu(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/access/menus/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
