"use client";

import type { ReactNode } from "react";
import { AlertTriangle, ChevronRight, Inbox, LoaderCircle } from "lucide-react";
import { PageContainer } from "@/components/layout/page-container";
import { Button } from "@/components/ui/button";
import { EmptyState, PageHeader, Tag } from "@/components/ui/surface";
import { cn } from "@/lib/utils/cn";

export function BusinessPage({
  title,
  description,
  actions,
  children,
  className,
  fitViewport = false,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  fitViewport?: boolean;
}) {
  return (
    <PageContainer>
      <div className={cn("list-page", fitViewport && "is-fit-viewport", className)}>
        <PageHeader title={title} description={description}>
          {actions ? <div className="page-actions">{actions}</div> : null}
        </PageHeader>
        {children}
      </div>
    </PageContainer>
  );
}

export function SearchForm({
  children,
  onSearch,
  onReset,
  loading = false,
}: {
  children: ReactNode;
  onSearch: () => void;
  onReset?: () => void;
  loading?: boolean;
}) {
  return (
    <form
      className="search-form m-float reveal"
      onSubmit={(event) => {
        event.preventDefault();
        onSearch();
      }}
    >
      {children}
      <div className="search-actions">
        {onReset ? (
          <Button type="button" variant="ghost" icon="refresh" onClick={onReset}>
            重置
          </Button>
        ) : null}
        <Button type="submit" variant="primary" icon="search" loading={loading}>
          查询
        </Button>
      </div>
    </form>
  );
}

export function DataState({
  loading,
  error,
  permissionDenied,
  empty,
  emptyTitle = "暂无数据",
  emptyDescription,
  onRetry,
  children,
}: {
  loading?: boolean;
  error?: string | null;
  permissionDenied?: boolean;
  empty?: boolean;
  emptyTitle?: string;
  emptyDescription?: string;
  onRetry?: () => void;
  children: ReactNode;
}) {
  if (permissionDenied) {
    return (
      <EmptyState
        title="当前账号没有访问权限"
        description="后端授权仍是最终判断，请联系管理员调整角色权限。"
        icon="shield"
      />
    );
  }
  if (error) {
    return (
      <div className="empty">
        <span className="empty-ico m-groove">
          <AlertTriangle size={20} />
        </span>
        <span className="empty-title">数据加载失败</span>
        <span>{error}</span>
        {onRetry ? (
          <Button variant="secondary" icon="refresh" onClick={onRetry}>
            重试
          </Button>
        ) : null}
      </div>
    );
  }
  if (loading) {
    return (
      <div className="empty" aria-live="polite">
        <LoaderCircle className="spin" size={22} />
        <span>正在加载真实业务数据…</span>
      </div>
    );
  }
  if (empty) {
    return <EmptyState title={emptyTitle} description={emptyDescription} icon="help" />;
  }
  return <>{children}</>;
}

const STATUS_META: Record<string, { label: string; tone: "ok" | "danger" | "info" | "neutral" }> = {
  active: { label: "正常", tone: "ok" },
  enabled: { label: "启用", tone: "ok" },
  success: { label: "成功", tone: "ok" },
  completed: { label: "已完成", tone: "ok" },
  published: { label: "已发布", tone: "ok" },
  running: { label: "运行中", tone: "info" },
  pending: { label: "待处理", tone: "info" },
  draft: { label: "草稿", tone: "neutral" },
  disabled: { label: "停用", tone: "danger" },
  suspended: { label: "停用", tone: "danger" },
  paused: { label: "已暂停", tone: "neutral" },
  failure: { label: "失败", tone: "danger" },
  failed: { label: "失败", tone: "danger" },
  denied: { label: "拒绝", tone: "danger" },
  withdrawn: { label: "已撤回", tone: "neutral" },
};

export function StatusTag({ status, label }: { status?: string | null; label?: string }) {
  const normalized = (status ?? "").toLowerCase();
  const meta = STATUS_META[normalized] ?? { label: status || "未知", tone: "neutral" as const };
  return (
    <Tag tone={meta.tone} dot>
      {label ?? meta.label}
    </Tag>
  );
}

export function BulkActionBar({
  count,
  children,
  onClear,
}: {
  count: number;
  children: ReactNode;
  onClear: () => void;
}) {
  if (count <= 0) return null;
  return (
    <div className="float-bar m-float" role="status">
      <span className="float-bar-count">
        已选择 <b>{count}</b> 项
      </span>
      <span className="float-bar-sep" aria-hidden="true" />
      {children}
      <Button variant="text" className="float-bar-clear" onClick={onClear}>
        取消选择
      </Button>
    </div>
  );
}

export function TreeNav<T>({
  items,
  getId,
  getLabel,
  getChildren,
  getMeta,
  selectedId,
  onSelect,
  rootLabel,
  rootCount,
}: {
  items: T[];
  getId: (item: T) => string;
  getLabel: (item: T) => string;
  getChildren: (item: T) => T[] | undefined;
  getMeta?: (item: T) => string | undefined;
  selectedId?: string | null;
  onSelect: (id: string | null) => void;
  rootLabel?: string;
  rootCount?: number;
}) {
  const renderItems = (nodes: T[], depth: number): ReactNode =>
    nodes.map((item) => {
      const id = getId(item);
      const children = getChildren(item) ?? [];
      return (
        <div key={id}>
          <button
            type="button"
            className={cn("tree-row", selectedId === id && "is-selected")}
            style={{ paddingLeft: 8 + depth * 14 }}
            onClick={() => onSelect(id)}
          >
            <ChevronRight size={13} className={children.length ? undefined : "invisible"} />
            <span className="tree-label">{getLabel(item)}</span>
            {getMeta?.(item) ? <span className="tree-count">{getMeta(item)}</span> : null}
          </button>
          {children.length ? renderItems(children, depth + 1) : null}
        </div>
      );
    });

  return (
    <div className="dept-tree">
      <div className="split-side-head">
        <span className="section-label">组织筛选</span>
      </div>
      <div className="tree" role="tree">
        <button
          type="button"
          className={cn("tree-row", !selectedId && "is-selected")}
          onClick={() => onSelect(null)}
        >
          <Inbox size={14} />
          <span className="tree-label">{rootLabel ?? "全部数据"}</span>
          {rootCount !== undefined ? <span className="tree-count">{rootCount}</span> : null}
        </button>
        {renderItems(items, 0)}
      </div>
    </div>
  );
}

export function DetailGrid({
  items,
  className,
}: {
  items: Array<{ label: string; value: ReactNode; wide?: boolean }>;
  className?: string;
}) {
  return (
    <dl className={cn("kv-grid", className)}>
      {items.map((item) => (
        <div className={cn("kv", item.wide && "span-2")} key={item.label}>
          <dt className="kv-label">{item.label}</dt>
          <dd className="kv-value">{item.value ?? "—"}</dd>
        </div>
      ))}
    </dl>
  );
}

export function SectionBand({
  title,
  meta,
  children,
  actions,
}: {
  title: string;
  meta?: ReactNode;
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <section className="band reveal">
      <div className="band-head">
        <div>
          <span className="section-label">{title}</span>
          {meta ? (
            <div className="faint" style={{ marginTop: 4 }}>
              {meta}
            </div>
          ) : null}
        </div>
        {actions ? <div className="band-tools">{actions}</div> : null}
      </div>
      {children}
    </section>
  );
}

export function formatDateTime(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(date);
}
