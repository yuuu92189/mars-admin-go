"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Activity, ArrowRight, Database, RefreshCw, ShieldCheck } from "lucide-react";
import { PageContainer } from "@/components/layout/page-container";
import {
  PageHeader,
  Card,
  EmptyState,
  Metric,
  Skeleton,
  Tag,
  Avatar,
} from "@/components/ui/surface";
import { Button } from "@/components/ui/button";
import { Segmented } from "@/components/ui/select";
import { useAuth } from "@/components/auth/auth-provider";
import { BarChart, Swimlane, TickRing, TrendChart } from "@/components/charts/metric-charts";
import { ApiError } from "@/lib/api/client";
import { getDashboard, type DashboardRange } from "@/lib/api/dashboard";
import type { Dashboard } from "@/lib/api/types";
import { useCardEntrance } from "@/lib/motion";
import { buildNavGroups } from "@/components/layout/nav";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const RANGE_OPTIONS = [
  { value: "24h", label: "24 小时" },
  { value: "7d", label: "7 天" },
  { value: "30d", label: "30 天" },
];

const RANGE_LABEL: Record<DashboardRange, string> = {
  "24h": "近 24 小时",
  "7d": "近 7 天",
  "30d": "近 30 天",
};

function greeting(): string {
  const hour = new Date().getHours();
  return hour < 6
    ? "夜深了"
    : hour < 12
      ? "上午好"
      : hour < 14
        ? "中午好"
        : hour < 18
          ? "下午好"
          : "晚上好";
}

function relativeTime(value: string): string {
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) return value;
  const minutes = Math.max(0, Math.round((Date.now() - timestamp) / 60000));
  if (minutes < 1) return "刚刚";
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(value));
}

