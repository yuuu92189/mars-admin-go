"use client";

import { useId, useMemo } from "react";
import { cn } from "@/lib/utils/cn";

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export function TrendChart({
  values,
  labels,
  height = 212,
  unit = "",
  tone = "ink",
  ariaLabel = "趋势图",
}: {
  values: number[];
  labels: string[];
  height?: number;
  unit?: string;
  tone?: "ink" | "blue";
  ariaLabel?: string;
}) {
  const gradientId = useId().replace(/:/g, "");
  const width = 720;
  const pad = { top: 18, right: 12, bottom: 28, left: 38 };
  const innerW = width - pad.left - pad.right;
  const innerH = height - pad.top - pad.bottom;
  const max = Math.max(1, ...values);
  const points = values.map((value, index) => {
    const x = pad.left + (values.length <= 1 ? innerW / 2 : (index / (values.length - 1)) * innerW);
    const y = pad.top + innerH - (value / max) * innerH;
    return { x, y, value, label: labels[index] ?? "" };
  });
  const path = points
    .map((point, index) => `${index === 0 ? "M" : "L"} ${point.x.toFixed(2)} ${point.y.toFixed(2)}`)
    .join(" ");
  const areaPath = points.length
    ? `${path} L ${points.at(-1)?.x ?? pad.left} ${pad.top + innerH} L ${points[0]?.x ?? pad.left} ${pad.top + innerH} Z`
    : "";
  const grid = [0, 0.25, 0.5, 0.75, 1];
  return (
    <div
      className={cn("trend", tone === "blue" ? "tone-blue" : "tone-ink")}
      role="img"
      aria-label={ariaLabel}
    >
      <svg viewBox={`0 0 ${width} ${height}`} width="100%" height={height}>
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop className="trend-stop" offset="0%" stopOpacity="0.24" />
            <stop className="trend-stop" offset="100%" stopOpacity="0" />
          </linearGradient>
        </defs>
        <g className="trend-grid">
          {grid.map((ratio) => {
            const y = pad.top + innerH * ratio;
            return <line key={ratio} x1={pad.left} x2={width - pad.right} y1={y} y2={y} />;
          })}
        </g>
        {areaPath ? <path d={areaPath} fill={`url(#${gradientId})`} /> : null}
        {path ? <path className="trend-line" d={path} /> : null}
        <g className="trend-nodes">
          {points.map((point, index) => (
            <g key={`${point.label}-${index}`}>
              {index === points.length - 1 ? (
                <circle className="trend-halo" cx={point.x} cy={point.y} r="8" />
              ) : null}
              <circle cx={point.x} cy={point.y} r="3.5" />
            </g>
          ))}
        </g>
        <g className="trend-axis">
          {points.map((point, index) => {
            if (points.length > 12 && index % 3 !== 0 && index !== points.length - 1) return null;
            return (
              <text key={`label-${index}`} x={point.x} y={height - 8} textAnchor="middle">
                {point.label}
              </text>
            );
          })}
        </g>
        <g className="trend-axis">
          {grid.map((ratio) => {
            const value = Math.round(max * (1 - ratio));
            const y = pad.top + innerH * ratio + 3;
            return (
              <text key={`axis-${ratio}`} x={pad.left - 8} y={y} textAnchor="end">
                {value}
              </text>
            );
          })}
        </g>
      </svg>
      <span className="sr-only">
        {points.map((point) => `${point.label}: ${point.value}${unit}`).join("，")}
      </span>
    </div>
  );
}

export function BarChart({
  values,
  labels,
  failed = [],
  highlight,
  height = 154,
  unit = "",
  ariaLabel = "柱状图",
}: {
  values: number[];
  labels: string[];
  failed?: number[];
  highlight?: number | null;
  height?: number;
  unit?: string;
  ariaLabel?: string;
}) {
  const width = 420;
  const pad = { top: 14, right: 8, bottom: 26, left: 8 };
  const innerW = width - pad.left - pad.right;
  const innerH = height - pad.top - pad.bottom;
  const max = Math.max(1, ...values);
  const gap = 8;
  const barW = values.length
    ? Math.max(8, (innerW - gap * (values.length - 1)) / values.length)
    : 0;
  return (
    <div className="column-chart" role="img" aria-label={ariaLabel}>
      <svg viewBox={`0 0 ${width} ${height}`} width="100%" height={height}>
        {values.map((value, index) => {
          const x = pad.left + index * (barW + gap);
          const h = (value / max) * innerH;
          const failedValue = failed[index] ?? 0;
          const failedH = (failedValue / max) * innerH;
          const active = highlight === index;
          return (
            <g key={`${labels[index] ?? index}-${index}`} className="column-chart-slot">
              <rect
                x={x}
                y={pad.top + innerH - h}
                width={barW}
                height={Math.max(value > 0 ? 3 : 0, h)}
                rx="5"
                className={cn("mchart-bar", active && "is-active")}
              />
              {failedH > 0 ? (
                <rect
                  x={x + barW * 0.28}
                  y={pad.top + innerH - failedH}
                  width={barW * 0.44}
                  height={failedH}
                  rx="3"
                  className="mchart-fail"
                />
              ) : null}
              <text x={x + barW / 2} y={height - 8} textAnchor="middle" className="mchart-label">
                {labels[index] ?? ""}
              </text>
            </g>
          );
        })}
      </svg>
      <span className="sr-only">
        {values.map((value, index) => `${labels[index]}: ${value}${unit}`).join("，")}
      </span>
    </div>
  );
}

