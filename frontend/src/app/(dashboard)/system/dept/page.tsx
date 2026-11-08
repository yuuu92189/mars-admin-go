import { requirePageAccess } from "@/lib/server/page-access";
import { DepartmentsView } from "@/features/departments/departments-view";

export default async function DepartmentsPage() {
  await requirePageAccess("/system/dept", ["department:read"]);
  return <DepartmentsView />;
}
