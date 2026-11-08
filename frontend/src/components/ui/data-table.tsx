"use client";

import type { ReactNode } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils/cn";
import { Checkbox } from "./input";

export type TableColumn<T> = {
  key: string;
  header: string;
  render: (row: T, index: number) => ReactNode;
  align?: "left" | "right";
  className?: string;
};

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  selectable = false,
  selected = new Set<string>(),
  onSelectionChange,
  emptyText = "暂无数据",
  loading = false,
}: {
  columns: TableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  selectable?: boolean;
  selected?: Set<string>;
  onSelectionChange?: (next: Set<string>) => void;
  emptyText?: string;
  loading?: boolean;
}) {
  const allSelected = rows.length > 0 && rows.every((row) => selected.has(rowKey(row)));
  const toggleAll = () => {
    if (!onSelectionChange) return;
    const next = new Set(selected);
    if (allSelected) rows.forEach((row) => next.delete(rowKey(row)));
    else rows.forEach((row) => next.add(rowKey(row)));
    onSelectionChange(next);
  };
  const toggle = (id: string) => {
    if (!onSelectionChange) return;
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onSelectionChange(next);
  };

  return (
    <div className="table-wrap is-scrollable" tabIndex={0}>
      <table className="table">
        <thead>
          <tr>
            {selectable ? (
              <th className="col-check">
                <Checkbox checked={allSelected} onChange={toggleAll} label="" />
              </th>
            ) : null}
            {columns.map((column) => (
              <th
                key={column.key}
                className={cn(column.align === "right" && "col-actions", column.className)}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <tr>
              <td colSpan={columns.length + (selectable ? 1 : 0)} className="muted">
                正在加载…
              </td>
            </tr>
          ) : rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length + (selectable ? 1 : 0)} className="empty">
                {emptyText}
              </td>
            </tr>
          ) : (
            rows.map((row, index) => {
              const id = rowKey(row);
              return (
                <tr key={id} className={selected.has(id) ? "is-selected" : undefined}>
                  {selectable ? (
                    <td className="col-check">
                      <Checkbox checked={selected.has(id)} onChange={() => toggle(id)} label="" />
                    </td>
                  ) : null}
                  {columns.map((column) => (
                    <td
                      key={column.key}
                      className={cn(column.align === "right" && "col-actions", column.className)}
                    >
                      {column.render(row, index)}
                    </td>
                  ))}
                </tr>
              );
            })
          )}
        </tbody>
      </table>
    </div>
  );
}

export function Pagination({
  page,
  pageSize,
  total,
  onChange,
}: {
  page: number;
  pageSize: number;
  total: number;
  onChange: (page: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="pager">
      <span className="pager-info">
        共 <b>{total}</b> 条 · 第 {page} / {pages} 页
      </span>
      <div className="pager-ctrl">
        <button
          type="button"
          className="pg-btn"
          disabled={page <= 1}
          aria-label="上一页"
          onClick={() => onChange(page - 1)}
        >
          <ChevronLeft size={15} />
        </button>
        <button type="button" className="pg-btn is-active" aria-current="page">
          {page}
        </button>
        <button
          type="button"
          className="pg-btn"
          disabled={page >= pages}
          aria-label="下一页"
          onClick={() => onChange(page + 1)}
        >
          <ChevronRight size={15} />
        </button>
      </div>
    </div>
  );
}

export function Tabs({
  tabs,
  value,
  onChange,
}: {
  tabs: Array<{ value: string; label: string }>;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="segmented m-groove" role="tablist" aria-label="视图切换">
      {tabs.map((tab) => (
        <button
          type="button"
          key={tab.value}
          role="tab"
          aria-selected={value === tab.value}
          className={cn(value === tab.value && "is-active")}
          onClick={() => onChange(tab.value)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
