import { requirePageAccess } from "@/lib/server/page-access";
import { CodegenView } from "@/features/codegen/codegen-view";

export default async function CodegenPage() {
  await requirePageAccess("/tool/generator", ["codegen:read"]);
  return <CodegenView />;
}
