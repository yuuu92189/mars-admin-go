"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { Check, Plus, ShieldCheck, Trash2, UserRoundSearch } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import {
  BusinessPage,
  DataState,
  DetailGrid,
  SearchForm,
  StatusTag,
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
  createRole,
  deleteRole,
  getRoleMenus,
  listMenus,
  listPermissions,
  listRoles,
  setRoleMenus,
  setRolePermissions,
  updateRole,
} from "@/lib/api/access";
import { getUser } from "@/lib/api/users";
import { hasPermission } from "@/lib/permissions";
import type { Menu, Permission, Role, RolePage, UpdateRole, User } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 20;
const SCOPES = [
  { value: "all", label: "全部数据权限", desc: "可查看系统内所有部门的数据" },
  { value: "department_tree", label: "本部门及以下", desc: "可查看本部门与全部下级部门" },
  { value: "department", label: "本部门", desc: "仅查看本部门数据" },
  { value: "self", label: "仅本人", desc: "仅查看本人创建或所属数据" },
  { value: "custom", label: "自定义", desc: "按角色关联部门范围查看" },
];

type RoleForm = {
  name: string;
  code: string;
  description: string;
  dataScope: string;
  status: string;
};

const EMPTY_FORM: RoleForm = {
  name: "",
  code: "",
  description: "",
  dataScope: "self",
  status: "active",
};

function flattenMenus(nodes: Menu[]): Menu[] {
  return nodes.flatMap((node) => [node, ...flattenMenus(node.children ?? [])]);
}

function descendantIds(menu: Menu): string[] {
  return [menu.id, ...(menu.children ?? []).flatMap(descendantIds)];
}

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function scopeLabel(value: string): string {
  return SCOPES.find((scope) => scope.value === value)?.label ?? value;
}

function PermissionTree({
  menus,
  selected,
  onChange,
  disabled,
}: {
  menus: Menu[];
  selected: Set<string>;
  onChange: (next: Set<string>) => void;
  disabled?: boolean;
}) {
  const toggle = (menu: Menu) => {
    const next = new Set(selected);
    const ids = descendantIds(menu);
    const allSelected = ids.every((id) => next.has(id));
    ids.forEach((id) => (allSelected ? next.delete(id) : next.add(id)));
    onChange(next);
  };
  const render = (nodes: Menu[], depth: number) =>
    nodes.map((menu) => (
      <div key={menu.id}>
        <label
          className="tree-row"
          style={{ paddingLeft: 8 + depth * 16, cursor: disabled ? "default" : "pointer" }}
        >
          <Checkbox
            checked={selected.has(menu.id)}
            disabled={disabled}
            label=""
            onChange={() => toggle(menu)}
          />
          <span className="tree-label">{menu.name}</span>
          <span className="code-chip">
            {menu.type === 3 ? menu.permission || "按钮" : menu.path || "目录"}
          </span>
        </label>
        {menu.children?.length ? render(menu.children, depth + 1) : null}
      </div>
    ));
  return <div className="tree">{render(menus, 0)}</div>;
}

