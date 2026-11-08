"use client";

import { useCallback, useMemo, useState } from "react";
import { BookOpen, Palette, Plus } from "lucide-react";
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
import { Dialog, Drawer } from "@/components/ui/overlay";
import { Select } from "@/components/ui/select";
import { Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import {
  createDictionary,
  createDictionaryItem,
  deleteDictionary,
  deleteDictionaryItem,
  listDictionaries,
  listDictionaryItems,
  updateDictionary,
  updateDictionaryItem,
} from "@/lib/api/system";
import { hasPermission } from "@/lib/permissions";
import type {
  DictionaryItem,
  DictionaryItemInput,
  DictionaryType,
  DictionaryTypeInput,
} from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

type TypeForm = {
  code: string;
  name: string;
  description: string;
  status: "active" | "disabled";
};

type ItemForm = {
  value: string;
  label: string;
  sort: number;
  tagStyle: string;
  isDefault: boolean;
  status: "active" | "disabled";
};

const EMPTY_TYPE: TypeForm = { code: "", name: "", description: "", status: "active" };
const EMPTY_ITEM: ItemForm = {
  value: "",
  label: "",
  sort: 10,
  tagStyle: "default",
  isDefault: false,
  status: "active",
};

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function swatchTone(value: string): string {
  const tones = ["lime", "blue", "red", "amber", "violet"];
  let hash = 0;
  for (const char of value) hash = (hash + char.charCodeAt(0)) % tones.length;
  return tones[hash] ?? "blue";
}

export function DictionariesView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "dictionary:manage");
  const [types, setTypes] = useState<DictionaryType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [view, setView] = useState<"swatch" | "table">("swatch");
  const [typeDialog, setTypeDialog] = useState(false);
  const [editingType, setEditingType] = useState<DictionaryType | null>(null);
  const [typeForm, setTypeForm] = useState<TypeForm>(EMPTY_TYPE);
  const [typeError, setTypeError] = useState<string | null>(null);
  const [typeSaving, setTypeSaving] = useState(false);
  const [deleteType, setDeleteType] = useState<DictionaryType | null>(null);
  const [activeType, setActiveType] = useState<DictionaryType | null>(null);
  const [items, setItems] = useState<DictionaryItem[]>([]);
  const [itemsLoading, setItemsLoading] = useState(false);
  const [itemDialog, setItemDialog] = useState(false);
  const [editingItem, setEditingItem] = useState<DictionaryItem | null>(null);
  const [itemForm, setItemForm] = useState<ItemForm>(EMPTY_ITEM);
  const [itemError, setItemError] = useState<string | null>(null);
  const [itemSaving, setItemSaving] = useState(false);
  const [deleteItem, setDeleteItem] = useState<DictionaryItem | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const result = await listDictionaries();
      setTypes(result);
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useDeferredEffect(load);

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return types;
    return types.filter((type) =>
      [type.name, type.code, type.description]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [keyword, types]);

  const loadItems = useCallback(
    async (type: DictionaryType) => {
      setActiveType(type);
      setItemsLoading(true);
      try {
        setItems(await listDictionaryItems(type.id));
      } catch (cause) {
        pushToast({ title: "字典数据加载失败", message: apiMessage(cause), tone: "danger" });
        setItems([]);
      } finally {
        setItemsLoading(false);
      }
    },
    [pushToast],
  );

  const openCreateType = () => {
    setEditingType(null);
    setTypeForm(EMPTY_TYPE);
    setTypeError(null);
    setTypeDialog(true);
  };

  const openEditType = (type: DictionaryType) => {
    setEditingType(type);
    setTypeForm({
      code: type.code,
      name: type.name,
      description: type.description ?? "",
      status: type.status,
    });
    setTypeError(null);
    setTypeDialog(true);
  };

  const saveType = async () => {
    if (!typeForm.name.trim() || !typeForm.code.trim()) {
      setTypeError("字典名称和编码必填。");
      return;
    }
    setTypeSaving(true);
    setTypeError(null);
    const input: DictionaryTypeInput = {
      code: typeForm.code.trim(),
      name: typeForm.name.trim(),
      description: typeForm.description.trim() || undefined,
      status: typeForm.status,
    };
    try {
      if (editingType) {
        await updateDictionary(editingType.id, input);
        pushToast({ title: "字典类型已更新", tone: "ok" });
      } else {
        await createDictionary(input);
        pushToast({ title: "字典类型已创建", tone: "ok" });
      }
      setTypeDialog(false);
      await load();
    } catch (cause) {
      setTypeError(apiMessage(cause));
    } finally {
      setTypeSaving(false);
    }
  };

  const removeType = async () => {
    if (!deleteType) return;
    setActionLoading(true);
    try {
      await deleteDictionary(deleteType.id);
      pushToast({ title: "字典类型已删除", tone: "ok" });
      setDeleteType(null);
      if (activeType?.id === deleteType.id) setActiveType(null);
      await load();
    } catch (cause) {
      pushToast({ title: "字典类型删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const openCreateItem = () => {
    setEditingItem(null);
    setItemForm(EMPTY_ITEM);
    setItemError(null);
    setItemDialog(true);
  };

  const openEditItem = (item: DictionaryItem) => {
    setEditingItem(item);
    setItemForm({
      value: item.value,
      label: item.label,
      sort: item.sort,
      tagStyle: item.tagStyle ?? "default",
      isDefault: item.isDefault,
      status: item.status,
    });
    setItemError(null);
    setItemDialog(true);
  };

  const saveItem = async () => {
    if (!activeType) return;
    if (!itemForm.value.trim() || !itemForm.label.trim()) {
      setItemError("数据值和显示标签必填。");
      return;
    }
    setItemSaving(true);
    setItemError(null);
    const input: DictionaryItemInput = {
      typeId: activeType.id,
      value: itemForm.value.trim(),
      label: itemForm.label.trim(),
      sort: itemForm.sort,
      tagStyle: itemForm.tagStyle,
      isDefault: itemForm.isDefault,
      status: itemForm.status,
    };
    try {
      if (editingItem) {
        await updateDictionaryItem(editingItem.id, input);
        pushToast({ title: "字典数据已更新", tone: "ok" });
      } else {
        await createDictionaryItem(input);
        pushToast({ title: "字典数据已创建", tone: "ok" });
      }
      setItemDialog(false);
      await loadItems(activeType);
      await load();
    } catch (cause) {
      setItemError(apiMessage(cause));
    } finally {
      setItemSaving(false);
    }
  };

  const removeItem = async () => {
    if (!deleteItem || !activeType) return;
    setActionLoading(true);
    try {
      await deleteDictionaryItem(deleteItem.id);
      pushToast({ title: "字典数据已删除", tone: "ok" });
      setDeleteItem(null);
      await loadItems(activeType);
      await load();
    } catch (cause) {
      pushToast({ title: "字典数据删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const columns: TableColumn<DictionaryType>[] = [
    {
      key: "name",
      header: "字典类型",
      render: (type) => (
        <button type="button" className="cell-main link-btn" onClick={() => void loadItems(type)}>
          <span className="row-ico m-groove">
            <BookOpen size={15} />
          </span>
          <span>
            <span className="cell-title">{type.name}</span>
            <span className="cell-sub mono">{type.code}</span>
          </span>
        </button>
      ),
    },
    {
      key: "description",
      header: "说明",
      render: (type) => <span className="muted">{type.description || "—"}</span>,
    },
    {
      key: "count",
      header: "数据项",
      align: "right",
      render: (type) => <span className="tnum">{type.itemCount}</span>,
    },
    { key: "status", header: "状态", render: (type) => <StatusTag status={type.status} /> },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (type) => (
        <div className="row-actions">
          {canManage ? (
            <>
              <IconButton
                label="编辑字典类型"
                icon="edit"
                small
                onClick={() => openEditType(type)}
              />
              <IconButton
                label="删除字典类型"
                icon="trash"
                small
                danger
                onClick={() => setDeleteType(type)}
              />
            </>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="字典管理"
      description="字典类型与数据项使用真实 Go API；状态、默认项、色调和排序均持久化。"
      actions={
        <>
          <Tabs
            tabs={[
              { value: "swatch", label: "样本册" },
              { value: "table", label: "表格" },
            ]}
            value={view}
            onChange={(value) => setView(value as "swatch" | "table")}
          />
          {canManage ? (
            <Button variant="primary" icon="plus" onClick={openCreateType}>
              新增字典类型
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
            void load();
          }}
        >
          <Field label="关键词" wide>
            <TextInput
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="字典名称 / 编码 / 说明"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
        </SearchForm>
        {view === "table" ? (
          <Card className="reveal">
            <div className="panel-head">
              <span className="section-label">字典类型</span>
              <span className="panel-meta">{types.length} 个类型</span>
            </div>
            <DataState
              loading={loading}
              error={error}
              permissionDenied={permissionDenied}
              empty={!loading && !error && !filtered.length}
              emptyTitle="暂无字典类型"
              onRetry={() => void load()}
            >
              <DataTable columns={columns} rows={filtered} rowKey={(type) => type.id} />
            </DataState>
          </Card>
        ) : (
          <div className="dict-swatch-grid reveal">
            {loading ? <EmptyState title="正在加载字典样本册" icon="dict" /> : null}
            {!loading && !filtered.length ? <EmptyState title="暂无字典类型" icon="dict" /> : null}
            {filtered.map((type) => (
              <button
                type="button"
                className={`dict-swatch cat-${swatchTone(type.code) === "lime" ? 1 : swatchTone(type.code) === "blue" ? 2 : swatchTone(type.code) === "red" ? 3 : swatchTone(type.code) === "amber" ? 4 : 5}`}
                key={type.id}
                onClick={() => void loadItems(type)}
              >
                <span className="dict-swatch-top">
                  <span className="dict-swatch-code">{type.code}</span>
                  <StatusTag status={type.status} />
                </span>
                <strong>{type.name}</strong>
                <span className="dict-swatch-desc">{type.description || "未填写说明"}</span>
                <span className="dict-swatch-foot">
                  <span className="flag-row">
                    {Array.from(
                      { length: Math.min(5, Math.max(1, type.itemCount)) },
                      (_, index) => (
                        <i key={index} className="dict-chip" />
                      ),
                    )}
                  </span>
                  <span>{type.itemCount} 项</span>
                </span>
              </button>
            ))}
          </div>
        )}
      </div>

      <Dialog
        open={typeDialog}
        title={editingType ? "编辑字典类型" : "新增字典类型"}
        onClose={() => !typeSaving && setTypeDialog(false)}
        width={520}
        footer={
          <>
            <Button variant="ghost" disabled={typeSaving} onClick={() => setTypeDialog(false)}>
              取消
            </Button>
            <Button variant="primary" loading={typeSaving} onClick={() => void saveType()}>
              保存
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label="字典名称" required>
            <TextInput
              value={typeForm.name}
              onChange={(event) => setTypeForm({ ...typeForm, name: event.target.value })}
            />
          </Field>
          <Field label="字典编码" required>
            <TextInput
              value={typeForm.code}
              onChange={(event) => setTypeForm({ ...typeForm, code: event.target.value })}
            />
          </Field>
          <Field label="状态">
            <Select
              value={typeForm.status}
              onChange={(value) =>
                setTypeForm({ ...typeForm, status: value as "active" | "disabled" })
              }
              options={[
                { value: "active", label: "正常" },
                { value: "disabled", label: "停用" },
              ]}
            />
          </Field>
          <Field label="说明" wide>
            <TextInput
              multiline
              rows={3}
              value={typeForm.description}
              onChange={(event) => setTypeForm({ ...typeForm, description: event.target.value })}
            />
          </Field>
        </div>
        {typeError ? (
          <p className="field-error" style={{ marginTop: 14 }}>
            {typeError}
          </p>
        ) : null}
      </Dialog>

      <Drawer
        open={Boolean(activeType)}
        title={`字典数据 · ${activeType?.name ?? ""}`}
        description={activeType?.code}
        onClose={() => setActiveType(null)}
        width={620}
        footer={
          canManage ? (
            <Button variant="primary" icon="plus" onClick={openCreateItem}>
              新增数据项
            </Button>
          ) : undefined
        }
      >
        <div className="stack">
          <div className="drawer-toolbar">
            <span className="section-label">数据项</span>
            <span className="faint">{items.length} 项</span>
          </div>
          {itemsLoading ? <EmptyState title="正在加载字典数据" icon="dict" /> : null}
          {!itemsLoading && !items.length ? <EmptyState title="暂无字典数据" icon="dict" /> : null}
          {!itemsLoading
            ? items.map((item) => (
                <Card className="reveal" key={item.id}>
                  <div className="panel-head">
                    <div className="cell-main">
                      <span className={`dict-chip is-large ${item.tagStyle || ""}`}>
                        <Palette size={14} />
                      </span>
                      <div>
                        <div className="cell-title">{item.label}</div>
                        <div className="cell-sub mono">{item.value}</div>
                      </div>
                    </div>
                    <div className="row-actions">
                      {item.isDefault ? <Tag tone="ok">默认</Tag> : null}
                      <StatusTag status={item.status} />
                      {canManage ? (
                        <>
                          <IconButton
                            label="编辑数据项"
                            icon="edit"
                            small
                            onClick={() => openEditItem(item)}
                          />
                          <IconButton
                            label="删除数据项"
                            icon="trash"
                            small
                            danger
                            onClick={() => setDeleteItem(item)}
                          />
                        </>
                      ) : null}
                    </div>
                  </div>
                  <div className="panel-body faint">
                    排序 {item.sort} · 色调 {item.tagStyle || "default"}
                  </div>
                </Card>
              ))
            : null}
        </div>
      </Drawer>

      <Dialog
        open={itemDialog}
        title={editingItem ? "编辑字典数据" : "新增字典数据"}
        onClose={() => !itemSaving && setItemDialog(false)}
        width={520}
        footer={
          <>
            <Button variant="ghost" disabled={itemSaving} onClick={() => setItemDialog(false)}>
              取消
            </Button>
            <Button variant="primary" loading={itemSaving} onClick={() => void saveItem()}>
              保存
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label="显示标签" required>
            <TextInput
              value={itemForm.label}
              onChange={(event) => setItemForm({ ...itemForm, label: event.target.value })}
            />
          </Field>
          <Field label="数据值" required>
            <TextInput
              value={itemForm.value}
              onChange={(event) => setItemForm({ ...itemForm, value: event.target.value })}
            />
          </Field>
          <Field label="排序">
            <TextInput
              type="number"
              value={itemForm.sort}
              onChange={(event) => setItemForm({ ...itemForm, sort: Number(event.target.value) })}
            />
          </Field>
          <Field label="色调">
            <Select
              value={itemForm.tagStyle}
              onChange={(value) => setItemForm({ ...itemForm, tagStyle: value })}
              options={[
                { value: "default", label: "默认" },
                { value: "lime", label: "青柠" },
                { value: "blue", label: "蓝色" },
                { value: "red", label: "红色" },
                { value: "amber", label: "琥珀" },
                { value: "violet", label: "紫色" },
              ]}
            />
          </Field>
          <Field label="状态">
            <Select
              value={itemForm.status}
              onChange={(value) =>
                setItemForm({ ...itemForm, status: value as "active" | "disabled" })
              }
              options={[
                { value: "active", label: "正常" },
                { value: "disabled", label: "停用" },
              ]}
            />
          </Field>
          <Field label="默认项">
            <Switch
              checked={itemForm.isDefault}
              label={itemForm.isDefault ? "是默认项" : "不是默认项"}
              onChange={(isDefault) => setItemForm({ ...itemForm, isDefault })}
            />
          </Field>
        </div>
        {itemError ? (
          <p className="field-error" style={{ marginTop: 14 }}>
            {itemError}
          </p>
        ) : null}
      </Dialog>

      <ConfirmDialog
        open={Boolean(deleteType)}
        title="删除字典类型"
        message={`确认删除字典类型「${deleteType?.name ?? ""}」及其数据项吗？`}
        confirmText="删除"
        danger
        loading={actionLoading}
        onConfirm={() => void removeType()}
        onClose={() => setDeleteType(null)}
      />
      <ConfirmDialog
        open={Boolean(deleteItem)}
        title="删除字典数据"
        message={`确认删除「${deleteItem?.label ?? ""}」吗？`}
        confirmText="删除"
        danger
        loading={actionLoading}
        onConfirm={() => void removeItem()}
        onClose={() => setDeleteItem(null)}
      />
    </BusinessPage>
  );
}
