"use client";

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { ChevronDown, Search } from "lucide-react";
import { Avatar } from "@/components/ui/surface";
import { IconButton } from "@/components/ui/button";
import { Dropdown } from "@/components/ui/overlay";
import { AppIcon } from "@/components/ui/app-icon";
import { useAuth } from "@/components/auth/auth-provider";
import type { NavGroup } from "./nav";

export function Header({
  groups,
  onToggleRail,
  onCommand,
  onAppearance,
  onPassword,
  onLogout,
}: {
  groups: NavGroup[];
  onToggleRail: () => void;
  onCommand: () => void;
  onAppearance: () => void;
  onPassword: () => void;
  onLogout: () => void;
}) {
  const { user } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const chipRef = useRef<HTMLButtonElement>(null);

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

  useEffect(() => {
    if (!menuOpen) return;
    requestAnimationFrame(() => {
      menuRef.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
    });
  }, [menuOpen]);

  function onMenuKeyDown(event: ReactKeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      setMenuOpen(false);
      chipRef.current?.focus();
      return;
    }
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    event.preventDefault();
    const items = Array.from(
      menuRef.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [],
    );
    const index = items.indexOf(document.activeElement as HTMLElement);
    const next =
      items[(index + (event.key === "ArrowDown" ? 1 : -1) + items.length) % items.length];
    next?.focus();
  }

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
        aria-label="外观设置"
        title="外观设置"
        onClick={onAppearance}
      >
        <AppIcon name="contrast" size={17} />
      </button>
      <div className="user-chip-wrap" ref={menuRef} onKeyDown={onMenuKeyDown}>
        <button
          ref={chipRef}
          type="button"
          className="user-chip"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          aria-controls="user-menu"
          onClick={() => setMenuOpen((open) => !open)}
        >
          <Avatar name={user?.user?.displayName ?? user?.user?.username} size={32} />
          <span className="user-meta">
            <span className="user-name">{user?.user?.displayName ?? user?.user?.username}</span>
            <span className="user-role">{user?.roles?.[0] ?? "管理员"}</span>
          </span>
          <ChevronDown size={14} style={{ color: "var(--ink-soft)" }} />
        </button>
        <Dropdown id="user-menu" open={menuOpen} className="user-menu">
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
              router.push("/profile");
            }}
          >
            <AppIcon name="users" size={16} /> 个人中心
          </button>
          <button
            type="button"
            className="menu-item"
            role="menuitem"
            onClick={() => {
              setMenuOpen(false);
              onPassword();
            }}
          >
            <AppIcon name="key" size={16} /> 修改密码
          </button>
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
            className="menu-item is-danger"
            role="menuitem"
            onClick={() => {
              setMenuOpen(false);
              onLogout();
            }}
          >
            <AppIcon name="logout" size={16} /> 退出登录
          </button>
        </Dropdown>
      </div>
    </header>
  );
}
