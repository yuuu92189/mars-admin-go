"use client";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/surface";

export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <main className="stage" style={{ display: "grid", placeItems: "center" }}>
      <EmptyState
        title="页面出现错误"
        description="请重试；如果问题持续，请检查 Go API 连接。"
        icon="activity"
      />
      <Button variant="secondary" onClick={reset}>
        重新加载
      </Button>
    </main>
  );
}
