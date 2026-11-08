"use client";

import { useEffect, useMemo, useState } from "react";
import { Activity, ArrowRight, Database, ShieldCheck } from "lucide-react";
import { PageContainer } from "@/components/layout/page-container";
import { PageHeader, Card, EmptyState, Metric, Skeleton, Tag } from "@/components/ui/surface";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/components/auth/auth-provider";
import { ApiError } from "@/lib/api/client";
import { getDashboardCounters } from "@/lib/api/dashboard";
import type { DashboardCounters } from "@/lib/api/types";
import { useCardEntrance } from "@/lib/motion";

const EMPTY_COUNTERS: DashboardCounters = {};

export function DashboardView() {
  const { user } = useAuth();
  const [counters, setCounters] = useState<DashboardCounters>(EMPTY_COUNTERS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const contentRef = useCardEntrance<HTMLDivElement>();

  useEffect(() => {
    let active = true;
    getDashboardCounters()
      .then((data) => {
        if (active) setCounters(data);
      })
      .catch((cause: unknown) => {
        if (!active) return;
        if (cause instanceof ApiError && cause.status === 403) {
          setPermissionDenied(true);
          return;
        }
        setError(cause instanceof ApiError ? cause.message : "运行数据暂时不可用");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const date = useMemo(() => {
    const now = new Date();
    const weekday = "日一二三四五六"[now.getDay()];
    return `${now.getFullYear()} 年 ${now.getMonth() + 1} 月 ${now.getDate()} 日 · 星期${weekday}`;
  }, []);

  const greeting = useMemo(() => {
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
  }, []);

  return (
    <PageContainer>
      <div ref={contentRef}>
        <PageHeader
          title={`${greeting}，${user?.user?.displayName ?? user?.user?.username ?? "管理员"}`}
          description="系统运行概况与治理状态。这里只展示 Go API 返回的真实数据。"
        >
          <Tag tone={permissionDenied ? "danger" : "ok"} dot>
            {permissionDenied ? "需要 audit:read 权限" : "运行数据已接入"}
          </Tag>
        </PageHeader>

        {permissionDenied ? (
          <Card className="reveal">
            <EmptyState
              title="当前账号没有运行数据权限"
              description="请联系管理员分配 audit:read。"
              icon="shield"
            />
          </Card>
        ) : error ? (
          <Card className="reveal">
            <EmptyState title="运行数据加载失败" description={error} icon="activity" />
          </Card>
        ) : (
          <div className="bento-grid">
            <Card className="bento bento-hero reveal">
              <div className="hero-head">
                <div>
                  <span className="eyebrow">SYSTEM / OVERVIEW</span>
                  <h2 className="section-mark-title" style={{ marginTop: 8 }}>
                    系统运行概况
                  </h2>
                </div>
                <Tag tone="ok" dot>
                  实时
                </Tag>
              </div>
              <div className="hero-body">
                <div className="hero-figure">
                  <Metric
                    value={loading ? null : (counters.userCount ?? counters.users ?? 0)}
                    unit="人"
                    size="xl"
                    label="用户总数"
                  />
                  <div className="hero-delta">
                    <Tag tone="info">Cookie Session</Tag>
                    <span className="faint">服务端会话与 CSRF 防护已启用</span>
                  </div>
                </div>
                <div className="hero-readouts">
                  <div>
                    <dt>在线会话</dt>
                    <dd>{loading ? <Skeleton width="56px" /> : (counters.onlineSessions ?? 0)}</dd>
                  </div>
                  <div>
                    <dt>消息</dt>
                    <dd>{loading ? <Skeleton width="56px" /> : (counters.messages ?? 0)}</dd>
                  </div>
                  <div>
                    <dt>文件</dt>
                    <dd>{loading ? <Skeleton width="56px" /> : (counters.files ?? 0)}</dd>
                  </div>
                </div>
              </div>
              <div className="hero-actions">
                <Button variant="secondary" icon="refresh" onClick={() => window.location.reload()}>
                  刷新数据
                </Button>
                <span className="faint" style={{ alignSelf: "center", fontSize: 12 }}>
                  {date}
                </span>
              </div>
            </Card>

            <Card className="bento bento-side reveal">
              <div className="side-top">
                <span className="side-label">在线用户 · 当前</span>
                <span className="side-ico m-groove">
                  <Activity size={16} />
                </span>
              </div>
              <Metric value={loading ? null : (counters.onlineSessions ?? 0)} unit="人" size="l" />
              <div className="side-foot">
                <span className="muted">有效 Session</span>
                <Tag tone="ok" dot>
                  在线
                </Tag>
              </div>
            </Card>
            <Card className="bento bento-side reveal">
              <div className="side-top">
                <span className="side-label">消息记录 · 全量</span>
                <span className="side-ico m-groove">
                  <Database size={16} />
                </span>
              </div>
              <Metric value={loading ? null : (counters.messages ?? 0)} unit="条" size="l" />
              <div className="side-foot">
                <span className="muted">消息服务已接入</span>
                <ArrowRight size={15} className="quick-go" />
              </div>
            </Card>

            <Card className="bento bento-third reveal">
              <div className="bento-head">
                <span className="section-label">权限与菜单</span>
                <span className="bento-meta">GOVERNANCE</span>
              </div>
              <div className="readout-strip">
                <Metric label="角色" value={loading ? null : (counters.roleCount ?? 0)} size="m" />
                <Metric label="菜单" value={loading ? null : (counters.menuCount ?? 0)} size="m" />
                <Metric
                  label="权限"
                  value={loading ? null : (counters.permissionCount ?? 0)}
                  size="m"
                />
              </div>
            </Card>
            <Card className="bento bento-third reveal">
              <div className="bento-head">
                <span className="section-label">通知与文件</span>
                <span className="bento-meta">CONTENT</span>
              </div>
              <div className="readout-strip">
                <Metric
                  label="已发布通知"
                  value={loading ? null : (counters.publishedNotices ?? 0)}
                  size="m"
                />
                <Metric label="文件" value={loading ? null : (counters.files ?? 0)} size="m" />
              </div>
            </Card>
            <Card className="bento bento-third reveal">
              <div className="bento-head">
                <span className="section-label">Outbox 队列</span>
                <span className="bento-meta">DELIVERY</span>
              </div>
              <div className="readout-strip">
                <Metric
                  label="待处理"
                  value={loading ? null : (counters.outboxPending ?? 0)}
                  size="m"
                />
                <Metric label="死信" value={loading ? null : (counters.outboxDead ?? 0)} size="m" />
              </div>
            </Card>

            <Card className="bento bento-wide reveal">
              <div className="bento-head">
                <span className="section-label">安全基线</span>
                <ShieldCheck size={17} className="soft" />
              </div>
              <div className="share-list">
                <li>
                  <span className="share-label">认证</span>
                  <span className="share-track">
                    <i style={{ width: "100%" }} />
                  </span>
                  <span className="share-value">HttpOnly Cookie</span>
                </li>
                <li>
                  <span className="share-label">CSRF</span>
                  <span className="share-track">
                    <i style={{ width: "100%" }} />
                  </span>
                  <span className="share-value">X-CSRF-Token</span>
                </li>
                <li>
                  <span className="share-label">授权</span>
                  <span className="share-track">
                    <i style={{ width: "100%" }} />
                  </span>
                  <span className="share-value">Go API</span>
                </li>
              </div>
            </Card>
            <Card className="bento bento-narrow reveal">
              <div className="bento-head">
                <span className="section-label">数据说明</span>
              </div>
              <p className="muted" style={{ fontSize: 12.5, lineHeight: 1.7 }}>
                当前 canonical Dashboard API 只提供计数，不包含 Vue
                参考页面的趋势、泳道与漏洞统计。缺失模块保持空状态，不使用演示数据填充。
              </p>
            </Card>
          </div>
        )}
      </div>
    </PageContainer>
  );
}
