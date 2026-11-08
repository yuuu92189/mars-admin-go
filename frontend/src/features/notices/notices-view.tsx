"use client";

import { useCallback, useMemo, useState } from "react";
import { Bell, CheckCheck, Megaphone, Plus, Send, Trash2 } from "lucide-react";
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
import { Select } from "@/components/ui/select";
import { Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import {
  createNotice,
  deleteNotice,
  getNotice,
  listMyNotices,
  listNotices,
  markNoticeRead,
  publishNotice,
  unreadNoticeCount,
  withdrawNotice,
} from "@/lib/api/notices";
import { listDepartments } from "@/lib/api/organization";
import { listUsers } from "@/lib/api/users";
import { hasPermission } from "@/lib/permissions";
import type { Department, Notice, NoticeInput, NoticePage, User } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 12;

type NoticeForm = {
  title: string;
  content: string;
  noticeType: number;
  channels: string[];
  targetType: "all" | "users" | "departments";
  targetIds: string[];
};

const EMPTY_FORM: NoticeForm = {
  title: "",
  content: "",
  noticeType: 1,
  channels: ["station"],
  targetType: "all",
  targetIds: [],
};

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function flattenDepartments(nodes: Department[]): Department[] {
  return nodes.flatMap((node) => [node, ...flattenDepartments(node.children ?? [])]);
}

export function NoticesView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "notice:manage");
  const [tab, setTab] = useState<"my" | "sent">("my");
  const [page, setPage] = useState<NoticePage>({
    items: [],
    total: 0,
    page: 1,
    pageSize: PAGE_SIZE,
  });
  const [pageNumber, setPageNumber] = useState(1);
  const [status, setStatus] = useState("");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [unread, setUnread] = useState(0);
  const [detail, setDetail] = useState<Notice | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<NoticeForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Notice | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const result =
        tab === "my"
          ? await listMyNotices({ page: pageNumber, pageSize: PAGE_SIZE })
          : await listNotices({ page: pageNumber, pageSize: PAGE_SIZE, status });
      setPage(result);
      const count = await unreadNoticeCount().catch(() => 0);
      setUnread(count);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [pageNumber, status, tab]);

  useDeferredEffect(load);

  const loadRecipients = useCallback(() => {
    if (!canManage) return;
    return Promise.all([listUsers({ page: 1, pageSize: 100 }), listDepartments()])
      .then(([userPage, departmentTree]) => {
        setUsers(userPage.items ?? []);
        setDepartments(departmentTree);
      })
      .catch(() => undefined);
  }, [canManage]);
  useDeferredEffect(loadRecipients);

  const rows = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return page.items ?? [];
    return (page.items ?? []).filter((notice) =>
      [notice.title, notice.content, notice.targetType]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [keyword, page.items]);
  const flatDepartments = useMemo(() => flattenDepartments(departments), [departments]);

  const openDetail = async (notice: Notice) => {
    setDetail(notice);
    if (!notice.readAt && tab === "my") {
      try {
        await markNoticeRead(notice.id);
        setUnread((value) => Math.max(0, value - 1));
        setDetail({ ...notice, readAt: new Date().toISOString() });
        setPage((current) => ({
          ...current,
          items: (current.items ?? []).map((item) =>
            item.id === notice.id ? { ...item, readAt: new Date().toISOString() } : item,
          ),
        }));
      } catch {
        // Detail remains readable even if read receipt fails.
      }
    }
    if (canManage) {
      setDetailLoading(true);
      getNotice(notice.id)
        .then(setDetail)
        .catch(() => undefined)
        .finally(() => setDetailLoading(false));
    }
  };

  const submit = async () => {
    if (!form.title.trim() || !form.content.trim()) {
      setFormError("标题和正文必填。");
      return;
    }
    if (form.targetType !== "all" && !form.targetIds.length) {
      setFormError("请选择通知接收对象。");
      return;
    }
    setSaving(true);
    setFormError(null);
    const input: NoticeInput = {
      title: form.title.trim(),
      content: form.content.trim(),
      noticeType: form.noticeType,
      channels: form.channels,
      targetType: form.targetType,
      targetIds: form.targetType === "all" ? [] : form.targetIds,
    };
    try {
      const notice = await createNotice(input);
      pushToast({ title: "通知草稿已创建", message: "发布后才会进入接收人通知列表", tone: "ok" });
      setCreateOpen(false);
      setForm(EMPTY_FORM);
      setTab("sent");
      setPageNumber(1);
      if (notice.id) await publishNotice(notice.id);
      await load();
    } catch (cause) {
      setFormError(apiMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const publish = async (notice: Notice) => {
    setActionLoading(true);
    try {
      await publishNotice(notice.id);
      pushToast({ title: "通知已发布", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "发布失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const withdraw = async (notice: Notice) => {
    setActionLoading(true);
    try {
      await withdrawNotice(notice.id);
      pushToast({ title: "通知已撤回", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "撤回失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const remove = async () => {
    if (!deleteTarget) return;
    setActionLoading(true);
    try {
      await deleteNotice(deleteTarget.id);
      pushToast({ title: "通知已删除", tone: "ok" });
      setDeleteTarget(null);
      if (detail?.id === deleteTarget.id) setDetail(null);
      await load();
    } catch (cause) {
      pushToast({ title: "删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const columns: TableColumn<Notice>[] = [
    {
      key: "title",
      header: "通知",
      render: (notice) => (
        <button
          type="button"
          className="cell-main link-btn"
          onClick={() => void openDetail(notice)}
        >
          <span className="row-ico m-groove">
            <Bell size={15} />
          </span>
          <span>
            <span className="cell-title ellipsis">{notice.title}</span>
            <span className="cell-sub ellipsis">{notice.content}</span>
          </span>
          {tab === "my" && !notice.readAt ? <span className="tag tone-info">未读</span> : null}
        </button>
      ),
    },
    {
      key: "type",
      header: "类型",
      render: (notice) => (
        <span>{notice.noticeType === 1 ? "通知" : notice.noticeType === 2 ? "公告" : "提醒"}</span>
      ),
    },
    {
      key: "target",
      header: "接收对象",
      render: (notice) => (
        <span className="muted">
          {notice.targetType === "all"
            ? "全部用户"
            : notice.targetType === "users"
              ? `${notice.targetIds.length} 位用户`
              : `${notice.targetIds.length} 个部门`}
        </span>
      ),
    },
    { key: "status", header: "状态", render: (notice) => <StatusTag status={notice.status} /> },
    {
      key: "created",
      header: "创建时间",
      render: (notice) => <span className="tnum faint">{formatDateTime(notice.createdAt)}</span>,
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (notice) => (
        <div className="row-actions">
          <IconButton label="查看通知" icon="file" small onClick={() => void openDetail(notice)} />
          {canManage && notice.status === "draft" ? (
            <IconButton label="发布通知" icon="send" small onClick={() => void publish(notice)} />
          ) : null}
          {canManage && notice.status === "published" ? (
            <IconButton label="撤回通知" icon="undo" small onClick={() => void withdraw(notice)} />
          ) : null}
          {canManage ? (
            <IconButton
              label="删除通知"
              icon="trash"
              small
              danger
              onClick={() => setDeleteTarget(notice)}
            />
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="通知中心"
      description="我的通知、发送记录、未读状态和公告发布均连接真实 Notification API。"
      actions={
        <>
          <Tabs
            tabs={[
              { value: "my", label: `我的通知${unread ? ` · ${unread}` : ""}` },
              { value: "sent", label: "发送记录" },
            ]}
            value={tab}
            onChange={(value) => {
              setTab(value as "my" | "sent");
              setPageNumber(1);
            }}
          />
          {canManage ? (
            <Button
              variant="primary"
              icon="plus"
              onClick={() => {
                setForm(EMPTY_FORM);
                setFormError(null);
                setCreateOpen(true);
              }}
            >
              发送通知
            </Button>
          ) : null}
        </>
      }
    >
      <div className="stack">
        <SearchForm
          loading={loading}
          onSearch={() => void load()}
          onReset={() => {
            setKeyword("");
            setStatus("");
            void load();
          }}
        >
          <Field label="关键词" wide>
            <TextInput
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="标题 / 正文 / 接收对象"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
          {tab === "sent" ? (
            <Field label="状态">
              <Select
                value={status}
                onChange={setStatus}
                options={[
                  { value: "", label: "全部状态" },
                  { value: "draft", label: "草稿" },
                  { value: "published", label: "已发布" },
                  { value: "withdrawn", label: "已撤回" },
                ]}
              />
            </Field>
          ) : null}
        </SearchForm>
        <Card className="reveal">
          <div className="panel-head">
            <div>
              <span className="section-label">{tab === "my" ? "我的通知" : "发送记录"}</span>
              <span className="panel-meta">{page.total ?? 0} 条</span>
            </div>
            {unread ? <Tag tone="info">{unread} 条未读</Tag> : null}
          </div>
          <DataState
            loading={loading}
            error={error}
            permissionDenied={permissionDenied}
            empty={!loading && !error && !rows.length}
            emptyTitle={tab === "my" ? "暂时没有通知" : "还没有发送记录"}
            onRetry={() => void load()}
          >
            <DataTable columns={columns} rows={rows} rowKey={(notice) => notice.id} />
            <Pagination
              page={pageNumber}
              pageSize={PAGE_SIZE}
              total={page.total ?? 0}
              onChange={setPageNumber}
            />
          </DataState>
        </Card>
      </div>

      <Drawer
        open={Boolean(detail)}
        title={detail?.title ?? "通知详情"}
        description={
          detail ? `${detail.targetType} · ${formatDateTime(detail.createdAt)}` : undefined
        }
        onClose={() => setDetail(null)}
        width={600}
      >
        {detail ? (
          <div className="stack">
            <div className="drawer-toolbar">
              <StatusTag status={detail.status} />
              <Tag tone="info">{detail.channels.join(" / ") || "站内信"}</Tag>
            </div>
            <Card>
              <div className="panel-body" style={{ paddingTop: 20, lineHeight: 1.8 }}>
                {detailLoading ? "正在读取通知详情…" : detail.content}
              </div>
            </Card>
            <DetailGrid
              items={[
                { label: "通知 ID", value: detail.id, wide: true },
                { label: "创建时间", value: formatDateTime(detail.createdAt) },
                { label: "发布时间", value: formatDateTime(detail.publishedAt) },
                { label: "阅读时间", value: formatDateTime(detail.readAt) },
                {
                  label: "接收对象",
                  value:
                    detail.targetType === "all"
                      ? "全部用户"
                      : detail.targetType === "users"
                        ? detail.targetIds
                            .map((id) => users.find((user) => user.id === id)?.displayName ?? id)
                            .join("、")
                        : detail.targetIds
                            .map(
                              (id) =>
                                flatDepartments.find((department) => department.id === id)?.name ??
                                id,
                            )
                            .join("、"),
                  wide: true,
                },
              ]}
            />
            {canManage ? (
              <div className="drawer-toolbar">
                {detail.status === "draft" ? (
                  <Button icon="send" loading={actionLoading} onClick={() => void publish(detail)}>
                    发布
                  </Button>
                ) : null}
                {detail.status === "published" ? (
                  <Button icon="undo" loading={actionLoading} onClick={() => void withdraw(detail)}>
                    撤回
                  </Button>
                ) : null}
                <Button variant="danger" icon="trash" onClick={() => setDeleteTarget(detail)}>
                  删除
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}
      </Drawer>

      <Dialog
        open={createOpen}
        title="发送通知"
        description="创建草稿后立即调用真实发布接口；失败不会显示成功。"
        onClose={() => !saving && setCreateOpen(false)}
        width={680}
        footer={
          <>
            <Button variant="ghost" disabled={saving} onClick={() => setCreateOpen(false)}>
              取消
            </Button>
            <Button variant="primary" icon="send" loading={saving} onClick={() => void submit()}>
              发布通知
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label="通知标题" required wide>
            <TextInput
              value={form.title}
              onChange={(event) => setForm({ ...form, title: event.target.value })}
            />
          </Field>
          <Field label="通知类型">
            <Select
              value={String(form.noticeType)}
              onChange={(value) => setForm({ ...form, noticeType: Number(value) })}
              options={[
                { value: "1", label: "通知" },
                { value: "2", label: "公告" },
                { value: "3", label: "提醒" },
              ]}
            />
          </Field>
          <Field label="接收对象">
            <Select
              value={form.targetType}
              onChange={(value) =>
                setForm({
                  ...form,
                  targetType: value as NoticeForm["targetType"],
                  targetIds: [],
                })
              }
              options={[
                { value: "all", label: "全部用户" },
                { value: "users", label: "指定用户" },
                { value: "departments", label: "指定部门" },
              ]}
            />
          </Field>
          {form.targetType === "users" ? (
            <Field label="选择用户" wide>
              <div className="stack" style={{ maxHeight: 180, overflowY: "auto" }}>
                {users.map((user) => (
                  <Checkbox
                    key={user.id}
                    checked={form.targetIds.includes(user.id)}
                    label={`${user.displayName} · @${user.username}`}
                    onChange={(checked) =>
                      setForm({
                        ...form,
                        targetIds: checked
                          ? [...form.targetIds, user.id]
                          : form.targetIds.filter((id) => id !== user.id),
                      })
                    }
                  />
                ))}
              </div>
            </Field>
          ) : null}
          {form.targetType === "departments" ? (
            <Field label="选择部门" wide>
              <div className="stack" style={{ maxHeight: 180, overflowY: "auto" }}>
                {flatDepartments.map((department) => (
                  <Checkbox
                    key={department.id}
                    checked={form.targetIds.includes(department.id ?? "")}
                    label={`${department.name} · ${department.code}`}
                    onChange={(checked) =>
                      setForm({
                        ...form,
                        targetIds: checked
                          ? [...form.targetIds, department.id ?? ""]
                          : form.targetIds.filter((id) => id !== department.id),
                      })
                    }
                  />
                ))}
              </div>
            </Field>
          ) : null}
          <Field label="通知正文" required wide>
            <TextInput
              multiline
              rows={8}
              value={form.content}
              onChange={(event) => setForm({ ...form, content: event.target.value })}
            />
          </Field>
        </div>
        {formError ? (
          <p className="field-error" style={{ marginTop: 14 }}>
            {formError}
          </p>
        ) : null}
      </Dialog>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除通知"
        message={`确认删除通知「${deleteTarget?.title ?? ""}」吗？`}
        confirmText="删除"
        danger
        loading={actionLoading}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
