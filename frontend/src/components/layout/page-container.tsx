import type { ReactNode } from "react";

export function PageContainer({ children }: { children: ReactNode }) {
  return (
    <div className="page page-host" data-entrance-ready="true">
      {children}
    </div>
  );
}