export function DashboardView() {
  const { user } = useAuth();
  const router = useRouter();
  const [range, setRange] = useState<DashboardRange>("24h");
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const contentRef = useCardEntrance<HTMLDivElement>();
  const requestRef = useRef(0);

  const load = useCallback(async () => {
    const request = ++requestRef.current;
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const data = await getDashboard(range);
      if (request === requestRef.current) setDashboard(data);
    } catch (cause) {
      if (request !== requestRef.current) return;
      if (cause instanceof ApiError && cause.status === 403) {
        setPermissionDenied(true);
      } else {
        setError(cause instanceof ApiError ? cause.message : "运行数据暂时不可用");
      }
    } finally {
      if (request === requestRef.current) setLoading(false);
    }
  }, [range]);

  useDeferredEffect(load);

  const date = useMemo(() => {
    const now = new Date();
    const weekday = "日一二三四五六"[now.getDay()];
    return `${now.getFullYear()} 年 ${now.getMonth() + 1} 月 ${now.getDate()} 日 · 星期${weekday}`;
  }, []);

  const shortcuts = useMemo(() => {
    if (!user) return [];
    return buildNavGroups(user)
      .flatMap((group) =>
        group.items.map((item) => ({
          ...item,
          group: group.label === "导航" ? "常用入口" : group.label,
        })),
      )
      .filter((item) => item.path !== "/dashboard")
      .slice(0, 6);
  }, [user]);

  const loginTotal = dashboard?.loginTotal ?? 0;
  const previousTotal = dashboard?.loginPreviousTotal ?? 0;
  const loginDelta =
    previousTotal > 0
      ? Math.round(((loginTotal - previousTotal) / previousTotal) * 1000) / 10
      : null;
  const peak = Math.max(0, ...(dashboard?.loginTrend.map((point) => point.value) ?? []));
  const average = dashboard?.loginTrend.length
    ? Math.round((loginTotal / dashboard.loginTrend.length) * 10) / 10
    : 0;
  const roleTotal = Math.max(
    1,
    (dashboard?.roleDistribution ?? []).reduce((total, item) => total + item.value, 0),
  );
  const recentActivity = useMemo(
    () => dashboard?.recentActivity ?? [],
    [dashboard?.recentActivity],
  );
  const activityLanes = useMemo(() => {
    if (!recentActivity.length) return [];
    const today = new Date();
    const start = new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime();
    const grouped = new Map<string, Dashboard["recentActivity"]>();
    for (const item of recentActivity) {
      const key = item.target.split(" · ")[0] || "系统操作";
      grouped.set(key, [...(grouped.get(key) ?? []), item]);
    }
    return [...grouped.entries()].map(([label, items]) => ({
      id: label,
      label,
      items: items.map((item) => {
        const time = new Date(item.occurredAt).getTime();
        const second = Math.max(0, Math.min(86400, (time - start) / 1000));
        return {
          id: `${item.who}-${item.occurredAt}`,
          label: `${item.who} ${item.target}`,
          start: Math.max(0, second - 45),
          end: Math.min(86400, second + 45),
          failed: item.outcome !== "success",
        };
      }),
    }));
  }, [recentActivity]);
  const nowSeconds = useMemo(() => {
    const now = new Date();
    return now.getHours() * 3600 + now.getMinutes() * 60 + now.getSeconds();
  }, []);

  return (
    <PageContainer>
      <div ref={contentRef} className="dashboard">
        <PageHeader
          title={`${greeting()}，${user?.user?.displayName ?? user?.user?.username ?? "管理员"}`}
          description="登录脉搏、系统运行概况与真实审计活动。所有图表均来自 Go API 聚合结果。"
        >
          <Tag tone={permissionDenied ? "danger" : "ok"} dot>
            {permissionDenied ? "需要 audit:read 权限" : "真实运行数据"}
          </Tag>
        </PageHeader>
        <span className="page-date reveal">{date}</span>

        {permissionDenied ? (
          <Card className="reveal">
            <EmptyState
              title="当前账号没有运行数据权限"
              description="请联系管理员分配 audit:read。后端仍会独立拒绝未授权请求。"
              icon="shield"
            />
          </Card>
        ) : error ? (
          <Card className="reveal">
            <EmptyState title="运行数据加载失败" description={error} icon="activity" />
            <div style={{ display: "flex", justifyContent: "center", paddingBottom: 20 }}>
              <Button icon="refresh" onClick={() => void load()}>
                重试
              </Button>
            </div>
          </Card>
        ) : (
          <div className="bento-grid">
            <Card className="bento bento-hero m-glow reveal">
              <span className="glow-layer" aria-hidden="true" />
              <div className="hero-head">
                <div>
                  <span className="eyebrow">AUTH / LOGIN PULSE</span>
                  <h2 className="section-mark-title" style={{ marginTop: 8 }}>
                    登录脉搏
                  </h2>
                </div>
                <Segmented
                  value={range}
                  onChange={(value) => setRange(value as DashboardRange)}
                  options={RANGE_OPTIONS}
                  label="时间范围"
                />
              </div>
              <div className="hero-body">
                <div className="hero-figure">
                  <Metric
                    value={loading ? null : loginTotal}
                    unit="次"
                    size="xl"
                    label={RANGE_LABEL[range]}
                  />
                  <div className="hero-delta">
                    <Tag
                      tone={loginDelta == null ? "neutral" : loginDelta >= 0 ? "ok" : "danger"}
                      dot
                    >
                      {loginDelta == null
                        ? "无对比基线"
                        : `${loginDelta >= 0 ? "+" : ""}${loginDelta}%`}
                    </Tag>
                    <span className="faint">较上一等长周期</span>
                  </div>
                  <dl className="hero-readouts">
                    <div>
                      <dt>峰值</dt>
                      <dd className="tnum">{loading ? "—" : peak}</dd>
                    </div>
                    <div>
                      <dt>均值</dt>
                      <dd className="tnum">{loading ? "—" : average}</dd>
                    </div>
                    <div>
                      <dt>范围</dt>
                      <dd>{RANGE_LABEL[range]}</dd>
                    </div>
                  </dl>
                </div>
                <div className="hero-chart">
                  {loading ? (
                    <div className="stack" style={{ paddingTop: 32 }}>
                      <Skeleton height={18} />
                      <Skeleton width="78%" height={18} />
                      <Skeleton width="62%" height={18} />
                    </div>
                  ) : dashboard?.loginTrend.length ? (
                    <TrendChart
                      values={dashboard.loginTrend.map((point) => point.value)}
                      labels={dashboard.loginTrend.map((point) => point.label)}
                      height={212}
                      unit=" 次"
                      ariaLabel={`${RANGE_LABEL[range]}登录趋势`}
                    />
                  ) : (
                    <EmptyState title="所选时间范围暂无登录记录" icon="clock" />
                  )}
                </div>
              </div>
              <div className="hero-actions">
                <Button
                  variant="primary"
                  icon="refresh"
                  loading={loading}
                  onClick={() => void load()}
                >
                  刷新数据
                </Button>
                <span className="faint" style={{ alignSelf: "center", fontSize: 12 }}>
                  时区：Asia/Shanghai · 聚合自 audit_login_events
                </span>
              </div>
            </Card>

            <Card className="bento bento-side reveal">
              <div className="side-top">
                <span className="side-label">在线会话 · 当前</span>
                <span className="side-ico m-groove">
                  <Activity size={16} />
                </span>
              </div>
              <Metric value={loading ? null : dashboard?.onlineSessions} unit="人" size="l" />
              <div className="side-foot">
                <span className="muted">未撤销且未过期</span>
                <Tag tone="ok" dot>
                  在线
                </Tag>
              </div>
            </Card>
            <Card className="bento bento-side reveal">
              <div className="side-top">
                <span className="side-label">操作成功率 · {RANGE_LABEL[range]}</span>
                <span className="side-ico m-groove">
                  <ShieldCheck size={16} />
                </span>
              </div>
              <Metric value={loading ? null : dashboard?.operationSuccessRate} unit="%" size="l" />
              <div className="side-foot">
                <span className="muted">{dashboard?.operationTotal ?? 0} 次操作</span>
                <Tag tone="info">Go Audit</Tag>
              </div>
            </Card>

            <Card className="bento bento-third reveal">
              <div className="bento-head">
                <span className="section-label">近 7 天操作量</span>
                <span className="bento-meta tnum">
                  {dashboard?.operationTrend.reduce((total, point) => total + point.value, 0) ?? 0}{" "}
                  次
                </span>
              </div>
              {loading ? (
                <Skeleton height={132} />
              ) : dashboard?.operationTrend.length ? (
                <BarChart
                  values={dashboard.operationTrend.map((point) => point.value)}
                  failed={dashboard.operationTrend.map((point) => point.failed ?? 0)}
                  labels={dashboard.operationTrend.map((point, index) =>
                    index === dashboard.operationTrend.length - 1 ? "今天" : point.label,
                  )}
                  highlight={dashboard.operationTrend.length - 1}
                  height={154}
                  unit=" 次"
                  ariaLabel="近 7 天操作量"
                />
              ) : (
                <EmptyState title="暂无操作记录" icon="chart" />
              )}
            </Card>

            <Card className="bento bento-third bento-ring reveal">
              <div className="bento-head">
                <span className="section-label">操作成功率</span>
                <span className="bento-meta">{RANGE_LABEL[range]}</span>
              </div>
              <div className="ring-wrap">
                {loading ? (
                  <Skeleton width="164px" height={164} />
                ) : (
                  <TickRing value={dashboard?.operationSuccessRate} label="成功" size={164} />
                )}
                <dl className="ring-readouts">
                  <div>
                    <dt>操作</dt>
                    <dd className="tnum">{loading ? "—" : dashboard?.operationTotal}</dd>
                  </div>
                  <div>
                    <dt>成功率</dt>
                    <dd className="tnum">
                      {loading ? "—" : `${dashboard?.operationSuccessRate ?? 0}%`}
                    </dd>
                  </div>
                </dl>
              </div>
            </Card>

            <Card className="bento bento-third reveal">
              <div className="bento-head">
                <span className="section-label">数据权限分布</span>
                <span className="bento-meta">按角色用户数</span>
              </div>
              {loading ? (
                <div className="stack">
                  <Skeleton />
                  <Skeleton width="82%" />
                  <Skeleton width="64%" />
                </div>
              ) : dashboard?.roleDistribution.length ? (
                <ul className="share-list">
                  {dashboard.roleDistribution.map((item, index) => (
                    <li key={item.label} className={`cat-${(index % 5) + 1}`}>
                      <span className="share-label">{item.label}</span>
                      <span className="share-track">
                        <i
                          style={{
                            width: `${Math.max(4, (item.value / roleTotal) * 100)}%`,
                          }}
                        />
                      </span>
                      <span className="share-value tnum">
                        {item.value}
                        <small> 人 · {Math.round((item.value / roleTotal) * 100)}%</small>
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <EmptyState title="暂无角色数据" icon="role" />
              )}
            </Card>

            <Card className="bento bento-wide reveal">
              <div className="bento-head">
                <span className="section-label">今日操作泳道</span>
                <span className="bento-meta">RECENT AUDIT EVENTS</span>
              </div>
              {activityLanes.length ? (
                <Swimlane lanes={activityLanes} now={nowSeconds} compact />
              ) : (
                <EmptyState title="今天还没有操作记录" icon="clock" />
              )}
            </Card>

            <Card className="bento bento-narrow dash-login-card reveal">
              <div className="bento-head">
                <span className="section-label">最近登录</span>
                <Database size={16} className="soft" />
              </div>
              <div className="activity-list dash-login-list">
                {loading ? (
                  <div className="stack">
                    <Skeleton />
                    <Skeleton width="76%" />
                    <Skeleton width="62%" />
                  </div>
                ) : dashboard?.recentLogins.length ? (
                  dashboard.recentLogins.map((item) => (
                    <div className="activity" key={`${item.who}-${item.occurredAt}`}>
                      <Avatar name={item.who} size={30} />
                      <div className="activity-main">
                        <div className="ellipsis">
                          <b>{item.who}</b> <span className="muted">{item.target}</span>
                        </div>
                        <div className="faint mono-time">{relativeTime(item.occurredAt)}</div>
                      </div>
                      <Tag tone={item.outcome === "success" ? "ok" : "danger"} dot>
                        {item.outcome === "success" ? "成功" : "失败"}
                      </Tag>
                    </div>
                  ))
                ) : (
                  <EmptyState title="暂无登录记录" icon="clock" />
                )}
              </div>
            </Card>

            {shortcuts.length ? (
              <Card className="bento bento-full reveal">
                <div className="bento-head">
                  <span className="section-label">常用入口</span>
                </div>
                <div className="shortcut-grid">
                  {shortcuts.map((item) => (
                    <button
                      key={item.key}
                      type="button"
                      className="quick-link lift m-surface"
                      onClick={() => router.push(item.path)}
                    >
                      <span className="quick-ico m-groove">
                        <Activity size={17} />
                      </span>
                      <span className="quick-text">
                        <span className="quick-title">{item.label}</span>
                        <span className="quick-sub">{item.group}</span>
                      </span>
                      <ArrowRight size={15} className="quick-go" />
                    </button>
                  ))}
                </div>
              </Card>
            ) : null}
          </div>
        )}
      </div>
    </PageContainer>
  );
}
