"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { Minus, Plus, Scan, UsersRound } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import {
  BusinessPage,
  DataState,
  DetailGrid,
  SearchForm,
  StatusTag,
} from "@/components/business/business-page";
import { Button, IconButton } from "@/components/ui/button";
import { DataTable, Tabs, type TableColumn } from "@/components/ui/data-table";
import { Field, TextInput } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Dialog, Drawer } from "@/components/ui/overlay";
import { Select } from "@/components/ui/select";
import { Avatar, Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import {
  createDepartment,
  deleteDepartment,
  listDepartments,
  updateDepartment,
} from "@/lib/api/organization";
import { listUsers } from "@/lib/api/users";
import { hasPermission } from "@/lib/permissions";
import type { Department, DepartmentInput, User } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

type FlatDepartment = { department: Department; depth: number };

type DepartmentForm = {
  parentId: string;
  name: string;
  code: string;
  leaderName: string;
  phone: string;
  email: string;
  sort: number;
  status: "active" | "disabled";
};

const EMPTY_FORM: DepartmentForm = {
  parentId: "",
  name: "",
  code: "",
  leaderName: "",
  phone: "",
  email: "",
  sort: 10,
  status: "active",
};

function flatten(nodes: Department[], depth = 0): FlatDepartment[] {
  return nodes.flatMap((department) => [
    { department, depth },
    ...flatten(department.children ?? [], depth + 1),
  ]);
}

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function OrgCanvas({
  departments,
  selectedId,
  memberCounts,
  onSelect,
}: {
  departments: Department[];
  selectedId: string | null;
  memberCounts: Map<string, number>;
  onSelect: (department: Department) => void;
}) {
  const flat = useMemo(() => flatten(departments), [departments]);
  const [zoom, setZoom] = useState(0.86);
  const [pan, setPan] = useState({ x: 32, y: 24 });
  const [panning, setPanning] = useState(false);
  const drag = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null);
  const nodeWidth = 184;
  const nodeHeight = 72;
  const positions = useMemo(() => {
    const byDepth = new Map<number, number>();
    return new Map(
      flat.map(({ department, depth }) => {
        const row = byDepth.get(depth) ?? 0;
        byDepth.set(depth, row + 1);
        return [
          department.id ?? "",
          {
            x: 50 + depth * 240,
            y: 40 + row * 108,
          },
        ];
      }),
    );
  }, [flat]);
  const width = Math.max(920, ...flat.map(({ depth }) => 320 + depth * 240));
  const height = Math.max(
    560,
    ...Array.from(new Set(flat.map(({ depth }) => depth))).map(
      (depth) => (flat.filter((item) => item.depth === depth).length + 1) * 108,
    ),
  );

  return (
    <div
      className={`org-canvas reveal ${panning ? "is-panning" : ""}`}
      onPointerDown={(event) => {
        drag.current = { x: event.clientX, y: event.clientY, panX: pan.x, panY: pan.y };
        setPanning(true);
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (!drag.current) return;
        setPan({
          x: drag.current.panX + event.clientX - drag.current.x,
          y: drag.current.panY + event.clientY - drag.current.y,
        });
      }}
      onPointerUp={() => {
        drag.current = null;
        setPanning(false);
      }}
      onPointerCancel={() => {
        drag.current = null;
        setPanning(false);
      }}
      onWheel={(event) => {
        event.preventDefault();
        setZoom((value) => Math.max(0.45, Math.min(1.8, value - event.deltaY * 0.001)));
      }}
    >
      <div className="org-dots" />
      <div
        className="org-layer"
        style={{ transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`, width, height }}
      >
        <svg className="org-links" width={width} height={height} aria-hidden="true">
          {flat.map(({ department }) => {
            const parent = department.parentId ? positions.get(department.parentId) : null;
            const current = positions.get(department.id ?? "");
            if (!parent || !current) return null;
            const x1 = parent.x + nodeWidth;
            const y1 = parent.y + nodeHeight / 2;
            const x2 = current.x;
            const y2 = current.y + nodeHeight / 2;
            return (
              <path
                key={department.id}
                className="org-link"
                d={`M ${x1} ${y1} C ${x1 + 60} ${y1}, ${x2 - 60} ${y2}, ${x2} ${y2}`}
              />
            );
          })}
        </svg>
        {flat.map(({ department }, index) => {
          const id = department.id ?? "";
          const position = positions.get(id);
          if (!position) return null;
          return (
            <button
              type="button"
              key={id}
              className={`org-node cat-${(index % 5) + 1} ${selectedId === id ? "is-selected" : ""} ${department.status === "disabled" ? "is-off" : ""}`}
              style={{
                left: position.x,
                top: position.y,
                width: nodeWidth,
                height: nodeHeight,
              }}
              onClick={(event) => {
                event.stopPropagation();
                onSelect(department);
              }}
              onPointerDown={(event) => event.stopPropagation()}
            >
              <span className="org-strip" />
              <span className="org-name">{department.name}</span>
              <span className="org-meta">
                <span className="org-leader">{department.leaderName || "未指定负责人"}</span>
                <span className="org-count">{memberCounts.get(id) ?? 0} 人</span>
              </span>
            </button>
          );
        })}
      </div>
      <div className="org-controls m-float" onPointerDown={(event) => event.stopPropagation()}>
        <IconButton
          label="缩小"
          icon="minus"
          small
          onClick={() => setZoom((value) => Math.max(0.45, value - 0.1))}
        />
        <IconButton
          label="放大"
          icon="plus"
          small
          onClick={() => setZoom((value) => Math.min(1.8, value + 0.1))}
        />
        <IconButton
          label="适配画布"
          icon="scan"
          small
          onClick={() => {
            setZoom(0.86);
            setPan({ x: 32, y: 24 });
          }}
        />
      </div>
    </div>
  );
}

export function DepartmentsView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "department:manage");
  const canReadUsers = hasPermission(currentUser, "user:read");
  const [departments, setDepartments] = useState<Department[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [view, setView] = useState<"table" | "canvas">("table");
  const [selected, setSelected] = useState<Department | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Department | null>(null);
  const [form, setForm] = useState<DepartmentForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Department | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const [tree, userPage] = await Promise.all([
        listDepartments(),
        canReadUsers ? listUsers({ page: 1, pageSize: 100 }) : Promise.resolve(null),
      ]);
      setDepartments(tree);
      setUsers(userPage?.items ?? []);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [canReadUsers]);

  useDeferredEffect(load);

  const flatDepartments = useMemo(() => flatten(departments), [departments]);
  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return flatDepartments;
    return flatDepartments.filter(({ department }) =>
      [department.name, department.code, department.leaderName]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [flatDepartments, keyword]);
  const memberCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const user of users) {
      for (const departmentId of user.departmentIds ?? []) {
        counts.set(departmentId, (counts.get(departmentId) ?? 0) + 1);
      }
    }
    return counts;
  }, [users]);
  const selectedMembers = selected?.id
    ? users.filter((user) => user.departmentIds?.includes(selected.id as string))
    : [];

  const openCreate = (parent?: Department) => {
    setEditing(null);
    setForm({ ...EMPTY_FORM, parentId: parent?.id ?? "" });
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (department: Department) => {
    setEditing(department);
    setForm({
      parentId: department.parentId ?? "",
      name: department.name ?? "",
      code: department.code ?? "",
      leaderName: department.leaderName ?? "",
      phone: department.phone ?? "",
      email: department.email ?? "",
      sort: department.sort ?? 10,
      status: (department.status ?? "active") as "active" | "disabled",
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const submit = async () => {
    if (!form.name.trim() || !form.code.trim()) {
      setFormError("部门名称和编码必填。");
      return;
    }
    setSaving(true);
    setFormError(null);
    const input: DepartmentInput = {
      parentId: form.parentId || undefined,
      name: form.name.trim(),
      code: form.code.trim(),
      leaderName: form.leaderName.trim(),
      phone: form.phone.trim(),
      email: form.email.trim(),
      sort: form.sort,
      status: form.status,
    };
    try {
      if (editing?.id) {
        await updateDepartment(editing.id, input);
        pushToast({ title: "部门已更新", tone: "ok" });
      } else {
        await createDepartment(input);
        pushToast({ title: "部门已创建", tone: "ok" });
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
      await deleteDepartment(deleteTarget.id);
      pushToast({ title: "部门已删除", tone: "ok" });
      setDeleteTarget(null);
      setSelected(null);
      await load();
    } catch (cause) {
      pushToast({ title: "部门删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setDeleting(false);
    }
  };

  const columns: TableColumn<FlatDepartment>[] = [
    {
      key: "name",
      header: "部门",
      render: ({ department, depth }) => (
        <div className="tree-cell" style={{ paddingLeft: depth * 18 }}>
          <span className="cell-title">{department.name}</span>
          <span className="cell-sub mono">{department.code}</span>
        </div>
      ),
    },
    {
      key: "leader",
      header: "负责人",
      render: ({ department }) => department.leaderName || <span className="faint">未指定</span>,
    },
    {
      key: "members",
      header: "成员",
      align: "right",
      render: ({ department }) => (
        <span className="tnum">{memberCounts.get(department.id ?? "") ?? 0}</span>
      ),
    },
    {
      key: "status",
      header: "状态",
      render: ({ department }) => <StatusTag status={department.status} />,
    },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: ({ department }) => (
        <div className="row-actions">
          {canManage ? (
            <>
              <IconButton
                label="新增子部门"
                icon="plus"
                small
                onClick={() => openCreate(department)}
              />
              <IconButton label="编辑部门" icon="edit" small onClick={() => openEdit(department)} />
              <IconButton
                label="删除部门"
                icon="trash"
                small
                danger
                onClick={() => setDeleteTarget(department)}
              />
            </>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="部门管理"
      description="维护真实组织树、负责人和状态；组织画布支持平移、缩放与视口复位。"
      fitViewport
      actions={
        <>
          <Tabs
            tabs={[
              { value: "table", label: "表格" },
              { value: "canvas", label: "组织画布" },
            ]}
            value={view}
            onChange={(value) => setView(value as "table" | "canvas")}
          />
          {canManage ? (
            <Button variant="primary" icon="plus" onClick={() => openCreate()}>
              新增部门
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
              placeholder="部门名称 / 编码 / 负责人"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
        </SearchForm>
        {view === "table" ? (
          <Card className="list-page-panel reveal">
            <div className="panel-head">
              <span className="section-label">组织树</span>
              <span className="panel-meta">{flatDepartments.length} 个部门</span>
            </div>
            <DataState
              loading={loading}
              error={error}
              permissionDenied={permissionDenied}
              empty={!loading && !error && !filtered.length}
              emptyTitle="暂无部门数据"
              onRetry={() => void load()}
            >
              <DataTable
                columns={columns}
                rows={filtered}
                rowKey={({ department }) => department.id ?? ""}
              />
            </DataState>
          </Card>
        ) : (
          <OrgCanvas
            departments={departments}
            selectedId={selected?.id ?? null}
            memberCounts={memberCounts}
            onSelect={setSelected}
          />
        )}
      </div>

      <Dialog
        open={dialogOpen}
        title={editing ? "编辑部门" : "新增部门"}
        onClose={() => !saving && setDialogOpen(false)}
        width={620}
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
          <Field label="上级部门">
            <Select
              value={form.parentId}
              onChange={(value) => setForm({ ...form, parentId: value })}
              options={[
                { value: "", label: "顶级部门" },
                ...flatDepartments
                  .filter(({ department }) => department.id !== editing?.id)
                  .map(({ department, depth }) => ({
                    value: department.id ?? "",
                    label: `${"— ".repeat(depth)}${department.name}`,
                  })),
              ]}
            />
          </Field>
          <Field label="部门名称" required>
            <TextInput
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field label="部门编码" required>
            <TextInput
              value={form.code}
              onChange={(event) => setForm({ ...form, code: event.target.value })}
              placeholder="engineering"
            />
          </Field>
          <Field label="负责人">
            <TextInput
              value={form.leaderName}
              onChange={(event) => setForm({ ...form, leaderName: event.target.value })}
            />
          </Field>
          <Field label="联系电话">
            <TextInput
              value={form.phone}
              onChange={(event) => setForm({ ...form, phone: event.target.value })}
            />
          </Field>
          <Field label="邮箱">
            <TextInput
              type="email"
              value={form.email}
              onChange={(event) => setForm({ ...form, email: event.target.value })}
            />
          </Field>
          <Field label="排序">
            <TextInput
              type="number"
              value={form.sort}
              onChange={(event) => setForm({ ...form, sort: Number(event.target.value) })}
            />
          </Field>
          <Field label="状态">
            <Select
              value={form.status}
              onChange={(value) => setForm({ ...form, status: value as "active" | "disabled" })}
              options={[
                { value: "active", label: "正常" },
                { value: "disabled", label: "停用" },
              ]}
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
        open={Boolean(selected)}
        title={selected?.name ?? "部门详情"}
        description={selected?.path}
        onClose={() => setSelected(null)}
        width={480}
      >
        {selected ? (
          <div className="stack">
            <DetailGrid
              items={[
                { label: "部门编码", value: selected.code ?? "—" },
                { label: "负责人", value: selected.leaderName || "未指定" },
                { label: "联系电话", value: selected.phone || "—" },
                { label: "邮箱", value: selected.email || "—" },
                { label: "成员数量", value: selectedMembers.length },
                { label: "状态", value: <StatusTag status={selected.status} /> },
              ]}
            />
            <Card>
              <div className="panel-head">
                <span className="section-label">部门成员</span>
                <Tag tone="info">
                  <UsersRound size={12} /> {selectedMembers.length}
                </Tag>
              </div>
              <div className="panel-body stack">
                {selectedMembers.length ? (
                  selectedMembers.map((member) => (
                    <div className="activity" key={member.id}>
                      <Avatar name={member.displayName} size={30} />
                      <div className="activity-main">
                        <div className="cell-title">{member.displayName}</div>
                        <div className="cell-sub">@{member.username}</div>
                      </div>
                      <StatusTag status={member.status} />
                    </div>
                  ))
                ) : (
                  <EmptyState
                    title="暂无可见成员"
                    description={
                      canReadUsers
                        ? "该部门当前没有成员。"
                        : "缺少 user:read 权限，成员列表不可见。"
                    }
                    icon="users"
                  />
                )}
              </div>
            </Card>
            {canManage ? (
              <div className="drawer-toolbar">
                <Button icon="edit" onClick={() => openEdit(selected)}>
                  编辑部门
                </Button>
                <Button variant="danger" icon="trash" onClick={() => setDeleteTarget(selected)}>
                  删除部门
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}
      </Drawer>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除部门"
        message={`确认删除部门「${deleteTarget?.name ?? ""}」吗？后端会校验子部门与成员关系。`}
        confirmText="删除"
        danger
        loading={deleting}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
