import { requirePageAccess } from "@/lib/server/page-access";
import { DictionariesView } from "@/features/dictionaries/dictionaries-view";

export default async function DictionariesPage() {
  await requirePageAccess("/system/dict", ["dictionary:read"]);
  return <DictionariesView />;
}
