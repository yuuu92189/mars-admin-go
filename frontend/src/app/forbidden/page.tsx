import Link from "next/link";
import { EmptyState } from "@/components/ui/surface";

export default function ForbiddenPage() {
  return (
    <main className="stage" style={{ display: "grid", placeItems: "center" }}>
      <EmptyState title="没有访问权限" description="当前账号没有访问该页面的权限。" />
      <Link href="/dashboard" className="btn btn-secondary">
        返回工作台
      </Link>
    </main>
  );
}
