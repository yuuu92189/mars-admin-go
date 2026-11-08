"use client";

import { useLayoutEffect, useRef } from "react";
import { usePathname, useRouter } from "next/navigation";
import { cn } from "@/lib/utils/cn";
import { APP } from "@/lib/config";
import type { NavGroup } from "./nav";
import { AppIcon } from "@/components/ui/app-icon";
import { BrandMark } from "@/components/business/brand-mark";

export function Sidebar({
  groups,
  collapsed,
  onNavigate,
}: {
  groups: NavGroup[];
  collapsed: boolean;
  onNavigate: () => void;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const navRef = useRef<HTMLElement>(null);
  const thumbRef = useRef<HTMLSpanElement>(null);
  const hasActive = groups.some((group) =>
    group.items.some((item) => pathname === item.path || pathname.startsWith(`${item.path}/`)),
  );

  useLayoutEffect(() => {
    const nav = navRef.current;
    const thumb = thumbRef.current;
    if (!nav || !thumb) return;

    const sync = () => {
      const target = nav.querySelector<HTMLElement>(".nav-item.is-active");
      if (!target) {
        thumb.style.opacity = "0";
        return;
      }
      const navRect = nav.getBoundingClientRect();
      const targetRect = target.getBoundingClientRect();
      thumb.style.opacity = "1";
      thumb.style.transform = `translateY(${targetRect.top - navRect.top}px)`;
    };

    sync();
    const observer = new ResizeObserver(sync);
    observer.observe(nav);
    return () => observer.disconnect();
  }, [groups, pathname, collapsed]);

  return (
    <aside className="rail m-rail" aria-label="主导航">
      <div className="rail-brand">
        <BrandMark />
        <div className="brand-text">
          <div className="brand-name">{APP.shortName}</div>
          <div className="brand-sub">{APP.subtitle}</div>
        </div>
      </div>
      <nav ref={navRef} className="rail-nav">
        <span
          ref={thumbRef}
          className="nav-thumb m-pill"
          aria-hidden="true"
          style={{ opacity: hasActive ? 1 : 0 }}
        />
        {groups.length ? (
          groups.map((group) => (
            <div className="nav-group" key={group.key}>
              {group.label !== "导航" ? <div className="nav-group-title">{group.label}</div> : null}
              <div className="nav-list">
                {group.items.map((item) => {
                  const active = pathname === item.path || pathname.startsWith(`${item.path}/`);
                  return (
                    <button
                      type="button"
                      key={item.key}
                      className={cn("nav-item", active && "is-active")}
                      aria-current={active ? "page" : undefined}
                      title={collapsed ? item.label : undefined}
                      onClick={() => {
                        onNavigate();
                        router.push(item.path);
                      }}
                    >
                      <AppIcon className="nav-ico" name={item.icon} size={18} />
                      <span className="nav-label">{item.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          ))
        ) : (
          <div className="nav-group">
            <div className="nav-group-title">导航</div>
            <div className="nav-list">
              <button
                type="button"
                className="nav-item is-active"
                onClick={() => router.push("/dashboard")}
              >
                <AppIcon className="nav-ico" name="dashboard" size={18} />
                <span className="nav-label">工作台</span>
              </button>
            </div>
          </div>
        )}
      </nav>
      <div className="rail-foot">
        <span className="nav-label faint" style={{ fontSize: 11, padding: "0 10px" }}>
          {APP.version}
        </span>
      </div>
    </aside>
  );
}
