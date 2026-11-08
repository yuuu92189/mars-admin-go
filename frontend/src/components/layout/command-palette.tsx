"use client";

import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Search, X } from "lucide-react";
import { createPortal } from "react-dom";
import { Avatar } from "@/components/ui/surface";
import { AppIcon } from "@/components/ui/app-icon";
import { useAuth } from "@/components/auth/auth-provider";
import { useTheme } from "@/components/providers/theme-provider";
import { hasPermission } from "@/lib/permissions";
import { readRecentPages } from "@/lib/recent-pages";
import { searchUsers } from "@/lib/api/users";
import { lockDocumentScroll } from "@/lib/scroll-lock";
import type { User } from "@/lib/api/types";
import type { NavGroup } from "./nav";

type CommandEntry = {
  key: string;
  group: string;
  label: string;
  sub?: string;
  icon?: string;
  avatar?: string;
  run: () => void;
};

const ASK_PRESETS = ["当前账号有哪些权限", "如何快速切换页面", "如何修改密码"];

function localAsk(question: string, user: ReturnType<typeof useAuth>["user"], groups: NavGroup[]) {
  const permissions = user?.permissions?.length ?? 0;
  const pages = groups.flatMap((group) => group.items);
  if (/权限|permission/i.test(question)) {
    return `当前账号载入 ${permissions} 个权限码。前端只用于隐藏不可用入口，最终授权仍由 Go API 校验。`;
  }
  if (/页面|菜单|跳转|搜索/i.test(question)) {
    return `当前可访问 ${pages.length} 个页面。按 Ctrl/⌘ K 打开命令面板，输入页面名称后用方向键选择、Enter 打开。`;
  }
  if (/密码|password/i.test(question)) {
    return "从用户菜单或命令面板打开“修改密码”。提交当前密码和新密码后，Go 会撤销当前会话并要求重新登录。";
  }
  return "这是本地规则分析，不会发送问题到外部 AI 服务。可询问权限、页面搜索、修改密码或退出登录。";
}

