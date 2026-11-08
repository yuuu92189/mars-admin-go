import type { CurrentUser, Menu } from "@/lib/api/types";
import { normalizeMenuPath } from "@/lib/navigation";

export type NavItem = {
  key: string;
  label: string;
  path: string;
  icon: string;
};

export type NavGroup = {
  key: string;
  label: string;
  items: NavItem[];
};

function menuToItem(menu: Menu): NavItem {
  return {
    key: menu.id,
    label: menu.name,
    path: normalizeMenuPath(menu.path) || "/dashboard",
    icon: normalizeIcon(menu.icon),
  };
}

function normalizeIcon(icon?: string): string {
  if (!icon) return "dashboard";
  const aliases: Record<string, string> = {
    PersonOutline: "users",
    PeopleOutline: "role",
    MenuOutline: "menu",
    BookOutline: "dict",
    SettingsSharp: "settings",
    FolderOutline: "file",
    GitNetworkOutline: "dept",
    IdCardOutline: "post",
    NotificationsOutline: "notice",
    ChatbubbleOutline: "message",
    LogInOutline: "audit",
    ListOutline: "audit",
    TimerOutline: "clock",
    DesktopOutline: "monitor",
    DocumentTextOutline: "workflow",
    CheckboxOutline: "workflow",
    CheckmarkDoneOutline: "workflow",
    AnalyticsOutline: "chart",
  };
  return aliases[icon] ?? icon;
}

export function buildNavGroups(user: CurrentUser): NavGroup[] {
  const groups: NavGroup[] = [];
  for (const menu of user.menus ?? []) {
    if (!menu.visible) continue;
    if (menu.type === 1 && menu.children?.length) {
      const items = menu.children
        .filter((child) => child.visible && child.type === 2)
        .map(menuToItem);
      if (items.length) groups.push({ key: menu.id, label: menu.name, items });
      continue;
    }
    if (menu.type === 2) {
      const existing = groups.find((group) => group.key === "root");
      if (existing) existing.items.push(menuToItem(menu));
      else groups.push({ key: "root", label: "导航", items: [menuToItem(menu)] });
    }
  }
  return groups;
}
