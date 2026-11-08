"use client";

import { useCallback, useMemo, useState } from "react";
import { Activity, FileSearch, ShieldCheck, Timer } from "lucide-react";
import {
  BusinessPage,
  DataState,
  DetailGrid,
  SearchForm,
  SectionBand,
  StatusTag,
  formatDateTime,
} from "@/components/business/business-page";
import { BarChart, Swimlane } from "@/components/charts/metric-charts";
import { Button, IconButton } from "@/components/ui/button";
import { DataTable, Pagination, type TableColumn } from "@/components/ui/data-table";
import { Field, TextInput } from "@/components/ui/input";
import { Drawer } from "@/components/ui/overlay";
import { Card, EmptyState, Metric, Tag } from "@/components/ui/surface";
import { ApiError } from "@/lib/api/client";
import { listAuditEvents } from "@/lib/api/audit";
import type { AuditEvent, AuditEventPage } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 20;

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function dayKey(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  return new Intl.DateTimeFormat("zh-CN", { month: "2-digit", day: "2-digit" }).format(date);
}

export function OperationLogsView() {
  const [page, setPage] = useState<AuditEventPage>({
    items: [],
    total: 0,
    page: 1,
    pageSize: PAGE_SIZE,
  });
  const [pageNumber, setPageNumber] = useState(1);
  const [action, setAction] = useState("");
  const [resourceType, setResourceType] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [detail, setDetail] = useState<AuditEvent | null>(null);
  const [sample, setSample] = useState<AuditEvent[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const [result, recent] = await Promise.all([
        listAuditEvents({
          page: pageNumber,
          pageSize: PAGE_SIZE,
          action,
          resourceType,
          from: from ? new Date(from).toISOString() : undefined,
          to: to ? new Date(to).toISOString() : undefined,
        }),
        listAuditEvents({ page: 1, pageSize: 100 }),
      ]);
      setPage(result);
      setSample(recent.items);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [action, from, pageNumber, resourceType, to]);

  useDeferredEffect(load);

  const trend = useMemo(() => {
    const buckets = new Map<string, { value: number; failed: number }>();
    for (const event of sample) {
      const key = dayKey(event.occurredAt);
      if (!key) continue;
      const current = buckets.get(key) ?? { value: 0, failed: 0 };
      current.value += 1;
      if (event.outcome !== "success") current.failed += 1;
      buckets.set(key, current);
    }
    return [...buckets.entries()].slice(-7);
  }, [sample]);
  const success = sample.filter((event) => event.outcome === "success").length;
  const denied = sample.filter((event) => event.outcome === "denied").length;
  const averageDuration = useMemo(() => {
    const durations = sample
      .map((event) => event.metadata?.duration_ms)
      .filter((value): value is number => typeof value === "number");
    return durations.length
      ? Math.round(durations.reduce((total, value) => total + value, 0) / durations.length)
      : 0;
  }, [sample]);
  const lanes = useMemo(() => {
    const grouped = new Map<string, AuditEvent[]>();
    for (const event of sample.slice(0, 40)) {
      const key = event.resourceType || "system";
      grouped.set(key, [...(grouped.get(key) ?? []), event]);
    }
    return [...grouped.entries()].slice(0, 6).map(([label, events]) => ({
      id: label,
      label,
      items: events.map((event, index) => ({
        id: event.id ?? `${label}-${index}`,
        label: event.action ?? "action",
        start: index * 900,
        end: index * 900 + 420,
        failed: event.outcome !== "success",
      })),
    }));
  }, [sample]);

  const columns: TableColumn<AuditEvent>[] = [
    {
      key: "action",
      header: "操作",
      render: (event) => (
        <button type="button" className="cell-main link-btn" onClick={() => setDetail(event)}>
          <span className="row-ico m-groove">
            <Activity size={15} />
          </span>
          <span>
            <span className="cell-title">{event.action || "未命名操作"}</span>
            <span className="cell-sub">{event.resourceType || "system"}</span>
          </span>
        </button>
      ),
    },
    { key: "actor", header: "操作人", render: (event) => event.actorName || "系统" },
    { key: "outcome", header: "结果", render: (event) => <StatusTag status={event.outcome} /> },
    {
      key: "ip",
      header: "IP",
      render: (event) => <span className="mono">{event.ipAddress || "—"}</span>,
    },
    {
      key: "request",
      header: "请求 ID",
      render: (event) => <span className="code-chip">{event.requestId?.slice(0, 12) || "—"}</span>,
    },
    {
      key: "time",
      header: "发生时间",
      render: (event) => <span className="tnum faint">{formatDateTime(event.occurredAt)}</span>,
    },
    {
      key: "actions",
      header: "详情",
      align: "right",
      render: (event) => (
        <IconButton label="查看详情" icon="file" small onClick={() => setDetail(event)} />
      ),
    },
  ];

  return (
    <BusinessPage
      title="操作日志"
      description="操作审计、结果、请求链路与真实事件明细；当前 Go API 不提供日志删除。"
    >
      <div className="stack">
        <SearchForm
          loading={loading}
          onSearch={() => {
            setPageNumber(1);
            void load();
          }}
          onReset={() => {
            setAction("");
            setResourceType("");
            setFrom("");
            setTo("");
            setPageNumber(1);
          }}
        >
          <Field label="操作">
            <TextInput
              value={action}
              onChange={(event) => setAction(event.target.value)}
              placeholder="GET / POST"
            />
          </Field>
          <Field label="资源类型">
            <TextInput
              value={resourceType}
              onChange={(event) => setResourceType(event.target.value)}
              placeholder="users / roles"
            />
          </Field>
          <Field label="开始时间">
            <TextInput
              type="datetime-local"
              value={from}
              onChange={(event) => setFrom(event.target.value)}
            />
          </Field>
          <Field label="结束时间">
            <TextInput
              type="datetime-local"
              value={to}
              onChange={(event) => setTo(event.target.value)}
            />
          </Field>
        </SearchForm>

        <SectionBand
          title="审计概览"
          meta={`最近 ${sample.length} 条真实审计事件 · 时区 Asia/Shanghai`}
          actions={<Tag tone="info">Go Audit</Tag>}
        >
          <div className="readout-strip">
            <Metric label="样本操作" value={sample.length} size="m" />
            <Metric label="成功" value={success} size="m" />
            <Metric label="拒绝" value={denied} size="m" />
            <Metric label="平均耗时" value={averageDuration} unit="ms" size="m" />
          </div>
        </SectionBand>

        <div className="band-grid-2">
          <SectionBand title="操作趋势" meta="按最近 100 条事件日期聚合">
            {trend.length ? (
              <BarChart
                values={trend.map(([, value]) => value.value)}
                failed={trend.map(([, value]) => value.failed)}
                labels={trend.map(([label]) => label)}
                height={180}
                ariaLabel="操作日志趋势"
              />
            ) : (
              <EmptyState title="暂无趋势数据" icon="chart" />
            )}
          </SectionBand>
          <SectionBand title="操作泳道" meta="按资源类型分组">
            {lanes.length ? (
              <Swimlane
                lanes={lanes}
                compact
                now={Math.max(0, ...lanes.flatMap((lane) => lane.items.map((item) => item.end)))}
              />
            ) : (
              <EmptyState title="暂无泳道数据" icon="clock" />
            )}
          </SectionBand>
        </div>

        <Card className="reveal">
          <div className="panel-head">
            <div>
              <span className="section-label">事件明细</span>
              <span className="panel-meta">{page.total ?? 0} 条</span>
            </div>
            <Tag tone="info">
              <ShieldCheck size={13} /> RFC 9457 错误链路
            </Tag>
          </div>
          <DataState
            loading={loading}
            error={error}
            permissionDenied={permissionDenied}
            empty={!loading && !error && !(page.items ?? []).length}
            emptyTitle="没有匹配的审计事件"
            onRetry={() => void load()}
          >
            <DataTable
              columns={columns}
              rows={page.items ?? []}
              rowKey={(event) => event.id ?? ""}
            />
            <Pagination
              page={pageNumber}
              pageSize={PAGE_SIZE}
              total={page.total ?? 0}
              onChange={setPageNumber}
            />
          </DataState>
        </Card>
      </div>

      <Drawer
        open={Boolean(detail)}
        title="操作事件详情"
        description={detail?.action}
        onClose={() => setDetail(null)}
        width={560}
      >
        {detail ? (
          <div className="stack">
            <div className="drawer-toolbar">
              <StatusTag status={detail.outcome} />
              <Tag tone="info">{detail.resourceType || "system"}</Tag>
            </div>
            <DetailGrid
              items={[
                { label: "事件 ID", value: detail.id, wide: true },
                { label: "操作人", value: detail.actorName || "系统" },
                { label: "发生时间", value: formatDateTime(detail.occurredAt) },
                { label: "IP 地址", value: detail.ipAddress || "—" },
                { label: "请求 ID", value: detail.requestId || "—" },
                { label: "Trace ID", value: detail.traceId || "—" },
                { label: "User Agent", value: detail.userAgent || "—", wide: true },
              ]}
            />
            <Card>
              <div className="panel-head">
                <span className="section-label">脱敏元数据</span>
              </div>
              <div className="panel-body">
                <pre className="code-block">{JSON.stringify(detail.metadata ?? {}, null, 2)}</pre>
              </div>
            </Card>
          </div>
        ) : null}
      </Drawer>
    </BusinessPage>
  );
}
