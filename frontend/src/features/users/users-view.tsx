"use client";

import { useCallback, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound, Pencil, Plus, Shield, Trash2, UserRoundCheck, UsersRound } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import {
  BulkActionBar,
  BusinessPage,
  DataState,
  DetailGrid,
  SearchForm,
  StatusTag,
  TreeNav,
  formatDateTime,
} from "@/components/business/business-page";
import { Button, IconButton } from "@/components/ui/button";
import { DataTable, Pagination, Tabs, type TableColumn } from "@/components/ui/data-table";
import { Checkbox, Field, Switch, TextInput } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Dialog, Drawer } from "@/components/ui/overlay";
import { Select } from "@/components/ui/select";
import { Avatar, Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import {
  createUser,
  deleteUser,
  listUsers,
  resetUserPassword,
  revokeUserSessions,
  updateUser,
} from "@/lib/api/users";
import { listRoles } from "@/lib/api/access";
import { listDepartments, listPositions } from "@/lib/api/organization";
import { hasPermission } from "@/lib/permissions";
import type { Department, Position, Role, UpdateUser, User, UserPage } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 20;

type UserForm = {
  username: string;
  password: string;
  displayName: string;
  email: string;
  phone: string;
  userType: string;
  status: string;
  gender: number;
  roleIds: string[];
  departmentIds: string[];
  positionIds: string[];
};

const EMPTY_FORM: UserForm = {
  username: "",
  password: "",
  displayName: "",
  email: "",
  phone: "",
  userType: "admin",
  status: "active",
  gender: 0,
  roleIds: [],
  departmentIds: [],
  positionIds: [],
};

function flattenDepartments(nodes: Department[]): Department[] {
  return nodes.flatMap((node) => [node, ...flattenDepartments(node.children ?? [])]);
}

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function statusLabel(status: string): string {
  switch (status) {
    case "active":
      return "正常";
    case "pending":
      return "待审核";
    case "suspended":
      return "停用";
    case "rejected":
      return "已拒绝";
    case "left":
      return "已离职";
    default:
      return status || "未知";
  }
}

function MultiPicker({
  items,
  selected,
  onChange,
  emptyText,
}: {
  items: Array<{ id: string; label: string; description?: string }>;
  selected: string[];
  onChange: (ids: string[]) => void;
  emptyText: string;
}) {
  if (!items.length) return <span className="faint">{emptyText}</span>;
  return (
    <div className="stack" style={{ gap: 8, maxHeight: 170, overflowY: "auto" }}>
      {items.map((item) => (
        <Checkbox
          key={item.id}
          checked={selected.includes(item.id)}
          label={item.description ? `${item.label} · ${item.description}` : item.label}
          onChange={(checked) =>
            onChange(checked ? [...selected, item.id] : selected.filter((id) => id !== item.id))
          }
        />
      ))}
    </div>
  );
}

export function UsersView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const router = useRouter();
  const canCreate = hasPermission(currentUser, "user:create");
  const canUpdate = hasPermission(currentUser, "user:update");
  const canDelete = hasPermission(currentUser, "user:delete");

  const [page, setPage] = useState<UserPage>({ items: [], total: 0, page: 1, pageSize: PAGE_SIZE });
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [departmentId, setDepartmentId] = useState<string | null>(null);
  const [pageNumber, setPageNumber] = useState(1);
  const [view, setView] = useState<"table" | "board">("table");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [roles, setRoles] = useState<Role[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState<UserForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [detailUser, setDetailUser] = useState<User | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [resetTarget, setResetTarget] = useState<User | null>(null);
  const [temporaryPassword, setTemporaryPassword] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const result = await listUsers({
        page: pageNumber,
        pageSize: PAGE_SIZE,
        keyword,
        status,
        departmentId,
      });
      setPage(result);
      setSelected(new Set());
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [departmentId, keyword, pageNumber, status]);

  const loadOptions = useCallback(async () => {
    try {
      const [rolePage, departmentTree, positionList] = await Promise.all([
        listRoles({ page: 1, pageSize: 100 }),
        listDepartments(),
        listPositions(),
      ]);
      setRoles(rolePage.items);
      setDepartments(departmentTree);
      setPositions(positionList);
    } catch (cause) {
      pushToast({ title: "选项加载失败", message: apiMessage(cause), tone: "danger" });
    }
  }, [pushToast]);

  useDeferredEffect(load);
  useDeferredEffect(loadOptions);

  const flatDepartments = useMemo(() => flattenDepartments(departments), [departments]);
  const roleMap = useMemo(() => new Map(roles.map((role) => [role.id, role])), [roles]);
  const departmentMap = useMemo(
    () => new Map(flatDepartments.map((department) => [department.id, department])),
    [flatDepartments],
  );
  const positionMap = useMemo(
    () => new Map(positions.map((position) => [position.id, position])),
    [positions],
  );
  const rows = useMemo(() => page.items ?? [], [page.items]);

  const openCreate = () => {
    setEditing(null);
    setForm({
      ...EMPTY_FORM,
      departmentIds: departmentId ? [departmentId] : [],
      roleIds: roles.length === 1 && roles[0]?.id ? [roles[0].id] : [],
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (row: User) => {
    setEditing(row);
    setForm({
      username: row.username,
      password: "",
      displayName: row.displayName,
      email: row.email ?? "",
      phone: row.phone ?? "",
      userType: row.userType,
      status: row.status,
      gender: row.gender ?? 0,
      roleIds: row.roleIds ?? [],
      departmentIds: row.departmentIds ?? [],
      positionIds: row.positionIds ?? [],
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const submitForm = async () => {
    if (!form.username.trim() || !form.displayName.trim()) {
      setFormError("登录账号和用户姓名必填。");
      return;
    }
    if (!editing && form.password.length < 8) {
      setFormError("初始密码至少 8 位。");
      return;
    }
    if (!form.departmentIds.length) {
      setFormError("请至少选择一个所属部门。");
      return;
    }
    setSaving(true);
    setFormError(null);
    try {
      if (editing) {
        const input: UpdateUser = {
          displayName: form.displayName.trim(),
          email: form.email.trim() || undefined,
          phone: form.phone.trim() || undefined,
          userType: form.userType,
          status: form.status,
          gender: form.gender,
          version: editing.version ?? 0,
          roleIds: form.roleIds,
          departmentIds: form.departmentIds,
          primaryDepartmentId: form.departmentIds[0],
          positionIds: form.positionIds,
        };
        await updateUser(editing.id, input);
        pushToast({ title: "用户信息已更新", tone: "ok" });
      } else {
        await createUser({
          username: form.username.trim(),
          password: form.password,
          displayName: form.displayName.trim(),
          email: form.email.trim() || undefined,
          phone: form.phone.trim() || undefined,
          userType: form.userType,
          status: form.status,
          gender: form.gender,
          roleIds: form.roleIds,
          departmentIds: form.departmentIds,
          primaryDepartmentId: form.departmentIds[0],
          positionIds: form.positionIds,
        });
        pushToast({ title: "用户已创建", tone: "ok" });
      }
      setDialogOpen(false);
      await load();
    } catch (cause) {
      setFormError(apiMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setActionLoading(true);
    try {
      await deleteUser(deleteTarget.id);
      pushToast({ title: `已删除「${deleteTarget.displayName}」`, tone: "ok" });
      setDeleteTarget(null);
      await load();
    } catch (cause) {
      pushToast({ title: "删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const batchStatus = async (nextStatus: string) => {
    const targets = rows.filter((row) => selected.has(row.id));
    if (!targets.length) return;
    setActionLoading(true);
    try {
      await Promise.all(
        targets.map((row) =>
          updateUser(row.id, {
            displayName: row.displayName,
            email: row.email,
            phone: row.phone,
            avatarUrl: row.avatarUrl,
            userType: row.userType,
            status: nextStatus,
            gender: row.gender ?? 0,
            version: row.version ?? 0,
            roleIds: row.roleIds ?? [],
            departmentIds: row.departmentIds ?? [],
            positionIds: row.positionIds ?? [],
          }),
        ),
      );
      pushToast({ title: `已更新 ${targets.length} 个用户状态`, tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "批量更新失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const batchDelete = async () => {
    const targets = rows.filter((row) => selected.has(row.id));
    setActionLoading(true);
    try {
      await Promise.all(targets.map((row) => deleteUser(row.id)));
      pushToast({ title: `已删除 ${targets.length} 个用户`, tone: "ok" });
      setBatchDeleteOpen(false);
      await load();
    } catch (cause) {
      pushToast({ title: "批量删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const resetPassword = async () => {
    if (!resetTarget) return;
    setActionLoading(true);
    try {
      const result = await resetUserPassword(resetTarget.id);
      setTemporaryPassword(result.temporaryPassword);
      pushToast({ title: "密码已重置", message: "临时密码只展示一次", tone: "ok" });
    } catch (cause) {
      pushToast({ title: "密码重置失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const columns: TableColumn<User>[] = [
    {
      key: "user",
      header: "用户",
      render: (row) => (
        <button type="button" className="cell-main link-btn" onClick={() => setDetailUser(row)}>
          <Avatar name={row.displayName} size={32} />
          <span>
            <span className="cell-title ellipsis">{row.displayName}</span>
            <span className="cell-sub">@{row.username}</span>
          </span>
        </button>
      ),
    },
    {
      key: "department",
      header: "部门",
      render: (row) =>
        row.departmentIds?.length ? (
          <span>{departmentMap.get(row.departmentIds[0] ?? "")?.name ?? row.departmentIds[0]}</span>
        ) : (
          <span className="faint">未分配</span>
        ),
    },
    {
      key: "role",
      header: "角色",
      render: (row) =>
        row.roleIds?.length ? (
          <div className="flag-row">
            {row.roleIds.slice(0, 2).map((id) => (
              <Tag tone="info" key={id}>
                {roleMap.get(id)?.name ?? id.slice(0, 8)}
              </Tag>
            ))}
            {row.roleIds.length > 2 ? <Tag>+{row.roleIds.length - 2}</Tag> : null}
          </div>
        ) : (
          <span className="faint">未分配</span>
        ),
    },
    {
      key: "phone",
      header: "手机号",
      render: (row) => <span className="tnum muted">{row.phone || "—"}</span>,
    },
    {
      key: "status",
      header: "状态",
      render: (row) => <StatusTag status={row.status} label={statusLabel(row.status)} />,
    },
    {
      key: "lastLogin",
      header: "最近登录",
      render: (row) => <span className="tnum faint">{formatDateTime(row.lastLoginAt)}</span>,
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (row) => (
        <div className="row-actions">
          {canUpdate ? (
            <IconButton label="编辑用户" icon="edit" small onClick={() => openEdit(row)} />
          ) : null}
          {canUpdate ? (
            <IconButton
              label="重置密码"
              icon="key"
              small
              onClick={() => {
                setResetTarget(row);
                setTemporaryPassword(null);
              }}
            />
          ) : null}
          {canUpdate ? (
            <IconButton
              label="撤销会话"
              icon="shield"
              small
              onClick={async () => {
                try {
                  await revokeUserSessions(row.id);
                  pushToast({ title: "该用户会话已撤销", tone: "ok" });
                } catch (cause) {
                  pushToast({
                    title: "撤销会话失败",
                    message: apiMessage(cause),
                    tone: "danger",
                  });
                }
              }}
            />
          ) : null}
          {canDelete ? (
            <IconButton
              label="删除用户"
              icon="trash"
              small
              danger
              onClick={() => setDeleteTarget(row)}
            />
          ) : null}
        </div>
      ),
    },
  ];

  const boardGroups = useMemo(() => {
    const groups = new Map<string, User[]>();
    for (const row of rows) {
      const label =
        row.departmentIds?.[0] && departmentMap.get(row.departmentIds[0])
          ? (departmentMap.get(row.departmentIds[0])?.name ?? "未分配部门")
          : "未分配部门";
      groups.set(label, [...(groups.get(label) ?? []), row]);
    }
    return [...groups.entries()].sort((a, b) => b[1].length - a[1].length);
  }, [departmentMap, rows]);

  return (
    <BusinessPage
      title="用户管理"
      description="维护后台账号、组织归属、岗位与角色；权限由 Go API 实时校验。"
      fitViewport
      actions={
        <>
          <Tabs
            tabs={[
              { value: "table", label: "列表" },
              { value: "board", label: "看板" },
            ]}
            value={view}
            onChange={(value) => setView(value as "table" | "board")}
          />
          {canCreate ? (
            <Button variant="primary" icon="plus" onClick={openCreate}>
              新增用户
            </Button>
          ) : null}
        </>
      }
    >
      <div className="list-page-body split">
        <aside className="split-side m-float reveal">
          <TreeNav
            items={departments}
            getId={(item) => item.id ?? ""}
            getLabel={(item) => item.name ?? "未命名部门"}
            getChildren={(item) => item.children}
            selectedId={departmentId}
            onSelect={(id) => {
              setDepartmentId(id);
              setPageNumber(1);
            }}
            rootLabel="全部用户"
            rootCount={page.total}
          />
        </aside>
        <div className="list-page-stack stack">
          <SearchForm
            loading={loading}
            onSearch={() => {
              setPageNumber(1);
              void load();
            }}
            onReset={() => {
              setKeyword("");
              setStatus("");
              setDepartmentId(null);
              setPageNumber(1);
            }}
          >
            <Field label="关键词" wide>
              <TextInput
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder="账号 / 姓名 / 手机号"
                clearable
                onClear={() => setKeyword("")}
                icon="search"
              />
            </Field>
            <Field label="状态">
              <Select
                value={status}
                onChange={setStatus}
                options={[
                  { value: "", label: "全部状态" },
                  { value: "active", label: "正常" },
                  { value: "suspended", label: "停用" },
                  { value: "pending", label: "待审核" },
                  { value: "rejected", label: "已拒绝" },
                  { value: "left", label: "已离职" },
                ]}
              />
            </Field>
          </SearchForm>

          {view === "table" ? (
            <Card className="list-page-panel reveal">
              <div className="panel-head">
                <div>
                  <span className="section-label">用户列表</span>
                  <span className="panel-meta">共 {page.total} 个账号</span>
                </div>
              </div>
              <DataState
                loading={loading}
                error={error}
                permissionDenied={permissionDenied}
                empty={!loading && !error && rows.length === 0}
                emptyTitle="没有匹配的用户"
                onRetry={() => void load()}
              >
                <DataTable
                  columns={columns}
                  rows={rows}
                  rowKey={(row) => row.id}
                  selectable={canDelete || canUpdate}
                  selected={selected}
                  onSelectionChange={setSelected}
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
            <div className="alt-view reveal">
              <div className="board-toolbar">
                <span className="eyebrow">按部门分列</span>
                <span className="faint">
                  当前页 {rows.length} / {page.total}
                </span>
              </div>
              <div className="board">
                {boardGroups.map(([title, items], index) => (
                  <section className={`board-col cat-${(index % 5) + 1}`} key={title}>
                    <div className="board-col-head">
                      <span className="board-dot" />
                      <span className="board-col-title">{title}</span>
                      <span className="board-col-count">{items.length}</span>
                    </div>
                    <div className="board-col-body">
                      {items.map((item) => (
                        <button
                          type="button"
                          className="board-card"
                          key={item.id}
                          onClick={() => setDetailUser(item)}
                        >
                          <div className="ucard-head">
                            <Avatar name={item.displayName} size={32} />
                            <span style={{ minWidth: 0 }}>
                              <span className="ucard-name ellipsis">{item.displayName}</span>
                              <span className="ucard-sub ellipsis">@{item.username}</span>
                            </span>
                          </div>
                          <div className="ucard-tags">
                            {(item.roleIds ?? []).slice(0, 2).map((id) => (
                              <Tag tone="info" key={id}>
                                {roleMap.get(id)?.name ?? id.slice(0, 8)}
                              </Tag>
                            ))}
                          </div>
                          <div className="ucard-foot">
                            <span className="ucard-time">{formatDateTime(item.lastLoginAt)}</span>
                            <span
                              className={`ucard-state ${item.status !== "active" ? "is-off" : ""}`}
                            >
                              {statusLabel(item.status)}
                            </span>
                          </div>
                        </button>
                      ))}
                    </div>
                  </section>
                ))}
                {!boardGroups.length && !loading ? (
                  <EmptyState title="暂无用户数据" icon="users" />
                ) : null}
              </div>
            </div>
          )}
        </div>
      </div>

      <BulkActionBar count={selected.size} onClear={() => setSelected(new Set())}>
        {canUpdate ? (
          <>
            <Button
              variant="text"
              size="sm"
              disabled={actionLoading}
              onClick={() => void batchStatus("active")}
            >
              启用
            </Button>
            <Button
              variant="text"
              size="sm"
              disabled={actionLoading}
              onClick={() => void batchStatus("suspended")}
            >
              停用
            </Button>
          </>
        ) : null}
        {canDelete ? (
          <Button
            variant="text"
            size="sm"
            className="is-danger"
            disabled={actionLoading}
            onClick={() => setBatchDeleteOpen(true)}
          >
            删除
          </Button>
        ) : null}
      </BulkActionBar>

      <Dialog
        open={dialogOpen}
        title={editing ? "编辑用户" : "新增用户"}
        description={editing ? `@${editing.username}` : "创建真实后台账号并分配组织关系"}
        onClose={() => !saving && setDialogOpen(false)}
        width={680}
        footer={
          <>
            <Button variant="ghost" disabled={saving} onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button variant="primary" loading={saving} onClick={() => void submitForm()}>
              {editing ? "保存修改" : "创建用户"}
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label="登录账号" required>
            <TextInput
              value={form.username}
              disabled={Boolean(editing)}
              onChange={(event) => setForm({ ...form, username: event.target.value })}
              placeholder="4-50 位"
            />
          </Field>
          <Field label="用户姓名" required>
            <TextInput
              value={form.displayName}
              onChange={(event) => setForm({ ...form, displayName: event.target.value })}
              placeholder="真实姓名"
            />
          </Field>
          {!editing ? (
            <Field label="初始密码" required hint="至少 8 位，保存后不会回显">
              <TextInput
                type="password"
                autoComplete="new-password"
                value={form.password}
                onChange={(event) => setForm({ ...form, password: event.target.value })}
              />
            </Field>
          ) : null}
          <Field label="用户类型">
            <Select
              value={form.userType}
              onChange={(value) => setForm({ ...form, userType: value })}
              options={[
                { value: "admin", label: "管理端" },
                { value: "pc", label: "桌面端" },
                { value: "app", label: "移动端" },
              ]}
            />
          </Field>
          <Field label="手机号">
            <TextInput
              type="tel"
              value={form.phone}
              onChange={(event) => setForm({ ...form, phone: event.target.value })}
              placeholder="选填"
            />
          </Field>
          <Field label="邮箱">
            <TextInput
              type="email"
              value={form.email}
              onChange={(event) => setForm({ ...form, email: event.target.value })}
              placeholder="选填"
            />
          </Field>
          <Field label="所属部门" required wide>
            <MultiPicker
              items={flatDepartments.map((item) => ({
                id: item.id ?? "",
                label: item.name ?? "未命名部门",
                description: item.code,
              }))}
              selected={form.departmentIds}
              onChange={(ids) => setForm({ ...form, departmentIds: ids })}
              emptyText="暂无可选部门"
            />
          </Field>
          <Field label="角色">
            <MultiPicker
              items={roles.map((role) => ({
                id: role.id ?? "",
                label: role.name ?? "未命名角色",
                description: role.code,
              }))}
              selected={form.roleIds}
              onChange={(ids) => setForm({ ...form, roleIds: ids })}
              emptyText="暂无可选角色"
            />
          </Field>
          <Field label="岗位">
            <MultiPicker
              items={positions.map((position) => ({
                id: position.id ?? "",
                label: position.name ?? "未命名岗位",
                description: position.code,
              }))}
              selected={form.positionIds}
              onChange={(ids) => setForm({ ...form, positionIds: ids })}
              emptyText="暂无可选岗位"
            />
          </Field>
          <Field label="账号状态" wide>
            <Switch
              checked={form.status === "active"}
              label={form.status === "active" ? "正常，允许登录" : "停用，禁止登录"}
              onChange={(checked) => setForm({ ...form, status: checked ? "active" : "suspended" })}
            />
          </Field>
        </div>
        {formError ? (
          <p className="field-error" style={{ marginTop: 14 }}>
            {formError}
          </p>
        ) : null}
      </Dialog>

      <Drawer
        open={Boolean(detailUser)}
        title={detailUser?.displayName ?? "用户详情"}
        description={detailUser ? `@${detailUser.username}` : undefined}
        onClose={() => setDetailUser(null)}
        width={520}
        footer={
          <>
            <Button
              variant="ghost"
              onClick={() => {
                if (!detailUser) return;
                router.push(`/system/role?trace=user:${detailUser.id}`);
              }}
            >
              <Shield size={15} /> 权限溯源
            </Button>
            {detailUser && canUpdate ? (
              <Button variant="primary" icon="edit" onClick={() => openEdit(detailUser)}>
                编辑
              </Button>
            ) : null}
          </>
        }
      >
        {detailUser ? (
          <div className="stack">
            <div className="drawer-toolbar">
              <div className="cell-main">
                <Avatar name={detailUser.displayName} size={48} />
                <div>
                  <div className="cell-title">{detailUser.displayName}</div>
                  <div className="cell-sub">@{detailUser.username}</div>
                </div>
              </div>
              <StatusTag status={detailUser.status} label={statusLabel(detailUser.status)} />
            </div>
            <DetailGrid
              items={[
                { label: "用户 ID", value: detailUser.id, wide: true },
                { label: "邮箱", value: detailUser.email || "—" },
                { label: "手机号", value: detailUser.phone || "—" },
                { label: "用户类型", value: detailUser.userType },
                { label: "最近登录", value: formatDateTime(detailUser.lastLoginAt) },
                { label: "创建时间", value: formatDateTime(detailUser.createdAt) },
                { label: "更新时间", value: formatDateTime(detailUser.updatedAt) },
                { label: "MFA", value: detailUser.mfaEnabled ? "已启用" : "未启用" },
                { label: "版本", value: detailUser.version ?? 0 },
              ]}
            />
            <Card>
              <div className="panel-head">
                <span className="section-label">组织与授权</span>
              </div>
              <div className="panel-body stack">
                <DetailGrid
                  items={[
                    {
                      label: "部门",
                      value:
                        (detailUser.departmentIds ?? [])
                          .map((id) => departmentMap.get(id)?.name ?? id)
                          .join("、") || "未分配",
                      wide: true,
                    },
                    {
                      label: "角色",
                      value:
                        (detailUser.roleIds ?? [])
                          .map((id) => roleMap.get(id)?.name ?? id)
                          .join("、") || "未分配",
                      wide: true,
                    },
                    {
                      label: "岗位",
                      value:
                        (detailUser.positionIds ?? [])
                          .map((id) => positionMap.get(id)?.name ?? id)
                          .join("、") || "未分配",
                      wide: true,
                    },
                  ]}
                />
              </div>
            </Card>
          </div>
        ) : null}
      </Drawer>

      <Dialog
        open={Boolean(resetTarget)}
        title="重置用户密码"
        description={
          resetTarget ? `${resetTarget.displayName} · @${resetTarget.username}` : undefined
        }
        onClose={() => {
          setResetTarget(null);
          setTemporaryPassword(null);
        }}
        width={470}
        footer={
          <>
            <Button
              variant="ghost"
              onClick={() => {
                setResetTarget(null);
                setTemporaryPassword(null);
              }}
            >
              关闭
            </Button>
            {!temporaryPassword ? (
              <Button variant="danger" loading={actionLoading} onClick={() => void resetPassword()}>
                确认重置
              </Button>
            ) : null}
          </>
        }
      >
        {temporaryPassword ? (
          <div className="stack">
            <p className="muted">临时密码只在此处展示一次，请通过安全渠道交付给用户。</p>
            <div className="code-block" aria-label="临时密码">
              {temporaryPassword}
            </div>
            <Button
              variant="secondary"
              icon="file"
              onClick={() => void navigator.clipboard.writeText(temporaryPassword)}
            >
              复制临时密码
            </Button>
          </div>
        ) : (
          <EmptyState
            title="确认重置该用户密码？"
            description="后端会生成新的临时密码，并立即使该用户现有会话失效。"
            icon="key"
          />
        )}
      </Dialog>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除用户"
        message={`确认删除「${deleteTarget?.displayName ?? ""}」吗？删除后该账号将无法登录。`}
        confirmText="删除"
        danger
        loading={actionLoading}
        onConfirm={() => void confirmDelete()}
        onClose={() => setDeleteTarget(null)}
      />
      <ConfirmDialog
        open={batchDeleteOpen}
        title="批量删除用户"
        message={`确认删除选中的 ${selected.size} 个用户吗？此操作不可撤销。`}
        confirmText="批量删除"
        danger
        loading={actionLoading}
        onConfirm={() => void batchDelete()}
        onClose={() => setBatchDeleteOpen(false)}
      />
    </BusinessPage>
  );
}
