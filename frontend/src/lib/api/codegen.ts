import { apiDownload, apiRequest } from "./client";
import type {
  CodegenOutput,
  CodegenTable,
  ImportedCodegenTable,
  ImportedCodegenTablePage,
} from "./types";

export function listDatabaseTables(schema = "public"): Promise<CodegenTable[]> {
  const params = new URLSearchParams({ schema });
  return apiRequest<CodegenTable[]>(`/api/v1/admin/codegen/tables?${params.toString()}`);
}

export function listImportedTables(
  query = "",
  page = 1,
  pageSize = 20,
): Promise<ImportedCodegenTablePage> {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });
  if (query.trim()) params.set("query", query.trim());
  return apiRequest<ImportedCodegenTablePage>(
    `/api/v1/admin/codegen/imported?${params.toString()}`,
  );
}

export function importTables(schema: string, tables: string[]): Promise<ImportedCodegenTable[]> {
  return apiRequest<ImportedCodegenTable[]>("/api/v1/admin/codegen/imported", {
    method: "POST",
    body: { schema, tables },
  });
}

export function getImportedTable(id: string): Promise<ImportedCodegenTable> {
  return apiRequest<ImportedCodegenTable>(
    `/api/v1/admin/codegen/imported/${encodeURIComponent(id)}`,
  );
}

export function updateImportedTable(
  id: string,
  input: ImportedCodegenTable,
): Promise<ImportedCodegenTable> {
  return apiRequest<ImportedCodegenTable>(
    `/api/v1/admin/codegen/imported/${encodeURIComponent(id)}`,
    { method: "PUT", body: input },
  );
}

export function deleteImportedTable(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/codegen/imported/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function previewImportedTable(id: string): Promise<CodegenOutput> {
  return apiRequest<CodegenOutput>(
    `/api/v1/admin/codegen/imported/${encodeURIComponent(id)}/preview`,
  );
}

export function syncImportedTable(id: string): Promise<ImportedCodegenTable> {
  return apiRequest<ImportedCodegenTable>(
    `/api/v1/admin/codegen/imported/${encodeURIComponent(id)}/sync`,
    { method: "POST" },
  );
}

export function downloadGeneratedCode(ids: string[]) {
  const params = new URLSearchParams({ ids: ids.join(",") });
  return apiDownload(`/api/v1/admin/codegen/download?${params.toString()}`);
}
