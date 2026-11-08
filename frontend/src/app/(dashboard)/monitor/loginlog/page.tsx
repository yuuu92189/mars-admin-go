import { requirePageAccess } from "@/lib/server/page-access";
import { LoginLogsView } from "@/features/audit/login-logs-view";

export default async function LoginLogsPage() {
  await requirePageAccess("/monitor/loginlog", ["audit:read"]);
  return <LoginLogsView />;
}
