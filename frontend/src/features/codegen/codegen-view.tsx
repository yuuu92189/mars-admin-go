"use client";

import { useCallback, useMemo, useState } from "react";
import { Braces, Check, Code2, Download, FileCode2, RefreshCw, Upload } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import {
  BusinessPage,
  DataState,
  DetailGrid,
  SearchForm,
  StatusTag,
  formatDateTime,
} from "@/components/business/business-page";
import { Button, IconButton } from "@/components/ui/button";
import { DataTable, Pagination, Tabs, type TableColumn } from "@/components/ui/data-table";
import { Checkbox, Field, TextInput } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Dialog, Drawer } from "@/components/ui/overlay";
import { Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import {
  deleteImportedTable,
  downloadGeneratedCode,
  getImportedTable,
  importTables,
  listDatabaseTables,
  listImportedTables,
  previewImportedTable,
  syncImportedTable,
  updateImportedTable,
} from "@/lib/api/codegen";
import { hasPermission } from "@/lib/permissions";
import type {
  CodegenOutput,
  CodegenTable,
  ImportedCodegenTable,
  ImportedCodegenTablePage,
} from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 12;
const METADATA_FIELDS: Array<[keyof ImportedCodegenTable, string]> = [
  ["className", "类名"],
  ["packageName", "包名"],
  ["moduleName", "模块名"],
  ["businessName", "业务名"],
  ["functionName", "功能名"],
  ["author", "作者"],
  ["genType", "生成类型"],
  ["genPath", "生成路径"],
  ["frontType", "前端类型"],
  ["formLayout", "表单布局"],
  ["parentMenuId", "父菜单 ID"],
];

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function triggerDownload(blob: Blob, filename?: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename || "mars-admin-codegen.zip";
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function Pipeline() {
  return (
    <div className="pipeline reveal" aria-label="代码生成流水线">
      {[
        { icon: Braces, title: "读取数据库表", desc: "PostgreSQL catalog" },
        { icon: Upload, title: "导入配置", desc: "真实 imported table" },
        { icon: FileCode2, title: "预览 / ZIP", desc: "Go Codegen output" },
      ].map((step, index) => (
        <div className="pipeline-step" key={step.title}>
          <span className="pipeline-index">0{index + 1}</span>
          <span className="quick-ico m-groove">
            <step.icon size={18} />
          </span>
          <span>
            <span className="pipeline-title">{step.title}</span>
            <span className="pipeline-desc">{step.desc}</span>
          </span>
          {index < 2 ? <span className="pipeline-link" aria-hidden="true" /> : null}
        </div>
      ))}
    </div>
  );
}

export function CodegenView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "codegen:manage");
  const [page, setPage] = useState<ImportedCodegenTablePage>({
    items: [],
    total: 0,
    page: 1,
    pageSize: PAGE_SIZE,
  });
  const [pageNumber, setPageNumber] = useState(1);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [view, setView] = useState<"imported" | "database">("imported");
  const [databaseTables, setDatabaseTables] = useState<CodegenTable[]>([]);
  const [databaseLoading, setDatabaseLoading] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [importQuery, setImportQuery] = useState("");
  const [importSelected, setImportSelected] = useState<Set<string>>(new Set());
  const [importing, setImporting] = useState(false);
  const [editor, setEditor] = useState<ImportedCodegenTable | null>(null);
  const [editorTab, setEditorTab] = useState<"metadata" | "preview">("metadata");
  const [editorForm, setEditorForm] = useState<ImportedCodegenTable | null>(null);
  const [preview, setPreview] = useState<CodegenOutput | null>(null);
  const [previewFile, setPreviewFile] = useState("");
  const [previewLoading, setPreviewLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ImportedCodegenTable | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [downloadIds, setDownloadIds] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      setPage(await listImportedTables(query, pageNumber, PAGE_SIZE));
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [pageNumber, query]);

  const loadDatabaseTables = useCallback(async () => {
    setDatabaseLoading(true);
    try {
      setDatabaseTables(await listDatabaseTables("public"));
    } catch (cause) {
      pushToast({ title: "数据库表读取失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setDatabaseLoading(false);
    }
  }, [pushToast]);

  useDeferredEffect(load);
  const loadDatabaseOnDemand = useCallback(() => {
    if (view === "database" || importOpen) return loadDatabaseTables();
  }, [importOpen, loadDatabaseTables, view]);
  useDeferredEffect(loadDatabaseOnDemand);

  const filteredDatabaseTables = useMemo(() => {
    const normalized = importQuery.trim().toLowerCase();
    if (!normalized) return databaseTables;
    return databaseTables.filter((table) =>
      [table.name, table.comment]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(normalized)),
    );
  }, [databaseTables, importQuery]);

  const openImport = () => {
    setImportSelected(new Set());
    setImportQuery("");
    setImportOpen(true);
  };

  const submitImport = async () => {
    if (!importSelected.size) return;
    setImporting(true);
    try {
      const imported = await importTables("public", [...importSelected]);
      pushToast({ title: `已导入 ${imported.length} 张表`, tone: "ok" });
      setImportOpen(false);
      setView("imported");
      await load();
      if (imported[0]) await openEditor(imported[0]);
    } catch (cause) {
      pushToast({ title: "表导入失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setImporting(false);
    }
  };

  const openEditor = async (table: ImportedCodegenTable) => {
    setEditor(table);
    setEditorTab("metadata");
    setEditorForm(table);
    setPreview(null);
    setPreviewFile("");
    try {
      const latest = await getImportedTable(table.id);
      setEditor(latest);
      setEditorForm(latest);
    } catch {
      // The row already contains the server-returned imported table.
    }
  };

  const loadPreview = async () => {
    if (!editor) return;
    setPreviewLoading(true);
    try {
      const output = await previewImportedTable(editor.id);
      setPreview(output);
      setPreviewFile(output.files[0]?.path ?? "");
      setEditorTab("preview");
    } catch (cause) {
      pushToast({ title: "代码预览失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setPreviewLoading(false);
    }
  };

  const saveMetadata = async () => {
    if (!editorForm) return;
    setSaving(true);
    try {
      const updated = await updateImportedTable(editorForm.id, editorForm);
      setEditor(updated);
      setEditorForm(updated);
      pushToast({ title: "代码生成配置已保存", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "配置保存失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setSaving(false);
    }
  };

  const sync = async () => {
    if (!editor) return;
    setActionLoading(true);
    try {
      const updated = await syncImportedTable(editor.id);
      setEditor(updated);
      setEditorForm(updated);
      pushToast({ title: "数据库结构已同步", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "结构同步失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const syncTable = async (table: ImportedCodegenTable) => {
    setActionLoading(true);
    try {
      const updated = await syncImportedTable(table.id);
      setEditor(updated);
      setEditorForm(updated);
      pushToast({ title: "数据库结构已同步", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "结构同步失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const download = async (ids: string[]) => {
    if (!ids.length) return;
    setActionLoading(true);
    try {
      const result = await downloadGeneratedCode(ids);
      triggerDownload(result.blob, result.filename);
      pushToast({ title: "ZIP 已生成", tone: "ok" });
    } catch (cause) {
      pushToast({ title: "ZIP 下载失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const remove = async () => {
    if (!deleteTarget) return;
    setActionLoading(true);
    try {
      await deleteImportedTable(deleteTarget.id);
      pushToast({ title: "导入配置已删除", tone: "ok" });
      setDeleteTarget(null);
      if (editor?.id === deleteTarget.id) setEditor(null);
      await load();
    } catch (cause) {
      pushToast({ title: "删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const columns: TableColumn<ImportedCodegenTable>[] = [
    {
      key: "table",
      header: "数据库表",
      render: (table) => (
        <button type="button" className="cell-main link-btn" onClick={() => void openEditor(table)}>
          <span className="row-ico m-groove">
            <Code2 size={15} />
          </span>
          <span>
            <span className="cell-title">{table.tableName}</span>
            <span className="cell-sub">{table.tableComment || "无表说明"}</span>
          </span>
        </button>
      ),
    },
    {
      key: "module",
      header: "模块 / 业务",
      render: (table) => (
        <span>
          {table.moduleName || "—"} / {table.businessName || "—"}
        </span>
      ),
    },
    {
      key: "columns",
      header: "字段",
      align: "right",
      render: (table) => <span className="tnum">{table.columns.length}</span>,
    },
    {
      key: "type",
      header: "生成类型",
      render: (table) => <span className="code-chip">{table.genType || "crud"}</span>,
    },
    {
      key: "updated",
      header: "更新时间",
      render: (table) => <span className="tnum faint">{formatDateTime(table.updatedAt)}</span>,
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (table) => (
        <div className="row-actions">
          <IconButton
            label="预览代码"
            icon="codegen"
            small
            onClick={() => void openEditor(table)}
          />
          {canManage ? (
            <>
              <IconButton
                label="同步结构"
                icon="refresh"
                small
                onClick={() => void syncTable(table)}
              />
              <IconButton
                label="删除配置"
                icon="trash"
                small
                danger
                onClick={() => setDeleteTarget(table)}
              />
            </>
          ) : null}
        </div>
      ),
    },
  ];

  const selectedPreviewFile =
    preview?.files.find((file) => file.path === previewFile) ?? preview?.files[0];

  return (
    <BusinessPage
      title="代码生成器"
      description="数据库表读取、导入配置、字段元数据、代码预览和真实 ZIP 下载全部来自 Go Codegen API。"
      actions={
        <>
          <Tabs
            tabs={[
              { value: "imported", label: "已导入" },
              { value: "database", label: "数据库表" },
            ]}
            value={view}
            onChange={(value) => setView(value as "imported" | "database")}
          />
          {canManage ? (
            <Button variant="primary" icon="plus" onClick={openImport}>
              导入数据库表
            </Button>
          ) : null}
        </>
      }
    >
      <div className="stack">
        <Pipeline />
        <SearchForm
          loading={loading}
          onSearch={() => {
            setPageNumber(1);
            void load();
          }}
          onReset={() => {
            setQuery("");
            setPageNumber(1);
          }}
        >
          <Field label="搜索已导入表" wide>
            <TextInput
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="表名 / 说明"
              icon="search"
              clearable
              onClear={() => setQuery("")}
            />
          </Field>
        </SearchForm>

        {view === "imported" ? (
          <Card className="reveal">
            <div className="panel-head">
              <div>
                <span className="section-label">已导入表配置</span>
                <span className="panel-meta">{page.total ?? 0} 张表</span>
              </div>
              {downloadIds.size ? (
                <Button
                  variant="primary"
                  icon="download"
                  loading={actionLoading}
                  onClick={() => void download([...downloadIds])}
                >
                  下载选中 ZIP
                </Button>
              ) : null}
            </div>
            <DataState
              loading={loading}
              error={error}
              permissionDenied={permissionDenied}
              empty={!loading && !error && !(page.items ?? []).length}
              emptyTitle="还没有导入数据库表"
              emptyDescription="点击“导入数据库表”开始真实生成配置。"
              onRetry={() => void load()}
            >
              <DataTable
                columns={columns}
                rows={page.items ?? []}
                rowKey={(table) => table.id}
                selectable={canManage}
                selected={downloadIds}
                onSelectionChange={setDownloadIds}
              />
              <Pagination
                page={pageNumber}
                pageSize={PAGE_SIZE}
                total={page.total ?? 0}
                onChange={setPageNumber}
              />
            </DataState>
          </Card>
        ) : (
          <Card className="reveal">
            <div className="panel-head">
              <span className="section-label">PostgreSQL 数据库表</span>
              <Button
                icon="refresh"
                loading={databaseLoading}
                onClick={() => void loadDatabaseTables()}
              >
                刷新
              </Button>
            </div>
            <div className="panel-body">
              {databaseLoading ? <EmptyState title="正在读取数据库表" icon="database" /> : null}
              {!databaseLoading && !databaseTables.length ? (
                <EmptyState title="没有可读取的数据表" icon="database" />
              ) : null}
              <div className="import-list">
                {databaseTables.map((table) => (
                  <div className="import-row" key={`${table.schema}.${table.name}`}>
                    <span className="row-ico m-groove">
                      <Braces size={15} />
                    </span>
                    <div className="cell-main" style={{ flex: 1 }}>
                      <span className="cell-title">{table.name}</span>
                      <span className="cell-sub">{table.comment || "无表说明"}</span>
                    </div>
                    <Tag tone="info">{table.columns?.length ?? 0} 字段</Tag>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        )}
      </div>

      <Dialog
        open={importOpen}
        title="导入数据库表"
        description="选择真实 PostgreSQL 表并生成 Go Codegen 导入配置。"
        onClose={() => !importing && setImportOpen(false)}
        width={680}
        footer={
          <>
            <Button variant="ghost" disabled={importing} onClick={() => setImportOpen(false)}>
              取消
            </Button>
            <Button
              variant="primary"
              loading={importing}
              disabled={!importSelected.size}
              onClick={() => void submitImport()}
            >
              导入 {importSelected.size || ""} 张表
            </Button>
          </>
        }
      >
        <div className="import-search">
          <TextInput
            value={importQuery}
            onChange={(event) => setImportQuery(event.target.value)}
            placeholder="搜索表名或说明"
            icon="search"
            clearable
            onClear={() => setImportQuery("")}
          />
        </div>
        <div className="import-list">
          {filteredDatabaseTables.map((table) => {
            const key = table.name ?? "";
            return (
              <label
                className={`import-row ${importSelected.has(key) ? "is-on" : ""}`}
                key={`${table.schema}.${key}`}
              >
                <Checkbox
                  checked={importSelected.has(key)}
                  label=""
                  onChange={(checked) => {
                    const next = new Set(importSelected);
                    if (checked) next.add(key);
                    else next.delete(key);
                    setImportSelected(next);
                  }}
                />
                <div className="cell-main" style={{ flex: 1 }}>
                  <span className="cell-title">{table.name}</span>
                  <span className="cell-sub">{table.comment || "无表说明"}</span>
                </div>
                <Tag>{table.schema || "public"}</Tag>
              </label>
            );
          })}
          {!databaseLoading && !filteredDatabaseTables.length ? (
            <EmptyState title="没有匹配的数据库表" icon="database" />
          ) : null}
        </div>
      </Dialog>

      <Drawer
        open={Boolean(editor)}
        title={`生成配置 · ${editor?.tableName ?? ""}`}
        description={editor?.tableComment}
        onClose={() => setEditor(null)}
        width={820}
        footer={
          <>
            <Button variant="ghost" onClick={() => setEditor(null)}>
              关闭
            </Button>
            {editorTab === "metadata" && canManage ? (
              <Button variant="primary" loading={saving} onClick={() => void saveMetadata()}>
                保存配置
              </Button>
            ) : null}
            {editorTab === "preview" ? (
              <Button
                variant="primary"
                icon="download"
                loading={actionLoading}
                onClick={() => editor && void download([editor.id])}
              >
                下载 ZIP
              </Button>
            ) : null}
          </>
        }
      >
        {editor && editorForm ? (
          <div className="stack">
            <div className="drawer-toolbar">
              <Tabs
                tabs={[
                  { value: "metadata", label: "字段与元数据" },
                  { value: "preview", label: "代码预览" },
                ]}
                value={editorTab}
                onChange={(value) => setEditorTab(value as "metadata" | "preview")}
              />
              <div className="page-actions">
                <Button icon="refresh" loading={actionLoading} onClick={() => void sync()}>
                  同步结构
                </Button>
                <Button icon="codegen" loading={previewLoading} onClick={() => void loadPreview()}>
                  生成预览
                </Button>
              </div>
            </div>
            {editorTab === "metadata" ? (
              <>
                <div className="form-grid">
                  {METADATA_FIELDS.map(([key, label]) => (
                    <Field label={label} key={key}>
                      <TextInput
                        value={String(editorForm[key] ?? "")}
                        onChange={(event) =>
                          setEditorForm({ ...editorForm, [key]: event.target.value })
                        }
                      />
                    </Field>
                  ))}
                  <Field label="备注" wide>
                    <TextInput
                      multiline
                      rows={2}
                      value={editorForm.remark}
                      onChange={(event) =>
                        setEditorForm({ ...editorForm, remark: event.target.value })
                      }
                    />
                  </Field>
                </div>
                <Card>
                  <div className="panel-head">
                    <span className="section-label">字段映射</span>
                    <Tag tone="info">{editorForm.columns.length} 字段</Tag>
                  </div>
                  <div className="panel-body">
                    <div className="table-wrap is-scrollable">
                      <table className="table">
                        <thead>
                          <tr>
                            <th>字段</th>
                            <th>类型</th>
                            <th>可空</th>
                            <th>主键</th>
                            <th>说明</th>
                          </tr>
                        </thead>
                        <tbody>
                          {editorForm.columns.map((column) => (
                            <tr key={column.name}>
                              <td className="mono">{column.name}</td>
                              <td>
                                <span className="code-chip">{column.dataType}</span>
                              </td>
                              <td>{column.nullable ? "是" : "否"}</td>
                              <td>{column.primaryKey ? <Tag tone="ok">PK</Tag> : "—"}</td>
                              <td className="muted">{column.comment || "—"}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                </Card>
              </>
            ) : (
              <div className="preview">
                <div className="preview-files">
                  {(preview?.files ?? []).map((file) => (
                    <button
                      type="button"
                      className={`preview-file ${selectedPreviewFile?.path === file.path ? "is-active" : ""}`}
                      key={file.path}
                      onClick={() => setPreviewFile(file.path)}
                    >
                      <span className="preview-name">{file.name}</span>
                      <span className="preview-dir">{file.path}</span>
                    </button>
                  ))}
                  {!preview?.files.length ? (
                    <EmptyState title="尚未生成预览" icon="codegen" />
                  ) : null}
                </div>
                <div className="preview-code">
                  <div className="preview-head">
                    <span className="section-label">{selectedPreviewFile?.name ?? "预览"}</span>
                    <Button
                      variant="secondary"
                      icon="file"
                      onClick={() =>
                        selectedPreviewFile &&
                        void navigator.clipboard.writeText(selectedPreviewFile.code)
                      }
                    >
                      复制
                    </Button>
                  </div>
                  <pre className="code-block">
                    {selectedPreviewFile?.code ?? "选择“生成预览”读取真实 Go Codegen 输出。"}
                  </pre>
                  {preview?.warnings.length ? (
                    <div className="flag-row">
                      {preview.warnings.map((warning) => (
                        <Tag tone="danger" key={warning}>
                          {warning}
                        </Tag>
                      ))}
                    </div>
                  ) : null}
                </div>
              </div>
            )}
            <DetailGrid
              items={[
                { label: "表名", value: editorForm.tableName },
                { label: "Schema", value: editorForm.schema },
                { label: "更新时间", value: formatDateTime(editorForm.updatedAt) },
                { label: "导入 ID", value: editorForm.id, wide: true },
              ]}
            />
          </div>
        ) : null}
      </Drawer>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除代码生成配置"
        message={`确认删除「${deleteTarget?.tableName ?? ""}」的导入配置吗？数据库表本身不会被删除。`}
        confirmText="删除配置"
        danger
        loading={actionLoading}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
