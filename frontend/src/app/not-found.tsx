import Link from "next/link";
import { EmptyState } from "@/components/ui/surface";

export default function NotFoundPage() {
  return (
    <main className="stage" style={{ display: "grid", placeItems: "center" }}>
      <EmptyState title="页面不存在" description="这个地址没有对应的页面。" icon="help" />
      <Link href="/dashboard" className="btn btn-secondary">
        返回工作台
      </Link>
    </main>
  );
}
