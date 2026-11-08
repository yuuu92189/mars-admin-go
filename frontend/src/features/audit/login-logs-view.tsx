"use client";

import { useCallback, useMemo, useState } from "react";
import { KeyRound, ShieldAlert, UserCheck, Users } from "lucide-react";
import {
  BusinessPage,
  DataState,
  DetailGrid,
  SearchForm,
  SectionBand,
  StatusTag,
  formatDateTime,
} from "@/components/business/business-page";
import { HeatGrid, type HeatCell } from "@/components/charts/metric-charts";
import { DataTable, Pagination, type TableColumn } from "@/components/ui/data-table";
import { Field, TextInput } from "@/components/ui/input";
import { IconButton } from "@/components/ui/button";
import { Drawer } from "@/components/ui/overlay";
import { Card, EmptyState, Metric, Tag } from "@/components/ui/surface";
import { ApiError } from "@/lib/api/client";
import { listLoginEvents } from "@/lib/api/audit";
import type { LoginEvent, LoginEventPage } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 20;

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function heatDay(value?: string): string {
  if (!value) return "";
  return new Intl.DateTimeFormat("zh-CN", { weekday: "short" }).format(new Date(value));
}

export function LoginLogsView() {
  const [page, setPage] = useState<LoginEventPage>({
    items: [],
    total: 0,
    page: 1,
    pageSize: PAGE_SIZE,
  });
  const [pageNumber, setPageNumber] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [outcome, setOutcome] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [detail, setDetail] = useState<LoginEvent | null>(null);
  const [sample, setSample] = useState<LoginEvent[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const [result, recent] = await Promise.all([
        listLoginEvents(pageNumber, PAGE_SIZE),
        listLoginEvents(1, 100),
      ]);
      setPage(result);
      setSample(recent.items);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [pageNumber]);

  useDeferredEffect(load);

  const rows = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    return (page.items ?? []).filter((event) => {
      if (outcome && event.outcome !== outcome) return false;
      if (!query) return true;
      return [event.username, event.ipAddress, event.clientType, event.failureReason]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query));
    });
  }, [keyword, outcome, page.items]);

  const success = sample.filter((event) => event.outcome === "success").length;
  const failed = sample.length - success;
  const uniqueUsers = new Set(sample.map((event) => event.username).filter(Boolean)).size;
  const cells = useMemo<HeatCell[]>(() => {
    const buckets = new Map<string, HeatCell>();
    for (const event of sample) {
      if (!event.occurredAt) continue;
      const date = new Date(event.occurredAt);
      const day = heatDay(event.occurredAt);
      const key = `${day}-${date.getHours()}`;
      const current = buckets.get(key) ?? {
        day,
        hour: date.getHours(),
        value: 0,
        risky: false,
      };
      current.value += 1;
      if (event.outcome !== "success") current.risky = true;
      buckets.set(key, current);
    }
    return [...buckets.values()];
  }, [sample]);

  const columns: TableColumn<LoginEvent>[] = [
    {
      key: "user",
      header: "用户",
      render: (event) => (
        <button type="button" className="cell-main link-btn" onClick={() => setDetail(event)}>
          <span className="row-ico m-groove">
            <Users size={15} />
          </span>
          <span>
            <span className="cell-title">{event.username || "未知用户"}</span>
            <span className="cell-sub">{event.clientType || "admin"}</span>
          </span>
        </button>
      ),
    },
    { key: "outcome", header: "结果", render: (event) => <StatusTag status={event.outcome} /> },
    {
      key: "ip",
      header: "IP",
      render: (event) => <span className="mono">{event.ipAddress || "—"}</span>,
    },
    {
      key: "reason",
      header: "失败原因",
      render: (event) => <span className="muted">{event.failureReason || "—"}</span>,
    },
    {
      key: "time",
      header: "登录时间",
      render: (event) => <span className="tnum faint">{formatDateTime(event.occurredAt)}</span>,
    },
    {
      key: "actions",
      header: "详情",
      align: "right",
      render: (event) => (
        <IconButton label="查看登录详情" icon="file" small onClick={() => setDetail(event)} />
      ),
    },
  ];

  return (
    <BusinessPage
      title="登录日志"
      description="登录结果、IP、客户端和热力分布来自 Go 登录审计表；隐私字段按后端返回展示。"
    >
      <div className="stack">
        <SearchForm
          loading={loading}
          onSearch={() => {
            setPageNumber(1);
            void load();
          }}
          onReset={() => {
            setKeyword("");
            setOutcome("");
            setPageNumber(1);
          }}
        >
          <Field label="关键词" wide>
            <TextInput
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="用户名 / IP / 客户端"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
          <Field label="结果">
            <select
              className="input-shell"
              value={outcome}
              onChange={(event) => setOutcome(event.target.value)}
            >
              <option value="">全部结果</option>
              <option value="success">成功</option>
              <option value="failure">失败</option>
            </select>
          </Field>
        </SearchForm>

        <SectionBand
          title="登录概览"
          meta={`最近 ${sample.length} 条真实登录记录 · 热力图按星期与小时聚合`}
          actions={<Tag tone="info">audit_login_events</Tag>}
        >
          <div className="readout-strip">
            <Metric label="成功" value={success} size="m" />
            <Metric label="失败" value={failed} size="m" />
            <Metric label="用户数" value={uniqueUsers} size="m" />
            <Metric
              label="异常比例"
              value={sample.length ? Math.round((failed / sample.length) * 100) : 0}
              unit="%"
              size="m"
            />
          </div>
        </SectionBand>

        <SectionBand title="登录热力图" meta="最近 100 条登录事件 · 红色纹理表示异常">
          {cells.length ? (
            <HeatGrid cells={cells} />
          ) : (
            <EmptyState title="暂无热力数据" icon="chart" />
          )}
        </SectionBand>

        <Card className="reveal">
          <div className="panel-head">
            <div>
              <span className="section-label">登录记录</span>
              <span className="panel-meta">{page.total ?? 0} 条</span>
            </div>
            <Tag tone="info">
              <ShieldAlert size={13} /> 隐私字段仅用于审计
            </Tag>
          </div>
          <DataState
            loading={loading}
            error={error}
            permissionDenied={permissionDenied}
            empty={!loading && !error && !rows.length}
            emptyTitle="没有匹配的登录记录"
            onRetry={() => void load()}
          >
            <DataTable columns={columns} rows={rows} rowKey={(event) => event.id ?? ""} />
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
        title="登录事件详情"
        description={detail?.username}
        onClose={() => setDetail(null)}
        width={520}
      >
        {detail ? (
          <div className="stack">
            <div className="drawer-toolbar">
              <StatusTag status={detail.outcome} />
              <Tag tone="info">{detail.clientType || "admin"}</Tag>
            </div>
            <DetailGrid
              items={[
                { label: "事件 ID", value: detail.id, wide: true },
                { label: "用户 ID", value: detail.userId || "—" },
                { label: "登录时间", value: formatDateTime(detail.occurredAt) },
                { label: "IP 地址", value: detail.ipAddress || "—" },
                { label: "失败原因", value: detail.failureReason || "—" },
                { label: "User Agent", value: detail.userAgent || "—", wide: true },
              ]}
            />
            <Card>
              <div className="panel-body">
                <EmptyState
                  title={detail.outcome === "success" ? "登录成功" : "登录失败"}
                  description="该记录由 Go 登录流程写入，不能在前端修改或伪造。"
                  icon={detail.outcome === "success" ? "shield" : "help"}
                />
              </div>
            </Card>
          </div>
        ) : null}
      </Drawer>
    </BusinessPage>
  );
}
