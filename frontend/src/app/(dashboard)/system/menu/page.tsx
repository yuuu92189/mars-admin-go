import { requirePageAccess } from "@/lib/server/page-access";
import { MenusView } from "@/features/menus/menus-view";

export default async function MenusPage() {
  await requirePageAccess("/system/menu", ["role:read"]);
  return <MenusView />;
}