export function CommandPalette({
  open,
  groups,
  onClose,
  onAppearance,
  onPassword,
  onLogout,
}: {
  open: boolean;
  groups: NavGroup[];
  onClose: () => void;
  onAppearance: () => void;
  onPassword: () => void;
  onLogout: () => void;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuth();
  const { resolvedTheme, setTheme } = useTheme();
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const [users, setUsers] = useState<User[]>([]);
  const [recentPaths, setRecentPaths] = useState<string[]>([]);
  const [askAnswer, setAskAnswer] = useState<string | null>(null);
  const [searchingUsers, setSearchingUsers] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);

  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  const pages = useMemo(
    () =>
      groups.flatMap((group) =>
        group.items.map((item) => ({
          ...item,
          group: group.label === "导航" ? "" : group.label,
        })),
      ),
    [groups],
  );
  const usersPath = pages.find((page) => page.path === "/system/user")?.path;
  const canSearchUsers = Boolean(usersPath) && hasPermission(user, "user:read");
  const askMode = query.startsWith("?") || query.startsWith("？");

  useEffect(() => {
    if (!open) return;
    previousFocus.current = document.activeElement as HTMLElement | null;
    const allowed = new Set(pages.map((page) => page.path));
    /* eslint-disable react-hooks/set-state-in-effect */
    setQuery("");
    setActiveIndex(0);
    setUsers([]);
    setAskAnswer(null);
    setRecentPaths(readRecentPages(localStorage).filter((path) => allowed.has(path)));
    /* eslint-enable react-hooks/set-state-in-effect */
    const frame = requestAnimationFrame(() => inputRef.current?.focus());
    const releaseScroll = lockDocumentScroll();
    return () => {
      cancelAnimationFrame(frame);
      releaseScroll();
      previousFocus.current?.focus();
    };
  }, [open, pages]);

  useEffect(() => {
    if (!open) return;
    const onDocumentKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setQuery("");
      setUsers([]);
      setAskAnswer(null);
      onCloseRef.current();
    };
    document.addEventListener("keydown", onDocumentKeyDown);
    return () => document.removeEventListener("keydown", onDocumentKeyDown);
  }, [open]);

  useEffect(() => {
    if (!open || askMode || !canSearchUsers || query.trim().length < 2) return;
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      setSearchingUsers(true);
      try {
        const page = await searchUsers(query, controller.signal);
        setUsers((page.items ?? []).slice(0, 5));
      } catch {
        if (!controller.signal.aborted) setUsers([]);
      } finally {
        if (!controller.signal.aborted) setSearchingUsers(false);
      }
    }, 220);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [askMode, canSearchUsers, open, query]);

  const entries = useMemo<CommandEntry[]>(() => {
    const key = query.trim().toLowerCase();
    const result: CommandEntry[] = [];
    if (!askMode && !key) {
      recentPaths
        .filter((path) => path !== pathname)
        .map((path) => pages.find((page) => page.path === path))
        .filter((page): page is NonNullable<typeof page> => Boolean(page))
        .slice(0, 4)
        .forEach((page) =>
          result.push({
            key: `recent-${page.key}`,
            group: "最近访问",
            label: page.label,
            sub: page.group,
            icon: page.icon,
            run: () => go(page.path),
          }),
        );
    }
    if (!askMode) {
      pages
        .filter(
          (page) =>
            !key ||
            page.label.toLowerCase().includes(key) ||
            page.group.toLowerCase().includes(key) ||
            page.path.toLowerCase().includes(key),
        )
        .forEach((page) =>
          result.push({
            key: `page-${page.key}`,
            group: "页面",
            label: page.label,
            sub: page.group,
            icon: page.icon,
            run: () => go(page.path),
          }),
        );

      if (key && canSearchUsers) {
        users.forEach((item) =>
          result.push({
            key: `user-${item.id ?? item.username}`,
            group: "用户",
            label: item.displayName,
            sub: `@${item.username}`,
            avatar: item.displayName,
            run: () => go(usersPath ?? "/dashboard", { keyword: item.username }),
          }),
        );
      }

      const actions: CommandEntry[] = [
        {
          key: "action-theme",
          group: "动作",
          label: resolvedTheme === "dark" ? "切换到浅色外观" : "切换到深色外观",
          icon: resolvedTheme === "dark" ? "sun" : "moon",
          run: () => {
            setTheme(resolvedTheme === "dark" ? "light" : "dark");
            close();
          },
        },
        {
          key: "action-appearance",
          group: "动作",
          label: "外观设置",
          sub: "主题与表格密度",
          icon: "contrast",
          run: () => act(onAppearance),
        },
        {
          key: "action-password",
          group: "动作",
          label: "修改密码",
          icon: "key",
          run: () => act(onPassword),
        },
        {
          key: "action-logout",
          group: "动作",
          label: "退出登录",
          icon: "logout",
          run: () => act(onLogout),
        },
      ];
      actions
        .filter(
          (action) =>
            !key ||
            action.label.toLowerCase().includes(key) ||
            action.sub?.toLowerCase().includes(key),
        )
        .forEach((action) => result.push(action));
    }
    return result;

    function go(path: string, params?: Record<string, string>) {
      close();
      const search = params ? `?${new URLSearchParams(params).toString()}` : "";
      router.push(`${path}${search}`);
    }

    function act(callback: () => void) {
      close();
      callback();
    }

    function close() {
      setQuery("");
      setUsers([]);
      setAskAnswer(null);
      onClose();
    }
  }, [
    askMode,
    canSearchUsers,
    onAppearance,
    onClose,
    onLogout,
    onPassword,
    pages,
    pathname,
    query,
    recentPaths,
    resolvedTheme,
    router,
    setTheme,
    users,
    usersPath,
  ]);

  const grouped = useMemo(() => {
    const groupsResult: Array<{ name: string; items: Array<CommandEntry & { index: number }> }> =
      [];
    entries.forEach((entry, index) => {
      let group = groupsResult.find((item) => item.name === entry.group);
      if (!group) {
        group = { name: entry.group, items: [] };
        groupsResult.push(group);
      }
      group.items.push({ ...entry, index });
    });
    return groupsResult;
  }, [entries]);

  if (!open || typeof document === "undefined") return null;

  const effectiveActive = Math.min(activeIndex, Math.max(entries.length - 1, 0));

  function close() {
    setQuery("");
    setUsers([]);
    setAskAnswer(null);
    onClose();
  }

  function runAsk() {
    setAskAnswer(localAsk(query.replace(/^[?？]/, "").trim(), user, groups));
  }

  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Tab") {
      const focusable = Array.from(
        panelRef.current?.querySelectorAll<HTMLElement>(
          'button:not([disabled]):not([tabindex="-1"]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ) ?? [],
      );
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
      return;
    }
    if (askMode) {
      if (event.key === "Enter") {
        event.preventDefault();
        runAsk();
      }
      return;
    }
    if (!entries.length) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((effectiveActive + 1) % entries.length);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((effectiveActive - 1 + entries.length) % entries.length);
    } else if (event.key === "Enter") {
      event.preventDefault();
      entries[effectiveActive]?.run();
    }
  }

  return createPortal(
    <div
      className="overlay cmd-overlay"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) close();
      }}
    >
      <div
        ref={panelRef}
        className="cmd-panel m-float"
        role="dialog"
        aria-modal="true"
        aria-label="命令面板"
        onKeyDown={onKeyDown}
      >
        <div className="cmd-input">
          <Search size={18} />
          <input
            ref={inputRef}
            value={query}
            role="combobox"
            aria-label="搜索页面、用户或动作"
            aria-expanded="true"
            aria-controls="cmd-results"
            aria-activedescendant={
              !askMode && entries.length ? `cmd-entry-${effectiveActive}` : undefined
            }
            placeholder="搜索页面、用户或动作，输入 ? 问系统"
            onChange={(event) => {
              const value = event.target.value;
              setQuery(value);
              setActiveIndex(0);
              setAskAnswer(null);
              if (value.trim().length < 2) setUsers([]);
            }}
          />
          <button type="button" className="icon-btn" aria-label="关闭" onClick={close}>
            <X size={16} />
          </button>
        </div>
        <div id="cmd-results" className="cmd-results" role={askMode ? "region" : "listbox"}>
          {askMode ? (
            <div className="cmd-ask">
              {!askAnswer ? (
                <>
                  <div className="ask-presets">
                    {ASK_PRESETS.map((preset) => (
                      <button
                        key={preset}
                        type="button"
                        onClick={() => {
                          setQuery(`?${preset}`);
                          setAskAnswer(localAsk(preset, user, groups));
                        }}
                      >
                        {preset}
                      </button>
                    ))}
                  </div>
                  <p className="cmd-ask-hint faint">输入问题后回车，分析在本地完成</p>
                </>
              ) : (
                <div className="cmd-ask-results" aria-live="polite">
                  <div className="ask-results-head">
                    <span>
                      <AppIcon name="activity" size={13} />
                      {query.replace(/^[?？]/, "").trim()}
                    </span>
                    <small>本地规则分析</small>
                  </div>
                  <article className="ask-card">
                    <h4>系统回答</h4>
                    <p>{askAnswer}</p>
                  </article>
                </div>
              )}
            </div>
          ) : (
            <>
              {grouped.map((group) => (
                <div key={group.name}>
                  <div className="cmd-group eyebrow" role="presentation">
                    {group.name}
                  </div>
                  {group.items.map((item) => (
                    <button
                      id={`cmd-entry-${item.index}`}
                      key={item.key}
                      type="button"
                      role="option"
                      aria-selected={item.index === effectiveActive}
                      className={`cmd-item ${item.index === effectiveActive ? "is-active" : ""}`}
                      tabIndex={-1}
                      onPointerEnter={() => setActiveIndex(item.index)}
                      onClick={item.run}
                    >
                      {item.avatar ? (
                        <Avatar name={item.avatar} size={30} />
                      ) : (
                        <span className="cmd-ico m-groove">
                          <AppIcon name={item.icon ?? "file"} size={16} />
                        </span>
                      )}
                      <span className="cmd-label">{item.label}</span>
                      {item.sub ? <span className="cmd-sub">{item.sub}</span> : null}
                      <AppIcon className="cmd-go" name="arrowRight" size={14} />
                    </button>
                  ))}
                </div>
              ))}
              {!entries.length ? (
                <div className="select-empty" style={{ padding: 28 }}>
                  {searchingUsers ? "正在搜索…" : "没有找到匹配的结果"}
                </div>
              ) : null}
            </>
          )}
        </div>
        <div className="cmd-foot">
          {askMode ? (
            <>
              <span>
                <span className="kbd">↵</span> 提问
              </span>
              <span className="faint">删去 ? 返回搜索</span>
            </>
          ) : (
            <>
              <span>
                <span className="kbd">↑</span>
                <span className="kbd">↓</span> 选择
              </span>
              <span>
                <span className="kbd">↵</span> 打开
              </span>
              <span>
                <span className="kbd">?</span> 问系统
              </span>
            </>
          )}
          <span className="spacer" />
          <span className="eyebrow">命令</span>
        </div>
      </div>
    </div>,
    document.body,
  );
}
