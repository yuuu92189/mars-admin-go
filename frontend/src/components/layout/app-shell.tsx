"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { AuthProvider } from "@/components/auth/auth-provider";
import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { AppearanceDialog } from "./appearance-dialog";
import { CommandPalette } from "./command-palette";
import { buildNavGroups } from "./nav";
import { useShellEntrance } from "@/lib/motion";
import type { CurrentUser } from "@/lib/api/types";

export function AppShell({ user, children }: { user: CurrentUser; children: ReactNode }) {
  const groups = useMemo(() => buildNavGroups(user), [user]);
  const [collapsed, setCollapsed] = useState(false);
  const [mobile, setMobile] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [commandOpen, setCommandOpen] = useState(false);
  const [appearanceOpen, setAppearanceOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const shellRef = useShellEntrance<HTMLDivElement>();

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
    else setCollapsed((current) => !current);
  };

  return (
    <AuthProvider initialUser={user}>
      <div className="stage">
        <div
          ref={shellRef}
          data-entrance-ready="true"
          className={[
            "app-frame",
            collapsed && !mobile && "is-collapsed",
            mobile && mobileOpen && "is-mobile-open",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          <Sidebar groups={groups} collapsed={collapsed} onNavigate={() => setMobileOpen(false)} />
          {mobile && mobileOpen ? (
            <div className="rail-scrim" aria-hidden="true" onClick={() => setMobileOpen(false)} />
          ) : null}
          <div className={`workspace${scrolled ? " is-scrolled" : ""}`}>
            <Header
              groups={groups}
              onToggleRail={toggleRail}
              onCommand={() => setCommandOpen(true)}
              onAppearance={() => setAppearanceOpen(true)}
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
        />
        <AppearanceDialog open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
      </div>
    </AuthProvider>
  );
}
