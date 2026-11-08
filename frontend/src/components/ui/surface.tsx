import type { HTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/utils/cn";
import { AppIcon } from "./app-icon";

export function Card({ className, children, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("panel m-surface", className)} {...props}>
      {children}
    </div>
  );
}

export function GlassCard({ className, children, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("m-float", className)} {...props}>
      {children}
    </div>
  );
}

export function PageHeader({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children?: ReactNode;
}) {
  return (
    <header className="page-head reveal">
      <div className="page-head-main">
        <h1 className="page-title">{title}</h1>
        {description ? <p className="page-desc">{description}</p> : null}
      </div>
      {children ? <div className="page-head-side">{children}</div> : null}
    </header>
  );
}

export function SectionLabel({ children }: { children: ReactNode }) {
  return <span className="section-label">{children}</span>;
}

export function Tag({
  tone = "neutral",
  dot = false,
  children,
}: {
  tone?: "neutral" | "ok" | "info" | "danger";
  dot?: boolean;
  children: ReactNode;
}) {
  return (
    <span className={cn("tag", `tone-${tone}`)}>
      {dot ? <span className="tag-dot" aria-hidden="true" /> : null}
      {children}
    </span>
  );
}

export function Avatar({ name, size = 32 }: { name?: string; size?: number }) {
  const label = name?.trim() || "?";
  const initials = label.slice(0, 1).toUpperCase();
  return (
    <span
      className="avatar"
      style={{
        width: size,
        height: size,
        fontSize: Math.max(11, Math.round(size * 0.38)),
        background: "linear-gradient(145deg, var(--blue), #78a8ff)",
      }}
      aria-label={label}
      title={label}
    >
      {initials}
    </span>
  );
}

export function EmptyState({
  title,
  description,
  icon = "help",
}: {
  title: string;
  description?: string;
  icon?: string;
}) {
  return (
    <div className="empty">
      <span className="empty-ico m-groove">
        <AppIcon name={icon} size={20} />
      </span>
      <span className="empty-title">{title}</span>
      {description ? <span>{description}</span> : null}
    </div>
  );
}

export function Skeleton({ width = "100%", height = 12 }: { width?: string; height?: number }) {
  return <span className="skeleton" style={{ width, height }} aria-hidden="true" />;
}

export function Metric({
  value,
  label,
  unit,
  size = "m",
}: {
  value?: number | string | null;
  label?: string;
  unit?: string;
  size?: "s" | "m" | "l" | "xl";
}) {
  return (
    <div className={cn("metric", `metric-${size}`)}>
      {label ? <span className="metric-label">{label}</span> : null}
      <span className="metric-figure tnum">
        {value ?? "—"}
        {unit ? <span className="metric-unit">{unit}</span> : null}
      </span>
    </div>
  );
}
