"use client";

import { useCallback, useMemo, useState } from "react";
import { BriefcaseBusiness, Plus } from "lucide-react";
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
  createPosition,
  deletePosition,
  listPositions,
  updatePosition,
} from "@/lib/api/organization";
import { listUsers } from "@/lib/api/users";
import { hasPermission } from "@/lib/permissions";
import type { Position, PositionInput, User } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

type PositionForm = {
  parentId: string;
  code: string;
  name: string;
  sort: number;
  status: "active" | "disabled";
  description: string;
};

const EMPTY_FORM: PositionForm = {
  parentId: "",
  code: "",
  name: "",
  sort: 10,
  status: "active",
  description: "",
};

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function PostLadder({
  positions,
  members,
  onSelect,
}: {
  positions: Position[];
  members: Map<string, User[]>;
  onSelect: (position: Position) => void;
}) {
  const sorted = [...positions].sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0));
  return (
    <div className="ladder reveal">
      <div className="band-head" style={{ padding: "18px 20px 0" }}>
        <div>
          <span className="section-label">岗位阶梯</span>
          <div className="faint" style={{ marginTop: 4 }}>
            按真实排序值展开，节点人数来自当前可见用户。
          </div>
        </div>
      </div>
      <div className="ladder-stage">
        {sorted.map((position, index) => {
          const people = members.get(position.id ?? "") ?? [];
          return (
            <button
              type="button"
              className={`ladder-step cat-${(index % 5) + 1}`}
              key={position.id}
              style={{ marginLeft: `${Math.min(index, 5) * 34}px` }}
              onClick={() => onSelect(position)}
            >
              <span className="ladder-index">{String(index + 1).padStart(2, "0")}</span>
              <span className="ladder-main">
                <span className="ladder-name">{position.name}</span>
                <span className="ladder-code">{position.code}</span>
              </span>
              <span className="ladder-people">
                <b>{people.length}</b>
                <small>人</small>
              </span>
              <StatusTag status={position.status} />
            </button>
          );
        })}
        {!sorted.length ? <EmptyState title="暂无岗位" icon="post" /> : null}
      </div>
    </div>
  );
}

