import { requirePageAccess } from "@/lib/server/page-access";
import { OperationLogsView } from "@/features/audit/operation-logs-view";

export default async function OperationLogsPage() {
  await requirePageAccess("/monitor/operlog", ["audit:read"]);
  return <OperationLogsView />;
}