export function TickRing({
  value,
  size = 164,
  label = "成功率",
}: {
  value?: number | null;
  size?: number;
  label?: string;
}) {
  const radius = 62;
  const circumference = 2 * Math.PI * radius;
  const normalized = clamp(value ?? 0, 0, 100);
  const dash = (normalized / 100) * circumference;
  const ticks = useMemo(() => Array.from({ length: 60 }, (_, index) => index), []);
  return (
    <div
      className="ring-wrap"
      role="img"
      aria-label={`${label} ${value == null ? "暂无数据" : `${normalized}%`}`}
    >
      <svg viewBox="0 0 160 160" width={size} height={size}>
        {ticks.map((tick) => {
          const angle = (tick / ticks.length) * Math.PI * 2 - Math.PI / 2;
          const major = tick % 5 === 0;
          const outer = 70;
          const inner = major ? 64 : 67;
          return (
            <line
              key={tick}
              className={cn("mring-tick", major && "is-major")}
              x1={80 + Math.cos(angle) * inner}
              y1={80 + Math.sin(angle) * inner}
              x2={80 + Math.cos(angle) * outer}
              y2={80 + Math.sin(angle) * outer}
            />
          );
        })}
        <circle
          cx="80"
          cy="80"
          r={radius}
          fill="none"
          stroke="var(--line-strong)"
          strokeWidth="7"
        />
        <circle
          cx="80"
          cy="80"
          r={radius}
          fill="none"
          stroke="var(--lime)"
          strokeWidth="7"
          strokeLinecap="round"
          strokeDasharray={`${dash} ${circumference}`}
          transform="rotate(-90 80 80)"
        />
        <text x="80" y="75" textAnchor="middle" className="mring-value">
          {value == null ? "—" : `${normalized}%`}
        </text>
        <text x="80" y="94" textAnchor="middle" className="mring-label">
          {label}
        </text>
      </svg>
    </div>
  );
}

export type HeatCell = {
  day: string;
  hour: number;
  value: number;
  risky?: boolean;
};

export function HeatGrid({ cells }: { cells: HeatCell[] }) {
  const days = [...new Set(cells.map((cell) => cell.day))];
  const max = Math.max(1, ...cells.map((cell) => cell.value));
  const lookup = new Map(cells.map((cell) => [`${cell.day}-${cell.hour}`, cell]));
  return (
    <div className="heat" role="img" aria-label="登录活动热力图">
      <div className="heat-hours">
        {Array.from({ length: 24 }, (_, hour) => (
          <span key={hour}>{hour % 3 === 0 ? String(hour).padStart(2, "0") : ""}</span>
        ))}
      </div>
      {days.map((day) => (
        <div className="heat-row" key={day}>
          <span className="heat-day">{day}</span>
          {Array.from({ length: 24 }, (_, hour) => {
            const cell = lookup.get(`${day}-${hour}`);
            const value = cell?.value ?? 0;
            const level = value === 0 ? 0 : Math.max(1, Math.ceil((value / max) * 4));
            return (
              <span
                key={hour}
                className={cn("heat-cell", level > 0 && `lv-${level}`, cell?.risky && "is-risky")}
                title={`${day} ${String(hour).padStart(2, "0")}:00 · ${value} 次`}
              />
            );
          })}
        </div>
      ))}
      <div className="heat-foot">
        <span className="heat-readout">数据来自 Go 登录审计记录</span>
        <span className="heat-legend">
          少
          <i />
          <i className="lv-1" />
          <i className="lv-2" />
          <i className="lv-3" />
          <i className="lv-4" />多
        </span>
      </div>
    </div>
  );
}

export type SwimlaneItem = {
  id: string;
  label: string;
  start: number;
  end: number;
  failed?: boolean;
};

export function Swimlane({
  lanes,
  now = 0,
  compact = false,
}: {
  lanes: Array<{ id: string; label: string; items: SwimlaneItem[] }>;
  now?: number;
  compact?: boolean;
}) {
  const max = 24 * 60 * 60;
  const normalizedNow = clamp(now, 0, max);
  return (
    <div className={cn("swimlane", compact && "is-compact")} role="img" aria-label="操作时间泳道">
      <div className="swim-head">
        <span className="swim-corner" />
        <div className="swim-grid" style={{ ["--swim-cols" as string]: "24" }} />
      </div>
      <div className="swim-body">
        {lanes.map((lane) => (
          <div className="swim-lane" key={lane.id}>
            <div className="swim-label">
              <span className="swim-name">{lane.label}</span>
              <span className="swim-count">{lane.items.length} 次</span>
            </div>
            <div className="swim-track">
              {lane.items.map((item) => {
                const left = clamp(item.start / max, 0, 1) * 100;
                const width = Math.max(0.6, ((item.end - item.start) / max) * 100);
                return (
                  <span
                    key={item.id}
                    className={cn("swim-block", item.failed && "is-red")}
                    style={{ left: `${left}%`, width: `${width}%` }}
                    title={`${item.label} · ${item.failed ? "失败" : "成功"}`}
                  />
                );
              })}
              <span className="swim-now" style={{ left: `${(normalizedNow / max) * 100}%` }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
