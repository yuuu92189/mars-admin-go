"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { ChevronDown, Search } from "lucide-react";
import { Avatar } from "@/components/ui/surface";
import { IconButton } from "@/components/ui/button";
import { Dropdown } from "@/components/ui/overlay";
import { AppIcon } from "@/components/ui/app-icon";
import { useAuth } from "@/components/auth/auth-provider";
import { useTheme } from "@/components/providers/theme-provider";
import type { NavGroup } from "./nav";

export function Header({
  groups,
  onToggleRail,
  onCommand,
  onAppearance,
}: {
  groups: NavGroup[];
  onToggleRail: () => void;
  onCommand: () => void;
  onAppearance: () => void;
}) {
  const { user, logout } = useAuth();
  const { theme, cycleTheme } = useTheme();
  const router = useRouter();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const current = useMemo(
    () =>
      groups
        .flatMap((group) => group.items.map((item) => ({ ...item, group: group.label })))
        .find((item) => pathname === item.path || pathname.startsWith(`${item.path}/`)),
    [groups, pathname],
  );

  useEffect(() => {
    if (!menuOpen) return;
    const close = (event: PointerEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) setMenuOpen(false);
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [menuOpen]);

  return (
    <header className="topbar">
      <IconButton label="切换导航" icon="sidebar" onClick={onToggleRail} />
      <div className="crumbs" aria-label="当前位置">
        {current?.group ? (
          <>
            <span>{current.group}</span>
            <AppIcon className="crumb-sep" name="chevronRight" size={12} />
          </>
        ) : null}
        <span className="crumb-current">{current?.label ?? "工作台"}</span>
      </div>
      <div className="spacer" />
      <button
        type="button"
        className="cmd-trigger m-pill"
        aria-label="搜索菜单"
        onClick={onCommand}
      >
        <Search size={15} />
        <span className="cmd-text">搜索菜单</span>
        <span className="kbd">Ctrl K</span>
      </button>
      <button
        type="button"
        className="icon-btn"
        aria-label="切换主题"
        title={`主题：${theme}`}
        onClick={cycleTheme}
      >
        <AppIcon name={theme === "dark" ? "sun" : "moon"} size={17} />
      </button>
      <div className="user-chip-wrap" ref={menuRef}>
        <button
          type="button"
          className="user-chip"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((open) => !open)}
        >
          <Avatar name={user?.user?.displayName ?? user?.user?.username} size={32} />
          <span className="user-meta">
            <span className="user-name">{user?.user?.displayName ?? user?.user?.username}</span>
            <span className="user-role">{user?.roles?.[0] ?? "管理员"}</span>
          </span>
          <ChevronDown size={14} style={{ color: "var(--ink-soft)" }} />
        </button>
        <Dropdown open={menuOpen} className="user-menu">
          <div className="menu-head">
            <div style={{ fontWeight: 680 }}>{user?.user?.displayName ?? user?.user?.username}</div>
            <div className="soft" style={{ fontSize: 12 }}>
              @{user?.user?.username}
            </div>
          </div>
          <button
            type="button"
            className="menu-item"
            role="menuitem"
            onClick={() => {
              setMenuOpen(false);
              onAppearance();
            }}
          >
            <AppIcon name="contrast" size={16} /> 外观设置
          </button>
          <button
            type="button"
            className="menu-item"
            role="menuitem"
            onClick={() => {
              setMenuOpen(false);
              router.push("/settings/profile");
            }}
          >
            <AppIcon name="key" size={16} /> 个人资料
          </button>
          <button
            type="button"
            className="menu-item is-danger"
            role="menuitem"
            onClick={() => void logout()}
          >
            <AppIcon name="logout" size={16} /> 退出登录
          </button>
        </Dropdown>
      </div>
    </header>
  );
}
