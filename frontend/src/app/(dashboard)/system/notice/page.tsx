import { requirePageAccess } from "@/lib/server/page-access";
import { NoticesView } from "@/features/notices/notices-view";

export default async function NoticesPage() {
  await requirePageAccess("/system/notice", ["notice:read"]);
  return <NoticesView />;
}
