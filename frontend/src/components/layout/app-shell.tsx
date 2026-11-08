"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { usePathname } from "next/navigation";
import { AuthProvider, useAuth } from "@/components/auth/auth-provider";
import { PasswordDialog } from "@/components/auth/password-dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTheme } from "@/components/providers/theme-provider";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import { recordRecentPage } from "@/lib/recent-pages";
import { useShellEntrance } from "@/lib/motion";
import type { CurrentUser } from "@/lib/api/types";
import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { AppearanceDialog } from "./appearance-dialog";
import { CommandPalette } from "./command-palette";
import { buildNavGroups } from "./nav";

export function AppShell({ user, children }: { user: CurrentUser; children: ReactNode }) {
  return (
    <AuthProvider initialUser={user}>
      <ShellContent user={user}>{children}</ShellContent>
    </AuthProvider>
  );
}

function ShellContent({ user, children }: { user: CurrentUser; children: ReactNode }) {
  const groups = useMemo(() => buildNavGroups(user), [user]);
  const pathname = usePathname();
  const { logout } = useAuth();
  const { pushToast } = useToast();
  const { railCollapsed, setRailCollapsed } = useTheme();
  const [mobile, setMobile] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [commandOpen, setCommandOpen] = useState(false);
  const [appearanceOpen, setAppearanceOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [logoutOpen, setLogoutOpen] = useState(false);
  const [logoutLoading, setLogoutLoading] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const shellRef = useShellEntrance<HTMLDivElement>();

  useEffect(() => {
    const allowed = new Set(groups.flatMap((group) => group.items.map((item) => item.path)));
    recordRecentPage(localStorage, pathname, allowed);
    // A browser back/forward navigation must also dismiss the mobile drawer.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setMobileOpen(false);
  }, [groups, pathname]);

  useEffect(() => {
    const media = window.matchMedia("(max-width: 820px)");
    const sync = () => {
      setMobile(media.matches);
      if (!media.matches) setMobileOpen(false);
    };
    sync();
    media.addEventListener("change", sync);
    return () => media.removeEventListener("change", sync);
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCommandOpen((open) => !open);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const toggleRail = () => {
    if (mobile) setMobileOpen((open) => !open);
    else setRailCollapsed(!railCollapsed);
  };

  const confirmLogout = async () => {
    if (logoutLoading) return;
    setLogoutLoading(true);
    try {
      await logout();
      setLogoutOpen(false);
    } catch (error) {
      pushToast({
        title: "退出失败",
        message: error instanceof ApiError ? error.message : "服务暂时不可用，请稍后重试",
        tone: "danger",
      });
    } finally {
      setLogoutLoading(false);
    }
  };

  return (
    <div className="stage">
      <div
        ref={shellRef}
        data-entrance-ready="true"
        className={[
          "app-frame",
          railCollapsed && !mobile && "is-collapsed",
          mobile && mobileOpen && "is-mobile-open",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <Sidebar
          groups={groups}
          collapsed={railCollapsed && !mobile}
          onNavigate={() => setMobileOpen(false)}
        />
        {mobile && mobileOpen ? (
          <div className="rail-scrim" aria-hidden="true" onClick={() => setMobileOpen(false)} />
        ) : null}
        <div className={`workspace${scrolled ? " is-scrolled" : ""}`}>
          <Header
            groups={groups}
            onToggleRail={toggleRail}
            onCommand={() => setCommandOpen(true)}
            onAppearance={() => setAppearanceOpen(true)}
            onPassword={() => setPasswordOpen(true)}
            onLogout={() => setLogoutOpen(true)}
          />
          <main
            className="content"
            tabIndex={-1}
            onScroll={(event) => setScrolled(event.currentTarget.scrollTop > 8)}
          >
            {children}
          </main>
        </div>
      </div>
      <CommandPalette
        open={commandOpen}
        groups={groups}
        onClose={() => setCommandOpen(false)}
        onAppearance={() => setAppearanceOpen(true)}
        onPassword={() => setPasswordOpen(true)}
        onLogout={() => setLogoutOpen(true)}
      />
      <AppearanceDialog open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
      <PasswordDialog open={passwordOpen} onClose={() => setPasswordOpen(false)} />
      <ConfirmDialog
        open={logoutOpen}
        title="退出登录"
        message="确认退出当前账号吗？退出后需要重新登录。"
        confirmText="退出"
        danger
        loading={logoutLoading}
        onClose={() => setLogoutOpen(false)}
        onConfirm={() => void confirmLogout()}
      />
    </div>
  );
}
