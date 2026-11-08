"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Search, X } from "lucide-react";
import { createPortal } from "react-dom";
import type { NavGroup } from "./nav";
import { AppIcon } from "@/components/ui/app-icon";

export function CommandPalette({
  open,
  groups,
  onClose,
  onAppearance,
}: {
  open: boolean;
  groups: NavGroup[];
  onClose: () => void;
  onAppearance: () => void;
}) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const items = useMemo(
    () => groups.flatMap((group) => group.items.map((item) => ({ ...item, group: group.label }))),
    [groups],
  );
  const filtered = items.filter((item) =>
    `${item.label} ${item.group}`.toLowerCase().includes(query.toLowerCase()),
  );

  if (!open || typeof document === "undefined") return null;

  const close = () => {
    setQuery("");
    onClose();
  };

  return createPortal(
    <div
      className="overlay cmd-overlay"
      role="presentation"
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          close();
        }
      }}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) close();
      }}
    >
      <div className="cmd-panel m-float" role="dialog" aria-modal="true" aria-label="命令面板">
        <div className="cmd-input">
          <Search size={18} />
          <input
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="搜索菜单或动作…"
            aria-label="搜索菜单或动作"
          />
          <button type="button" className="icon-btn" aria-label="关闭" onClick={close}>
            <X size={16} />
          </button>
        </div>
        <div className="cmd-results">
          <span className="cmd-group">页面</span>
          {filtered.length ? (
            filtered.map((item) => (
              <button
                type="button"
                className="cmd-item"
                key={item.key}
                onClick={() => {
                  close();
                  router.push(item.path);
                }}
              >
                <span className="cmd-ico m-groove">
                  <AppIcon name={item.icon} size={16} />
                </span>
                <span className="cmd-label">{item.label}</span>
                <span className="cmd-sub">{item.group}</span>
                <AppIcon className="cmd-go" name="arrowRight" size={15} />
              </button>
            ))
          ) : (
            <div className="select-empty">没有匹配项</div>
          )}
          <span className="cmd-group">动作</span>
          <button
            type="button"
            className="cmd-item"
            onClick={() => {
              close();
              onAppearance();
            }}
          >
            <span className="cmd-ico m-groove">
              <AppIcon name="contrast" size={16} />
            </span>
            <span className="cmd-label">外观设置</span>
            <span className="cmd-sub">主题与密度</span>
            <AppIcon className="cmd-go" name="arrowRight" size={15} />
          </button>
        </div>
        <div className="cmd-foot">
          <span>
            <span className="kbd">Esc</span> 关闭
          </span>
          <span className="spacer" />
          <span>{filtered.length} 个页面</span>
        </div>
      </div>
    </div>,
    document.body,
  );
}
