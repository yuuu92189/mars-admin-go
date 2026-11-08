import { cookies } from "next/headers";
import { notFound } from "next/navigation";
import { PageContainer } from "@/components/layout/page-container";
import { PageHeader, Card, EmptyState, Tag } from "@/components/ui/surface";
import { getCurrentUserFromCookie } from "@/lib/api/server-auth";
import { isVisibleMenuPath } from "@/lib/navigation";

export default async function PendingFeaturePage({
  params,
}: {
  params: Promise<{ path: string[] }>;
}) {
  const { path } = await params;
  const pathname = `/${path.join("/")}`;
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map(({ name, value }) => `${name}=${value}`)
    .join("; ");
  const user = await getCurrentUserFromCookie(cookieHeader);
  if (!isVisibleMenuPath(user, pathname)) notFound();

  return (
    <PageContainer>
      <PageHeader
        title="页面待迁移"
        description="Phase 2 只交付前端底座、登录、工作台和主布局；该业务页面将在 Phase 3 按 Vue 页面清单逐项迁移。"
      >
        <Tag tone="info">Phase 3</Tag>
      </PageHeader>
      <Card className="reveal">
        <EmptyState
          title="该路由已预留"
          description={`${pathname} 的真实业务组件、数据接入和权限分支尚未迁移。`}
          icon="workflow"
        />
      </Card>
    </PageContainer>
  );
}
