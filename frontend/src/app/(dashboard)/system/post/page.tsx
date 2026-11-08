import { requirePageAccess } from "@/lib/server/page-access";
import { PositionsView } from "@/features/positions/positions-view";

export default async function PositionsPage() {
  await requirePageAccess("/system/post", ["department:read"]);
  return <PositionsView />;
}