export function PositionsView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "department:manage");
  const canReadUsers = hasPermission(currentUser, "user:read");
  const [positions, setPositions] = useState<Position[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [view, setView] = useState<"table" | "ladder">("table");
  const [selected, setSelected] = useState<Position | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Position | null>(null);
  const [form, setForm] = useState<PositionForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Position | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const [positionList, userPage] = await Promise.all([
        listPositions(),
        canReadUsers ? listUsers({ page: 1, pageSize: 100 }) : Promise.resolve(null),
      ]);
      setPositions(positionList);
      setUsers(userPage?.items ?? []);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [canReadUsers]);

  useDeferredEffect(load);

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return positions;
    return positions.filter((position) =>
      [position.name, position.code, position.description]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [keyword, positions]);
  const members = useMemo(() => {
    const result = new Map<string, User[]>();
    for (const user of users) {
      for (const positionId of user.positionIds ?? []) {
        result.set(positionId, [...(result.get(positionId) ?? []), user]);
      }
    }
    return result;
  }, [users]);
  const selectedMembers = selected?.id ? (members.get(selected.id) ?? []) : [];

  const openCreate = () => {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (position: Position) => {
    setEditing(position);
    setForm({
      parentId: position.parentId ?? "",
      code: position.code ?? "",
      name: position.name ?? "",
      sort: position.sort ?? 10,
      status: (position.status ?? "active") as "active" | "disabled",
      description: position.description ?? "",
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const submit = async () => {
    if (!form.name.trim() || !form.code.trim()) {
      setFormError("岗位名称和编码必填。");
      return;
    }
    setSaving(true);
    setFormError(null);
    const input: PositionInput = {
      parentId: form.parentId || undefined,
      code: form.code.trim(),
      name: form.name.trim(),
      sort: form.sort,
      status: form.status,
      description: form.description.trim(),
    };
    try {
      if (editing?.id) {
        await updatePosition(editing.id, input);
        pushToast({ title: "岗位已更新", tone: "ok" });
      } else {
        await createPosition(input);
        pushToast({ title: "岗位已创建", tone: "ok" });
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
      await deletePosition(deleteTarget.id);
      pushToast({ title: "岗位已删除", tone: "ok" });
      setDeleteTarget(null);
      setSelected(null);
      await load();
    } catch (cause) {
      pushToast({ title: "岗位删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setDeleting(false);
    }
  };

  const columns: TableColumn<Position>[] = [
    {
      key: "name",
      header: "岗位",
      render: (position) => (
        <div className="cell-main">
          <span className="row-ico m-groove">
            <BriefcaseBusiness size={15} />
          </span>
          <div>
            <div className="cell-title">{position.name}</div>
            <div className="cell-sub mono">{position.code}</div>
          </div>
        </div>
      ),
    },
    {
      key: "description",
      header: "说明",
      render: (position) => <span className="muted ellipsis">{position.description || "—"}</span>,
    },
    {
      key: "members",
      header: "成员",
      align: "right",
      render: (position) => (
        <span className="tnum">{members.get(position.id ?? "")?.length ?? 0}</span>
      ),
    },
    {
      key: "sort",
      header: "排序",
      align: "right",
      render: (position) => <span className="tnum">{position.sort ?? 0}</span>,
    },
    { key: "status", header: "状态", render: (position) => <StatusTag status={position.status} /> },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (position) => (
        <div className="row-actions">
          {canManage ? (
            <>
              <IconButton label="编辑岗位" icon="edit" small onClick={() => openEdit(position)} />
              <IconButton
                label="删除岗位"
                icon="trash"
                small
                danger
                onClick={() => setDeleteTarget(position)}
              />
            </>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="岗位管理"
      description="维护岗位层级、排序、状态与成员关系；岗位阶梯使用真实组织数据。"
      fitViewport
      actions={
        <>
          <Tabs
            tabs={[
              { value: "table", label: "列表" },
              { value: "ladder", label: "岗位阶梯" },
            ]}
            value={view}
            onChange={(value) => setView(value as "table" | "ladder")}
          />
          {canManage ? (
            <Button variant="primary" icon="plus" onClick={openCreate}>
              新增岗位
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
              placeholder="岗位名称 / 编码 / 说明"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
        </SearchForm>
        {view === "table" ? (
          <Card className="list-page-panel reveal">
            <div className="panel-head">
              <span className="section-label">岗位列表</span>
              <span className="panel-meta">{positions.length} 个岗位</span>
            </div>
            <DataState
              loading={loading}
              error={error}
              permissionDenied={permissionDenied}
              empty={!loading && !error && !filtered.length}
              emptyTitle="暂无岗位数据"
              onRetry={() => void load()}
            >
              <DataTable
                columns={columns}
                rows={filtered}
                rowKey={(position) => position.id ?? ""}
              />
            </DataState>
          </Card>
        ) : (
          <PostLadder positions={positions} members={members} onSelect={setSelected} />
        )}
      </div>

      <Dialog
        open={dialogOpen}
        title={editing ? "编辑岗位" : "新增岗位"}
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
          <Field label="上级岗位">
            <Select
              value={form.parentId}
              onChange={(value) => setForm({ ...form, parentId: value })}
              options={[
                { value: "", label: "顶级岗位" },
                ...positions
                  .filter((position) => position.id !== editing?.id)
                  .map((position) => ({ value: position.id ?? "", label: position.name ?? "" })),
              ]}
            />
          </Field>
          <Field label="岗位名称" required>
            <TextInput
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field label="岗位编码" required>
            <TextInput
              value={form.code}
              onChange={(event) => setForm({ ...form, code: event.target.value })}
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
          <Field label="岗位说明" wide>
            <TextInput
              multiline
              rows={3}
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
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
        title={selected?.name ?? "岗位详情"}
        description={selected?.code}
        onClose={() => setSelected(null)}
        width={460}
      >
        {selected ? (
          <div className="stack">
            <DetailGrid
              items={[
                { label: "岗位编码", value: selected.code ?? "—" },
                { label: "排序", value: selected.sort ?? 0 },
                { label: "状态", value: <StatusTag status={selected.status} /> },
                { label: "说明", value: selected.description || "—", wide: true },
              ]}
            />
            <Card>
              <div className="panel-head">
                <span className="section-label">岗位成员</span>
                <Tag tone="info">{selectedMembers.length}</Tag>
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
                    description={canReadUsers ? "该岗位当前没有成员。" : "缺少 user:read 权限。"}
                    icon="users"
                  />
                )}
              </div>
            </Card>
            {canManage ? (
              <div className="drawer-toolbar">
                <Button icon="edit" onClick={() => openEdit(selected)}>
                  编辑岗位
                </Button>
                <Button variant="danger" icon="trash" onClick={() => setDeleteTarget(selected)}>
                  删除岗位
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}
      </Drawer>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除岗位"
        message={`确认删除岗位「${deleteTarget?.name ?? ""}」吗？`}
        confirmText="删除"
        danger
        loading={deleting}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
