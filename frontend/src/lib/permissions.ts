import type { CurrentUser } from "./api/types";

export function hasPermission(
  user: Pick<CurrentUser, "permissions"> | null | undefined,
  permission: string,
): boolean {
  return Boolean(user?.permissions?.includes(permission));
}

export function hasAnyPermission(
  user: Pick<CurrentUser, "permissions"> | null | undefined,
  permissions: string[],
): boolean {
  return permissions.some((permission) => hasPermission(user, permission));
}

export function hasAllPermissions(
  user: Pick<CurrentUser, "permissions"> | null | undefined,
  permissions: string[],
): boolean {
  return permissions.every((permission) => hasPermission(user, permission));
}
