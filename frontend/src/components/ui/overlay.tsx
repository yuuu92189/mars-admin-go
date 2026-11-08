"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "@/lib/utils/cn";
import { lockDocumentScroll } from "@/lib/scroll-lock";

function useOverlay(open: boolean, onClose: () => void) {
  const panelRef = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);

  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement as HTMLElement | null;
    const releaseScroll = lockDocumentScroll();
    const focusableSelector =
      'button:not([disabled]):not([tabindex="-1"]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onCloseRef.current();
        return;
      }
      if (event.key !== "Tab" || !panelRef.current) return;
      const focusable = panelRef.current.querySelectorAll<HTMLElement>(focusableSelector);
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
    };
    document.addEventListener("keydown", onKeyDown);
    const focusTimer = window.setTimeout(() => {
      const preferred = panelRef.current?.querySelector<HTMLElement>("[data-autofocus]");
      const first = panelRef.current?.querySelector<HTMLElement>(focusableSelector);
      (preferred ?? first)?.focus();
    }, 0);
    return () => {
      window.clearTimeout(focusTimer);
      document.removeEventListener("keydown", onKeyDown);
      releaseScroll();
      previous?.focus();
    };
  }, [open]);

  return panelRef;
}

function Overlay({
  open,
  onClose,
  children,
  className,
}: {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  className?: string;
}) {
  if (!open || typeof document === "undefined") return null;
  return createPortal(
    <div
      className={cn("overlay", className)}
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      {children}
    </div>,
    document.body,
  );
}

export function Dialog({
  open,
  title,
  description,
  children,
  footer,
  onClose,
  width = 560,
}: {
  open: boolean;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  onClose: () => void;
  width?: number;
}) {
  const panelRef = useOverlay(open, onClose);
  const titleId = useId();
  const descriptionId = useId();
  return (
    <Overlay open={open} onClose={onClose}>
      <div
        ref={panelRef}
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        style={{ maxWidth: width }}
      >
        <div className="modal-head">
          <div>
            <h2 id={titleId} className="modal-title">
              {title}
            </h2>
            {description ? (
              <p id={descriptionId} className="modal-sub">
                {description}
              </p>
            ) : null}
          </div>
          <button type="button" className="icon-btn" aria-label="关闭" onClick={onClose}>
            <X size={18} />
          </button>
        </div>
        <div className="modal-body">{children}</div>
        {footer ? <div className="modal-foot">{footer}</div> : null}
      </div>
    </Overlay>
  );
}

export function Drawer({
  open,
  title,
  description,
  children,
  footer,
  onClose,
  width = 480,
}: {
  open: boolean;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  onClose: () => void;
  width?: number;
}) {
  const panelRef = useOverlay(open, onClose);
  const titleId = useId();
  const descriptionId = useId();
  return (
    <Overlay open={open} onClose={onClose} className="drawer-overlay">
      <div
        ref={panelRef}
        className="drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        style={{ width }}
      >
        <div className="drawer-head">
          <div>
            <h2 id={titleId} className="modal-title">
              {title}
            </h2>
            {description ? (
              <p id={descriptionId} className="modal-sub">
                {description}
              </p>
            ) : null}
          </div>
          <button type="button" className="icon-btn" aria-label="关闭" onClick={onClose}>
            <X size={18} />
          </button>
        </div>
        <div className="drawer-body">{children}</div>
        {footer ? <div className="drawer-foot">{footer}</div> : null}
      </div>
    </Overlay>
  );
}

export function Dropdown({
  open,
  children,
  className,
  id,
}: {
  open: boolean;
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  if (!open) return null;
  return (
    <div id={id} className={cn("menu-pop m-float", className)} role="menu">
      {children}
    </div>
  );
}

export function Tooltip({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span className="tooltip-wrap" data-tooltip={label}>
      {children}
    </span>
  );
}
