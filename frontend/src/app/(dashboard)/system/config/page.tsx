import { requirePageAccess } from "@/lib/server/page-access";
import { SettingsView } from "@/features/settings/settings-view";

export default async function SettingsPage() {
  await requirePageAccess("/system/config", ["settings:read"]);
  return <SettingsView />;
}
