import type { CurrentUser, Menu } from "@/lib/api/types";

export function normalizeMenuPath(path?: string): string {
  if (!path) return "";
  const prefixed = path.startsWith("/") ? path : `/${path}`;
  return prefixed.length > 1 ? prefixed.replace(/\/+$/, "") : prefixed;
}

function flattenMenus(menus: Menu[] = []): Menu[] {
  return menus.flatMap((menu) => [menu, ...flattenMenus(menu.children)]);
}

export function isVisibleMenuPath(user: CurrentUser | null, pathname: string): boolean {
  if (!user) return false;
  const normalized = normalizeMenuPath(pathname);
  return flattenMenus(user.menus).some(
    (menu) => menu.visible && menu.type === 2 && normalizeMenuPath(menu.path) === normalized,
  );
}
