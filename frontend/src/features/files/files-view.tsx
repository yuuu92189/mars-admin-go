"use client";

import { useCallback, useRef, useState, type ChangeEvent } from "react";
import { Download, FileUp } from "lucide-react";
import {
  BusinessPage,
  DataState,
  SearchForm,
  StatusTag,
  formatDateTime,
} from "@/components/business/business-page";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DataTable, Pagination, type TableColumn } from "@/components/ui/data-table";
import { Field, TextInput } from "@/components/ui/input";
import { Card, Tag } from "@/components/ui/surface";
import { ApiError } from "@/lib/api/client";
import { deleteFile, fileDownloadUrl, listFiles, uploadFile } from "@/lib/api/storage";
import type { StoredFile } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function errorMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

export function FilesView() {
  const { pushToast } = useToast();
  const fileInput = useRef<HTMLInputElement>(null);
  const [files, setFiles] = useState<StoredFile[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [keyword, setKeyword] = useState("");
  const [appliedKeyword, setAppliedKeyword] = useState("");
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<StoredFile | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const result = await listFiles({
        page,
        pageSize,
        keyword: appliedKeyword || undefined,
      });
      setFiles(result.items ?? []);
      setTotal(result.total ?? 0);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(errorMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [appliedKeyword, page, pageSize]);

  useDeferredEffect(load);

  function search() {
    setPage(1);
    setAppliedKeyword(keyword.trim());
  }

  function reset() {
    setKeyword("");
    setAppliedKeyword("");
    setPage(1);
  }

  async function onUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    setUploading(true);
    try {
      await uploadFile(file);
      pushToast({ title: "文件已上传", message: file.name, tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "上传失败", message: errorMessage(cause), tone: "danger" });
    } finally {
      setUploading(false);
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteFile(deleteTarget.id);
      setDeleteTarget(null);
      pushToast({ title: "文件已删除", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "删除失败", message: errorMessage(cause), tone: "danger" });
    } finally {
      setDeleting(false);
    }
  }

  const columns: TableColumn<StoredFile>[] = [
    {
      key: "name",
      header: "文件名",
      render: (file) => (
        <div>
          <strong>{file.originalName}</strong>
          <div className="faint">{file.contentType || "未知类型"}</div>
        </div>
      ),
    },
    { key: "size", header: "大小", render: (file) => formatBytes(file.sizeBytes) },
    { key: "provider", header: "存储", render: (file) => file.storageProvider },
    {
      key: "status",
      header: "状态",
      render: (file) => <StatusTag status={file.status} />,
    },
    {
      key: "created",
      header: "上传时间",
      render: (file) => formatDateTime(file.createdAt),
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (file) => (
        <div className="table-actions">
          <a
            className="btn btn-ghost btn-sm"
            href={fileDownloadUrl(file.id)}
            aria-label={`下载 ${file.originalName}`}
          >
            <Download size={14} /> 下载
          </a>
          <Button
            variant="text"
            size="sm"
            className="btn-danger-text"
            icon="trash"
            onClick={() => setDeleteTarget(file)}
          >
            删除
          </Button>
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="文件管理"
      description="文件由 Go 存储服务接收、校验并持久化；浏览器只访问同源受保护下载地址。"
      actions={
        <>
          <Tag tone="info">
            <FileUp size={13} /> 私有文件
          </Tag>
          <Button
            variant="primary"
            icon="upload"
            loading={uploading}
            onClick={() => fileInput.current?.click()}
          >
            上传文件
          </Button>
          <input ref={fileInput} type="file" hidden onChange={(event) => void onUpload(event)} />
        </>
      }
    >
      <SearchForm loading={loading} onSearch={search} onReset={reset}>
        <Field label="文件名" wide>
          <TextInput
            icon="search"
            value={keyword}
            clearable
            placeholder="搜索文件名"
            onChange={(event) => setKeyword(event.target.value)}
            onClear={() => setKeyword("")}
          />
        </Field>
      </SearchForm>
      <Card className="list-page-panel reveal">
        <DataState
          loading={loading}
          error={error}
          permissionDenied={permissionDenied}
          empty={!loading && !error && files.length === 0}
          emptyTitle="还没有文件"
          emptyDescription="上传第一个文件后，文件元数据和下载入口会出现在这里。"
          onRetry={() => void load()}
        >
          <DataTable columns={columns} rows={files} rowKey={(file) => file.id} loading={loading} />
          <Pagination page={page} pageSize={pageSize} total={total} onChange={setPage} />
        </DataState>
      </Card>
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除文件"
        message={deleteTarget ? `确认删除“${deleteTarget.originalName}”吗？` : ""}
        confirmText="删除"
        danger
        loading={deleting}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
      />
    </BusinessPage>
  );
}