function PermissionMatrix({
  menus,
  roles,
  grants,
  editable,
  saving,
  onToggle,
  onSave,
}: {
  menus: Menu[];
  roles: Role[];
  grants: Record<string, string[]>;
  editable: boolean;
  saving: boolean;
  onToggle: (roleId: string, menu: Menu) => void;
  onSave: () => void;
}) {
  const flatMenus = useMemo(() => flattenMenus(menus), [menus]);
  return (
    <div className="matrix reveal">
      <div className="band-head" style={{ padding: "16px 18px 0" }}>
        <div>
          <span className="section-label">角色 × 菜单权限矩阵</span>
          <div className="faint" style={{ marginTop: 4 }}>
            点击格子调整菜单与按钮授权，保存时调用真实 Go API。
          </div>
        </div>
        {editable ? (
          <Button variant="primary" loading={saving} onClick={onSave}>
            保存矩阵
          </Button>
        ) : null}
      </div>
      <div className="mx-scroll">
        <div
          className="mx-grid"
          style={{
            ["--mx-cols" as string]: String(roles.length),
          }}
        >
          <div className="mx-head">
            <div className="mx-corner">
              <span className="section-label">菜单 / 角色</span>
            </div>
            {roles.map((role) => (
              <div className="mx-role" key={role.id}>
                <span className="mx-role-name">{role.name}</span>
                <span className="mx-role-meta">{role.code}</span>
              </div>
            ))}
          </div>
          {flatMenus.map((menu) => (
            <div
              className={`mx-row ${menu.type === 1 ? "is-dir" : ""} ${menu.type === 3 ? "is-btn" : ""}`}
              key={menu.id}
            >
              <div className="mx-label">
                <span className="mx-name" style={{ paddingLeft: menu.type === 1 ? 0 : 16 }}>
                  {menu.name}
                </span>
                <span className="mx-type">
                  {menu.type === 1 ? "DIR" : menu.type === 2 ? "PAGE" : "BTN"}
                </span>
              </div>
              {roles.map((role) => {
                const roleId = role.id ?? "";
                const selected = grants[roleId]?.includes(menu.id) ?? false;
                const locked = !editable || role.isSuperAdmin;
                return (
                  <button
                    type="button"
                    key={roleId}
                    className={`mx-cell ${selected ? "is-on" : ""} ${locked ? "is-locked" : ""}`}
                    disabled={locked}
                    aria-label={`${role.name} ${menu.name}`}
                    aria-pressed={selected}
                    onClick={() => onToggle(roleId, menu)}
                  >
                    <span className="mx-mark">
                      {selected ? <Check size={12} strokeWidth={3} /> : null}
                    </span>
                  </button>
                );
              })}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function PermissionTrace({
  focus,
  roles,
  permissions,
  menus,
}: {
  focus: string | null;
  roles: Role[];
  permissions: Permission[];
  menus: Menu[];
}) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(false);
  const requestRef = useRef(0);
  const focusedRole = focus?.startsWith("role:")
    ? roles.find((role) => role.id === focus.slice(5))
    : null;
  const loadFocus = useCallback(async () => {
    if (!focus?.startsWith("user:")) {
      setUser(null);
      return;
    }
    const request = ++requestRef.current;
    setLoading(true);
    try {
      const result = await getUser(focus.slice(5));
      if (request === requestRef.current) setUser(result);
    } catch {
      if (request === requestRef.current) setUser(null);
    } finally {
      if (request === requestRef.current) setLoading(false);
    }
  }, [focus]);
  useDeferredEffect(loadFocus);

  const selectedRoles = user
    ? roles.filter((role) => user.roleIds?.includes(role.id ?? ""))
    : focusedRole
      ? [focusedRole]
      : roles.slice(0, 4);
  const permissionCodes = useMemo(() => {
    const menuCodes = flattenMenus(menus)
      .filter((menu) => menu.permission)
      .map((menu) => menu.permission as string);
    return permissions
      .filter((permission) => menuCodes.includes(permission.code ?? ""))
      .slice(0, 12);
  }, [menus, permissions]);

  return (
    <div className="trace reveal">
      <div className="trace-graph rg">
        <div className="trace-hint faint">
          {focus
            ? `当前聚焦：${user?.displayName ?? focusedRole?.name ?? focus}`
            : "选择一个角色或在地址栏使用 ?trace=role:<uuid> 查看授权链路"}
        </div>
        <div className="stack" style={{ position: "relative", zIndex: 2, padding: 24 }}>
          <div className="trace-legend m-float">
            <div>
              <dt>角色</dt>
              <dd>{selectedRoles.length}</dd>
            </div>
            <div>
              <dt>权限码</dt>
              <dd>{permissionCodes.length}</dd>
            </div>
            <div>
              <dt>菜单节点</dt>
              <dd>{flattenMenus(menus).length}</dd>
            </div>
          </div>
          {loading ? <EmptyState title="正在读取用户授权" icon="shield" /> : null}
          <div
            className="trace-panel"
            style={{ position: "relative", inset: "auto", width: "min(720px, 100%)" }}
          >
            <Card className="org-card" style={{ width: "100%", maxWidth: 720 }}>
              <span className="section-label">授权链路</span>
              <div className="stack" style={{ marginTop: 12 }}>
                <div className="cell-main">
                  <span className="quick-ico m-groove">
                    <UserRoundSearch size={18} />
                  </span>
                  <div>
                    <div className="cell-title">{user?.displayName ?? "当前角色对象"}</div>
                    <div className="cell-sub">
                      {user ? `@${user.username}` : (focus ?? "请选择角色")}
                    </div>
                  </div>
                </div>
                <div className="flag-row">
                  {selectedRoles.map((role) => (
                    <Tag tone="info" key={role.id}>
                      {role.name} · {role.code}
                    </Tag>
                  ))}
                </div>
                <div className="perm-chips">
                  {permissionCodes.map((permission) => (
                    <span className="perm-chip is-on" key={permission.id}>
                      <ShieldCheck size={12} /> {permission.code}
                    </span>
                  ))}
                  {!permissionCodes.length ? <span className="faint">暂无可见权限码</span> : null}
                </div>
                <DetailGrid
                  items={selectedRoles.map((role) => ({
                    label: role.name ?? "角色",
                    value: `${scopeLabel(role.dataScope ?? "self")} · ${role.userCount ?? 0} 位用户`,
                  }))}
                />
              </div>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}

export function RolesView({ traceFocus = null }: { traceFocus?: string | null }) {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canCreate = hasPermission(currentUser, "role:create");
  const canAssign = hasPermission(currentUser, "role:assign");
  const [page, setPage] = useState<RolePage>({ items: [], total: 0, page: 1, pageSize: PAGE_SIZE });
  const [keyword, setKeyword] = useState("");
  const [pageNumber, setPageNumber] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [view, setView] = useState<"table" | "matrix" | "trace">(traceFocus ? "trace" : "table");
  const [menus, setMenus] = useState<Menu[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [grants, setGrants] = useState<Record<string, string[]>>({});
  const [matrixLoading, setMatrixLoading] = useState(false);
  const [matrixSaving, setMatrixSaving] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Role | null>(null);
  const [form, setForm] = useState<RoleForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [permissionRole, setPermissionRole] = useState<Role | null>(null);
  const [permissionSelected, setPermissionSelected] = useState<Set<string>>(new Set());
  const [permissionScope, setPermissionScope] = useState("self");
  const [permissionLoading, setPermissionLoading] = useState(false);
  const [permissionSaving, setPermissionSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      setPage(await listRoles({ page: pageNumber, pageSize: PAGE_SIZE, keyword }));
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [keyword, pageNumber]);

  const loadMatrix = useCallback(async () => {
    setMatrixLoading(true);
    try {
      const [rolePage, menuTree, permissionCatalog] = await Promise.all([
        listRoles({ page: 1, pageSize: 100 }),
        listMenus(),
        listPermissions(),
      ]);
      const entries = await Promise.all(
        rolePage.items
          .filter((role) => role.id)
          .map(async (role) => [role.id as string, await getRoleMenus(role.id as string)] as const),
      );
      setPage(rolePage);
      setMenus(menuTree);
      setPermissions(permissionCatalog);
      setGrants(Object.fromEntries(entries));
    } catch (cause) {
      pushToast({ title: "权限矩阵加载失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setMatrixLoading(false);
    }
  }, [pushToast]);

  useDeferredEffect(load);
  const loadMatrixOnView = useCallback(() => {
    if (view !== "table") return loadMatrix();
  }, [loadMatrix, view]);
  useDeferredEffect(loadMatrixOnView);

  const openCreate = () => {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (role: Role) => {
    setEditing(role);
    setForm({
      name: role.name ?? "",
      code: role.code ?? "",
      description: role.description ?? "",
      dataScope: role.dataScope ?? "self",
      status: role.status ?? "active",
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const submit = async () => {
    if (!form.name.trim() || !form.code.trim()) {
      setFormError("角色名称和编码必填。");
      return;
    }
    setSaving(true);
    setFormError(null);
    try {
      if (editing?.id) {
        const input: UpdateRole = {
          name: form.name.trim(),
          description: form.description.trim() || undefined,
          dataScope: form.dataScope as UpdateRole["dataScope"],
          status: form.status as UpdateRole["status"],
        };
        await updateRole(editing.id, input);
        pushToast({ title: "角色已更新", tone: "ok" });
      } else {
        await createRole({
          name: form.name.trim(),
          code: form.code.trim(),
          description: form.description.trim() || undefined,
          dataScope: form.dataScope as "all" | "custom" | "department" | "department_tree" | "self",
        });
        pushToast({ title: "角色已创建", tone: "ok" });
      }
      setDialogOpen(false);
      await load();
    } catch (cause) {
      setFormError(apiMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!deleteTarget?.id) return;
    setDeleting(true);
    try {
      await deleteRole(deleteTarget.id);
      pushToast({ title: "角色已删除", tone: "ok" });
      setDeleteTarget(null);
      await load();
    } catch (cause) {
      pushToast({ title: "角色删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setDeleting(false);
    }
  };

  const openPermissions = async (role: Role) => {
    if (!role.id) return;
    setPermissionRole(role);
    setPermissionScope(role.dataScope ?? "self");
    setPermissionLoading(true);
    try {
      const ids = await getRoleMenus(role.id);
      setPermissionSelected(new Set(ids));
    } catch (cause) {
      pushToast({ title: "角色菜单加载失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setPermissionLoading(false);
    }
  };

  const savePermissions = async () => {
    if (!permissionRole?.id) return;
    setPermissionSaving(true);
    try {
      const menuIds = [...permissionSelected];
      const codes = flattenMenus(menus)
        .filter((menu) => menuIds.includes(menu.id) && menu.permission)
        .map((menu) => menu.permission as string);
      await Promise.all([
        setRoleMenus(permissionRole.id, menuIds),
        setRolePermissions(permissionRole.id, [...new Set(codes)]),
      ]);
      if (permissionScope !== permissionRole.dataScope) {
        await updateRole(permissionRole.id, {
          name: permissionRole.name ?? "",
          description: permissionRole.description,
          dataScope: permissionScope as UpdateRole["dataScope"],
          status: (permissionRole.status ?? "active") as UpdateRole["status"],
        });
      }
      pushToast({ title: "角色授权已保存", tone: "ok" });
      setPermissionRole(null);
      await load();
      if (view === "matrix") await loadMatrix();
    } catch (cause) {
      pushToast({ title: "角色授权保存失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setPermissionSaving(false);
    }
  };

  const toggleMatrix = (roleId: string, menu: Menu) => {
    const ids = descendantIds(menu);
    const current = grants[roleId] ?? [];
    const all = ids.every((id) => current.includes(id));
    setGrants({
      ...grants,
      [roleId]: all
        ? current.filter((id) => !ids.includes(id))
        : [...new Set([...current, ...ids])],
    });
  };

  const saveMatrix = async () => {
    setMatrixSaving(true);
    try {
      const editableRoles = page.items.filter((role) => role.id && !role.isSuperAdmin);
      for (const role of editableRoles) {
        const roleId = role.id as string;
        const menuIds = grants[roleId] ?? [];
        const codes = flattenMenus(menus)
          .filter((menu) => menuIds.includes(menu.id) && menu.permission)
          .map((menu) => menu.permission as string);
        await Promise.all([
          setRoleMenus(roleId, menuIds),
          setRolePermissions(roleId, [...new Set(codes)]),
        ]);
      }
      pushToast({ title: `已保存 ${editableRoles.length} 个角色的授权`, tone: "ok" });
      await loadMatrix();
    } catch (cause) {
      pushToast({ title: "权限矩阵保存失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setMatrixSaving(false);
    }
  };

  const columns: TableColumn<Role>[] = [
    {
      key: "name",
      header: "角色",
      render: (role) => (
        <div>
          <div className="cell-title">
            {role.name} {role.isSuperAdmin ? <Tag>内置</Tag> : null}
          </div>
          <div className="cell-sub mono">{role.code}</div>
        </div>
      ),
    },
    {
      key: "scope",
      header: "数据权限",
      render: (role) => <span>{scopeLabel(role.dataScope ?? "self")}</span>,
    },
    {
      key: "users",
      header: "用户数",
      align: "right",
      render: (role) => <span className="tnum">{role.userCount ?? 0}</span>,
    },
    { key: "status", header: "状态", render: (role) => <StatusTag status={role.status} /> },
    {
      key: "created",
      header: "创建时间",
      render: (role) =>
        role.createdAt ? new Date(role.createdAt).toLocaleDateString("zh-CN") : "—",
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (role) => (
        <div className="row-actions">
          {canCreate && !role.isSuperAdmin ? (
            <IconButton label="编辑角色" icon="edit" small onClick={() => openEdit(role)} />
          ) : null}
          {canAssign && !role.isSuperAdmin ? (
            <IconButton
              label="分配权限"
              icon="shield"
              small
              onClick={() => void openPermissions(role)}
            />
          ) : null}
          {canCreate && !role.isSuperAdmin ? (
            <IconButton
              label="删除角色"
              icon="trash"
              small
              danger
              onClick={() => setDeleteTarget(role)}
            />
          ) : null}
          {role.isSuperAdmin ? <span className="faint">全部权限</span> : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="角色管理"
      description="角色决定用户可见菜单、按钮与数据范围；矩阵和授权抽屉均写入真实 Go API。"
      fitViewport
      actions={
        <>
          <Tabs
            tabs={[
              { value: "table", label: "列表" },
              { value: "matrix", label: "矩阵" },
              { value: "trace", label: "溯源" },
            ]}
            value={view}
            onChange={(value) => setView(value as "table" | "matrix" | "trace")}
          />
          {canCreate ? (
            <Button variant="primary" icon="plus" onClick={openCreate}>
              新增角色
            </Button>
          ) : null}
        </>
      }
    >
      {view === "table" ? (
        <div className="list-page-stack stack">
          <SearchForm
            loading={loading}
            onSearch={() => {
              setPageNumber(1);
              void load();
            }}
            onReset={() => {
              setKeyword("");
              setPageNumber(1);
            }}
          >
            <Field label="关键词" wide>
              <TextInput
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder="角色名称 / 编码"
                icon="search"
                clearable
                onClear={() => setKeyword("")}
              />
            </Field>
          </SearchForm>
          <Card className="list-page-panel reveal">
            <div className="panel-head">
              <span className="section-label">角色列表</span>
              <span className="panel-meta">共 {page.total ?? 0} 个角色</span>
            </div>
            <DataState
              loading={loading}
              error={error}
              permissionDenied={permissionDenied}
              empty={!loading && !error && !(page.items ?? []).length}
              emptyTitle="暂无角色"
              onRetry={() => void load()}
            >
              <DataTable
                columns={columns}
                rows={page.items ?? []}
                rowKey={(role) => role.id ?? ""}
              />
              <Pagination
                page={pageNumber}
                pageSize={PAGE_SIZE}
                total={page.total ?? 0}
                onChange={setPageNumber}
              />
            </DataState>
          </Card>
        </div>
      ) : view === "matrix" ? (
        matrixLoading ? (
          <Card className="reveal">
            <EmptyState title="正在加载权限矩阵" icon="shield" />
          </Card>
        ) : menus.length ? (
          <PermissionMatrix
            menus={menus}
            roles={page.items ?? []}
            grants={grants}
            editable={canAssign}
            saving={matrixSaving}
            onToggle={toggleMatrix}
            onSave={() => void saveMatrix()}
          />
        ) : (
          <Card className="reveal">
            <EmptyState title="暂无菜单数据" icon="menu" />
          </Card>
        )
      ) : (
        <PermissionTrace
          focus={traceFocus}
          roles={page.items ?? []}
          permissions={permissions}
          menus={menus}
        />
      )}

      <Dialog
        open={dialogOpen}
        title={editing ? "编辑角色" : "新增角色"}
        description="角色编码创建后用于后端授权判断，不建议随意修改。"
        onClose={() => !saving && setDialogOpen(false)}
        width={560}
        footer={
          <>
            <Button variant="ghost" disabled={saving} onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button variant="primary" loading={saving} onClick={() => void submit()}>
              保存
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label="角色名称" required>
            <TextInput
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
              placeholder="如：系统管理员"
            />
          </Field>
          <Field
            label="角色编码"
            required
            hint={editing ? "编码创建后不可修改" : "小写字母开头，可含数字和下划线"}
          >
            <TextInput
              value={form.code}
              disabled={Boolean(editing)}
              onChange={(event) => setForm({ ...form, code: event.target.value })}
              placeholder="sys_admin"
            />
          </Field>
          <Field label="数据范围">
            <Select
              value={form.dataScope}
              onChange={(value) => setForm({ ...form, dataScope: value })}
              options={SCOPES.map((scope) => ({ value: scope.value, label: scope.label }))}
            />
          </Field>
          <Field label="角色状态">
            <Select
              value={form.status}
              onChange={(value) => setForm({ ...form, status: value })}
              options={[
                { value: "active", label: "正常" },
                { value: "disabled", label: "停用" },
              ]}
            />
          </Field>
          <Field label="备注" wide>
            <TextInput
              multiline
              rows={3}
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
              placeholder="角色用途说明"
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
        open={Boolean(permissionRole)}
        title={`分配权限 · ${permissionRole?.name ?? ""}`}
        description={permissionRole?.code}
        onClose={() => !permissionSaving && setPermissionRole(null)}
        width={620}
        footer={
          <>
            <Button variant="ghost" onClick={() => setPermissionRole(null)}>
              取消
            </Button>
            <Button
              variant="primary"
              loading={permissionSaving}
              onClick={() => void savePermissions()}
            >
              保存权限
            </Button>
          </>
        }
      >
        <div className="stack">
          <div className="drawer-toolbar">
            <span className="section-label">菜单与按钮</span>
            <span className="faint">已选 {permissionSelected.size} 个节点</span>
          </div>
          {permissionLoading ? (
            <EmptyState title="正在加载菜单授权" icon="shield" />
          ) : (
            <PermissionTree
              menus={menus}
              selected={permissionSelected}
              onChange={setPermissionSelected}
            />
          )}
          <div className="scope-list" role="radiogroup" aria-label="数据范围">
            {SCOPES.map((scope) => (
              <button
                type="button"
                key={scope.value}
                role="radio"
                aria-checked={permissionScope === scope.value}
                className={`scope-option ${permissionScope === scope.value ? "is-on m-pill" : ""}`}
                onClick={() => setPermissionScope(scope.value)}
              >
                <span className="scope-radio" aria-hidden="true">
                  {permissionScope === scope.value ? <Check size={12} /> : null}
                </span>
                <span>
                  <span className="scope-name">{scope.label}</span>
                  <span className="scope-desc">{scope.desc}</span>
                </span>
              </button>
            ))}
          </div>
        </div>
      </Drawer>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除角色"
        message={`确认删除角色「${deleteTarget?.name ?? ""}」吗？已分配该角色的用户将失去对应权限。`}
        confirmText="删除"
        danger
        loading={deleting}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
