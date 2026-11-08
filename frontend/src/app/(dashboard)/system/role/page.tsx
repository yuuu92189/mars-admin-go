import { requirePageAccess } from "@/lib/server/page-access";
import { RolesView } from "@/features/roles/roles-view";

export default async function RolesPage({
  searchParams,
}: {
  searchParams: Promise<{ trace?: string }>;
}) {
  await requirePageAccess("/system/role", ["role:read"]);
  const params = await searchParams;
  return <RolesView traceFocus={params.trace ?? null} />;
}
