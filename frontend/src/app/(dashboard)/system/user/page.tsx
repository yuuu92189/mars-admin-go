import { requirePageAccess } from "@/lib/server/page-access";
import { UsersView } from "@/features/users/users-view";

export default async function UsersPage() {
  await requirePageAccess("/system/user", ["user:read"]);
  return <UsersView />;
}
