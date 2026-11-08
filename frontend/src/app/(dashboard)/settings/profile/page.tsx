"use client";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader, Card, EmptyState, Tag } from "@/components/ui/surface";
import { useAuth } from "@/components/auth/auth-provider";

export default function ProfilePage() {
  const { user } = useAuth();
  return (
    <PageContainer>
      <PageHeader title="个人资料" description="当前账号信息来自 Go API 的 /api/v1/auth/me。" />
      <Card className="reveal">
        {user?.user ? (
          <div className="kv-grid" style={{ margin: 20 }}>
            <div className="kv">
              <div className="kv-label">显示名称</div>
              <div className="kv-value">{user.user.displayName}</div>
            </div>
            <div className="kv">
              <div className="kv-label">账号</div>
              <div className="kv-value">@{user.user.username}</div>
            </div>
            <div className="kv">
              <div className="kv-label">状态</div>
              <div className="kv-value">
                <Tag tone="ok" dot>
                  {user.user.status}
                </Tag>
              </div>
            </div>
            <div className="kv">
              <div className="kv-label">角色</div>
              <div className="kv-value">{user.roles?.join(" · ") || "—"}</div>
            </div>
          </div>
        ) : (
          <EmptyState title="当前用户信息不可用" icon="users" />
        )}
      </Card>
    </PageContainer>
  );
}
