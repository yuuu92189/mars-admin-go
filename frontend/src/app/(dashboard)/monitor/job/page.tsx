import { requirePageAccess } from "@/lib/server/page-access";
import { SchedulerView } from "@/features/scheduler/scheduler-view";

export default async function SchedulerPage() {
  await requirePageAccess("/monitor/job", ["scheduler:read"]);
  return <SchedulerView />;
}
