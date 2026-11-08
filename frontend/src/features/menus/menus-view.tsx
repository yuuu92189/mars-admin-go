"use client";

import { useCallback, useMemo, useState } from "react";
import { ChevronRight, Plus, Star } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import {
  BusinessPage,
  DataState,
  SearchForm,
  StatusTag,
} from "@/components/business/business-page";
import { Button, IconButton } from "@/components/ui/button";
import { DataTable, Tabs, type TableColumn } from "@/components/ui/data-table";
import { Checkbox, Field, Switch, TextInput } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Dialog } from "@/components/ui/overlay";
import { Select } from "@/components/ui/select";
import { Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import { createMenu, deleteMenu, listMenus, updateMenu } from "@/lib/api/access";
import { hasPermission } from "@/lib/permissions";
import type { Menu, MenuInput } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

type FlatMenu = { menu: Menu; depth: number };

type MenuForm = {
  parentId: string;
  name: string;
  type: number;
  path: string;
  component: string;
  permission: string;
  icon: string;
  sort: number;
  visible: boolean;
  isFrame: boolean;
};

const EMPTY_FORM: MenuForm = {
  parentId: "",
  name: "",
  type: 2,
  path: "",
  component: "",
  permission: "",
  icon: "",
  sort: 10,
  visible: true,
  isFrame: false,
};

function flatten(nodes: Menu[], depth = 0): FlatMenu[] {
  return nodes.flatMap((menu) => [{ menu, depth }, ...flatten(menu.children ?? [], depth + 1)]);
}

function descendants(menu: Menu): string[] {
  return [menu.id, ...(menu.children ?? []).flatMap(descendants)];
}

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function typeLabel(type: number): string {
  return type === 1 ? "目录" : type === 2 ? "菜单" : "按钮";
}

function MenuStarMap({
  menus,
  selectedId,
  onSelect,
}: {
  menus: Menu[];
  selectedId: string | null;
  onSelect: (menu: Menu) => void;
}) {
  const items = flatten(menus);
  const width = 900;
  const height = 600;
  const center = { x: width / 2, y: height / 2 };
  const positions = items.map(({ menu, depth }, index) => {
    const ring = Math.min(4, depth + 1);
    const radius = 90 + ring * 80;
    const angle = (index / Math.max(1, items.length)) * Math.PI * 2 - Math.PI / 2;
    return {
      menu,
      x: center.x + Math.cos(angle) * radius,
      y: center.y + Math.sin(angle) * radius,
    };
  });
  const byId = new Map(positions.map((item) => [item.menu.id, item]));
  const links = positions.flatMap((item) =>
    item.menu.parentId && byId.get(item.menu.parentId)
      ? [[byId.get(item.menu.parentId)!, item] as const]
      : [],
  );
  return (
    <div className="rg reveal" style={{ minHeight: 600 }} role="img" aria-label="菜单导航星图">
      <svg
        className="rg-stage-canvas"
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="xMidYMid meet"
      >
        {links.map(([from, to], index) => (
          <path
            key={`${from.menu.id}-${to.menu.id}-${index}`}
            d={`M ${from.x} ${from.y} Q ${center.x} ${center.y} ${to.x} ${to.y}`}
            className="rg-path"
            style={{ stroke: "var(--thread)" }}
          />
        ))}
        <circle
          cx={center.x}
          cy={center.y}
          r="42"
          className="m-groove"
          fill="var(--surface-solid)"
          stroke="var(--line-strong)"
        />
        <text
          x={center.x}
          y={center.y + 4}
          textAnchor="middle"
          style={{ fill: "var(--ink)", fontSize: 13, fontWeight: 700 }}
        >
          Mars
        </text>
        {positions.map(({ menu, x, y }) => (
          <g key={menu.id} onClick={() => onSelect(menu)} style={{ cursor: "pointer" }}>
            <circle
              cx={x}
              cy={y}
              r={selectedId === menu.id ? 12 : 8}
              fill={selectedId === menu.id ? "var(--blue)" : "var(--surface-solid)"}
              stroke={menu.visible ? "var(--blue)" : "var(--ink-faint)"}
              strokeWidth="2"
            />
            <text
              x={x}
              y={y + 23}
              textAnchor="middle"
              style={{ fill: "var(--ink-muted)", fontSize: 11 }}
            >
              {menu.name.length > 10 ? `${menu.name.slice(0, 10)}…` : menu.name}
            </text>
          </g>
        ))}
      </svg>
      <div style={{ position: "absolute", left: 18, bottom: 18 }} className="faint">
        星图节点来自 Go 菜单树 · 实心为当前选中
      </div>
    </div>
  );
}

export function MenusView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canAssign = hasPermission(currentUser, "role:assign");
  const [menus, setMenus] = useState<Menu[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [view, setView] = useState<"table" | "starmap">("table");
  const [selected, setSelected] = useState<Menu | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Menu | null>(null);
  const [form, setForm] = useState<MenuForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Menu | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      setMenus(await listMenus());
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useDeferredEffect(load);

  const flatMenus = useMemo(() => flatten(menus), [menus]);
  const filteredMenus = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return flatMenus;
    return flatMenus.filter(({ menu }) =>
      [menu.name, menu.path, menu.permission, menu.component]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [flatMenus, keyword]);

  const openCreate = (parent?: Menu) => {
    setEditing(null);
    setForm({ ...EMPTY_FORM, parentId: parent?.id ?? "" });
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (menu: Menu) => {
    setEditing(menu);
    setForm({
      parentId: menu.parentId ?? "",
      name: menu.name,
      type: menu.type,
      path: menu.path ?? "",
      component: menu.component ?? "",
      permission: menu.permission ?? "",
      icon: menu.icon ?? "",
      sort: menu.sort,
      visible: menu.visible,
      isFrame: menu.isFrame,
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const submit = async () => {
    if (!form.name.trim()) {
      setFormError("菜单名称必填。");
      return;
    }
    if (form.type === 2 && !form.path.trim()) {
      setFormError("菜单类型必须填写路由路径。");
      return;
    }
    setSaving(true);
    setFormError(null);
    const input: MenuInput = {
      parentId: form.parentId || undefined,
      name: form.name.trim(),
      type: form.type as 1 | 2 | 3,
      path: form.path.trim(),
      component: form.component.trim(),
      permission: form.permission.trim(),
      icon: form.icon.trim(),
      sort: form.sort,
      visible: form.visible,
      isFrame: form.isFrame,
    };
    try {
      if (editing) {
        await updateMenu(editing.id, input);
        pushToast({ title: "菜单已更新", tone: "ok" });
      } else {
        await createMenu(input);
        pushToast({ title: "菜单已创建", tone: "ok" });
      }
      setDialogOpen(false);
      await load();
    } catch (cause) {
      setFormError(apiMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const toggleVisible = async (menu: Menu) => {
    if (!canAssign) return;
    try {
      await updateMenu(menu.id, {
        parentId: menu.parentId,
        name: menu.name,
        type: menu.type,
        path: menu.path ?? "",
        component: menu.component ?? "",
        permission: menu.permission ?? "",
        icon: menu.icon ?? "",
        sort: menu.sort,
        visible: !menu.visible,
        isFrame: menu.isFrame,
      });
      pushToast({ title: menu.visible ? "菜单已隐藏" : "菜单已显示", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "菜单状态更新失败", message: apiMessage(cause), tone: "danger" });
    }
  };

  const remove = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteMenu(deleteTarget.id);
      pushToast({ title: "菜单已删除", tone: "ok" });
      setDeleteTarget(null);
      await load();
    } catch (cause) {
      pushToast({ title: "菜单删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setDeleting(false);
    }
  };

  const columns: TableColumn<FlatMenu>[] = [
    {
      key: "name",
      header: "菜单名称",
      render: ({ menu, depth }) => (
        <div className="tree-cell" style={{ paddingLeft: depth * 18 }}>
          {menu.children?.length ? <ChevronRight size={14} /> : <span style={{ width: 14 }} />}
          <span className="cell-title">{menu.name}</span>
          {menu.type === 1 ? <Tag>目录</Tag> : null}
        </div>
      ),
    },
    {
      key: "type",
      header: "类型",
      render: ({ menu }) => <span className="muted">{typeLabel(menu.type)}</span>,
    },
    {
      key: "path",
      header: "路由 / 组件",
      render: ({ menu }) => (
        <div>
          <div className="code-chip">{menu.path || "—"}</div>
          <div className="cell-sub">{menu.component || "—"}</div>
        </div>
      ),
    },
    {
      key: "permission",
      header: "权限标识",
      render: ({ menu }) => <span className="code-chip">{menu.permission || "—"}</span>,
    },
    {
      key: "visible",
      header: "可见",
      render: ({ menu }) => (
        <button
          type="button"
          className="link-btn"
          disabled={!canAssign}
          onClick={() => void toggleVisible(menu)}
        >
          <StatusTag
            status={menu.visible ? "active" : "disabled"}
            label={menu.visible ? "显示" : "隐藏"}
          />
        </button>
      ),
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: ({ menu }) => (
        <div className="row-actions">
          {canAssign ? (
            <>
              <IconButton label="新增子菜单" icon="plus" small onClick={() => openCreate(menu)} />
              <IconButton label="编辑菜单" icon="edit" small onClick={() => openEdit(menu)} />
              <IconButton
                label="删除菜单"
                icon="trash"
                small
                danger
                onClick={() => setDeleteTarget(menu)}
              />
            </>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="菜单管理"
      description="维护真实菜单树、路由映射、权限标识与可见性；路由路径不会被当作可执行组件。"
      fitViewport
      actions={
        <>
          <Tabs
            tabs={[
              { value: "table", label: "树表" },
              { value: "starmap", label: "导航星图" },
            ]}
            value={view}
            onChange={(value) => setView(value as "table" | "starmap")}
          />
          {canAssign ? (
            <Button variant="primary" icon="plus" onClick={() => openCreate()}>
              新增菜单
            </Button>
          ) : null}
        </>
      }
    >
      <div className="list-page-stack stack">
        <SearchForm
          loading={loading}
          onSearch={() => void load()}
          onReset={() => {
            setKeyword("");
            void load();
          }}
        >
          <Field label="关键词" wide>
            <TextInput
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="菜单名称 / 路由 / 权限码"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
        </SearchForm>
        {view === "table" ? (
          <Card className="list-page-panel reveal">
            <div className="panel-head">
              <span className="section-label">菜单树</span>
              <span className="panel-meta">{flatMenus.length} 个节点</span>
            </div>
            <DataState
              loading={loading}
              error={error}
              permissionDenied={permissionDenied}
              empty={!loading && !error && !filteredMenus.length}
              emptyTitle="暂无菜单数据"
              onRetry={() => void load()}
            >
              <DataTable columns={columns} rows={filteredMenus} rowKey={({ menu }) => menu.id} />
            </DataState>
          </Card>
        ) : (
          <MenuStarMap menus={menus} selectedId={selected?.id ?? null} onSelect={setSelected} />
        )}
      </div>

      <Dialog
        open={dialogOpen}
        title={editing ? "编辑菜单" : "新增菜单"}
        onClose={() => !saving && setDialogOpen(false)}
        width={640}
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
          <Field label="上级菜单">
            <Select
              value={form.parentId}
              onChange={(value) => setForm({ ...form, parentId: value })}
              options={[
                { value: "", label: "顶级菜单" },
                ...flatMenus
                  .filter(({ menu }) => menu.id !== editing?.id)
                  .map(({ menu, depth }) => ({
                    value: menu.id,
                    label: `${"— ".repeat(depth)}${menu.name}`,
                  })),
              ]}
            />
          </Field>
          <Field label="菜单名称" required>
            <TextInput
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field label="菜单类型">
            <Select
              value={String(form.type)}
              onChange={(value) => setForm({ ...form, type: Number(value) })}
              options={[
                { value: "1", label: "目录" },
                { value: "2", label: "菜单" },
                { value: "3", label: "按钮" },
              ]}
            />
          </Field>
          <Field label="显示排序">
            <TextInput
              type="number"
              value={form.sort}
              onChange={(event) => setForm({ ...form, sort: Number(event.target.value) })}
            />
          </Field>
          <Field label="路由路径" wide>
            <TextInput
              value={form.path}
              onChange={(event) => setForm({ ...form, path: event.target.value })}
              placeholder="/system/user"
            />
          </Field>
          <Field label="组件映射">
            <TextInput
              value={form.component}
              onChange={(event) => setForm({ ...form, component: event.target.value })}
              placeholder="仅记录，不执行任意路径"
            />
          </Field>
          <Field label="权限标识">
            <TextInput
              value={form.permission}
              onChange={(event) => setForm({ ...form, permission: event.target.value })}
              placeholder="user:read"
            />
          </Field>
          <Field label="图标">
            <TextInput
              value={form.icon}
              onChange={(event) => setForm({ ...form, icon: event.target.value })}
              placeholder="users"
            />
          </Field>
          <Field label="显示状态">
            <Switch
              checked={form.visible}
              label={form.visible ? "显示" : "隐藏"}
              onChange={(visible) => setForm({ ...form, visible })}
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
        title="删除菜单"
        message={`确认删除菜单「${deleteTarget?.name ?? ""}」及其子节点吗？`}
        confirmText="删除"
        danger
        loading={deleting}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
