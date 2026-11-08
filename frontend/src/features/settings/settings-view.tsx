"use client";

import { useCallback, useMemo, useState } from "react";
import { Eye, EyeOff, Save, Settings2, ShieldAlert, Undo2 } from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import {
  BusinessPage,
  DataState,
  SearchForm,
  StatusTag,
} from "@/components/business/business-page";
import { Button, IconButton } from "@/components/ui/button";
import { Field, Switch, TextInput } from "@/components/ui/input";
import { Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import { getSettingGroup, listSettingGroups, updateSettingGroup } from "@/lib/api/system";
import { hasPermission } from "@/lib/permissions";
import type { Setting, SettingGroup } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function displayValue(setting: Setting): unknown {
  if (setting.sensitive) return "";
  return setting.value;
}

function valuesEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  try {
    return JSON.stringify(a) === JSON.stringify(b);
  } catch {
    return false;
  }
}

function settingValueText(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value, null, 2);
}

export function SettingsView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "settings:manage");
  const [groups, setGroups] = useState<SettingGroup[]>([]);
  const [activeCode, setActiveCode] = useState<string>("");
  const [group, setGroup] = useState<SettingGroup | null>(null);
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [permissionDenied, setPermissionDenied] = useState(false);
  const [query, setQuery] = useState("");
  const [saving, setSaving] = useState(false);
  const [visibleSensitive, setVisibleSensitive] = useState<Set<string>>(new Set());

  const loadGroups = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      const result = await listSettingGroups();
      setGroups(result);
      setActiveCode((current) => current || result[0]?.code || "");
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadGroup = useCallback(async (code: string) => {
    if (!code) return;
    setLoading(true);
    setError(null);
    try {
      const result = await getSettingGroup(code);
      setGroup(result);
      setValues(
        Object.fromEntries(
          (result.settings ?? []).map((setting) => [setting.key, displayValue(setting)]),
        ),
      );
      setVisibleSensitive(new Set());
    } catch (cause) {
      setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useDeferredEffect(loadGroups);
  const loadActiveGroup = useCallback(() => {
    if (activeCode) return loadGroup(activeCode);
  }, [activeCode, loadGroup]);
  useDeferredEffect(loadActiveGroup);

  const settings = useMemo(() => {
    const all = group?.settings ?? [];
    const normalized = query.trim().toLowerCase();
    if (!normalized) return all;
    return all.filter((setting) =>
      [setting.key, setting.valueType]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(normalized)),
    );
  }, [group, query]);
  const original = useMemo(
    () =>
      Object.fromEntries(
        (group?.settings ?? []).map((setting) => [setting.key, displayValue(setting)]),
      ),
    [group],
  );
  const dirtyKeys = useMemo(
    () => Object.keys(values).filter((key) => !valuesEqual(values[key], original[key])),
    [original, values],
  );

  const setValue = (key: string, value: unknown) => {
    setValues((current) => ({ ...current, [key]: value }));
  };

  const reset = () => setValues(original);

  const save = async () => {
    if (!group || !dirtyKeys.length) return;
    const payload: Record<string, unknown> = {};
    for (const key of dirtyKeys) {
      const value = values[key];
      const setting = group.settings?.find((item) => item.key === key);
      if (setting?.sensitive && typeof value === "string" && !value.trim()) continue;
      payload[key] = value;
    }
    if (!Object.keys(payload).length) {
      pushToast({ title: "没有可保存的改动", tone: "info" });
      return;
    }
    setSaving(true);
    try {
      const updated = await updateSettingGroup(group.code, payload);
      setGroup(updated);
      setValues(
        Object.fromEntries(
          (updated.settings ?? []).map((setting) => [setting.key, displayValue(setting)]),
        ),
      );
      setVisibleSensitive(new Set());
      pushToast({ title: "系统配置已保存", message: "缓存已由后端失效", tone: "ok" });
      await loadGroups();
    } catch (cause) {
      pushToast({ title: "系统配置保存失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setSaving(false);
    }
  };

  const renderControl = (setting: Setting) => {
    const value = values[setting.key];
    if (setting.sensitive) {
      const visible = visibleSensitive.has(setting.key);
      return (
        <div className="settings-secret">
          <TextInput
            type={visible ? "text" : "password"}
            value={String(value ?? "")}
            placeholder="留空表示保持原值"
            autoComplete="new-password"
            onChange={(event) => setValue(setting.key, event.target.value)}
          />
          <IconButton
            label={visible ? "隐藏敏感值" : "显示输入内容"}
            icon={visible ? "eyeOff" : "eye"}
            small
            onClick={() => {
              const next = new Set(visibleSensitive);
              if (visible) next.delete(setting.key);
              else next.add(setting.key);
              setVisibleSensitive(next);
            }}
          />
        </div>
      );
    }
    if (setting.valueType === "boolean") {
      return (
        <Switch
          checked={Boolean(value)}
          label={Boolean(value) ? "已启用" : "已停用"}
          disabled={!canManage}
          onChange={(checked) => setValue(setting.key, checked)}
        />
      );
    }
    if (setting.valueType === "integer") {
      return (
        <TextInput
          type="number"
          className="settings-number"
          disabled={!canManage}
          value={Number(value ?? 0)}
          onChange={(event) => setValue(setting.key, Number(event.target.value))}
        />
      );
    }
    return (
      <TextInput
        disabled={!canManage}
        value={settingValueText(value)}
        onChange={(event) => {
          const next = event.target.value;
          if (setting.valueType === "json") {
            try {
              setValue(setting.key, JSON.parse(next));
            } catch {
              setValue(setting.key, next);
            }
          } else {
            setValue(setting.key, next);
          }
        }}
      />
    );
  };

  return (
    <BusinessPage
      title="系统配置"
      description="分组读取、逐项编辑、批量差异保存；敏感配置永不回显到浏览器。"
      actions={
        <Tag tone="info">
          <Settings2 size={13} /> Go Settings
        </Tag>
      }
    >
      <SearchForm
        loading={loading}
        onSearch={() => activeCode && void loadGroup(activeCode)}
        onReset={() => {
          setQuery("");
          if (activeCode) void loadGroup(activeCode);
        }}
      >
        <Field label="筛选配置项" wide>
          <TextInput
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="配置键 / 值类型"
            icon="search"
            clearable
            onClear={() => setQuery("")}
          />
        </Field>
      </SearchForm>
      <div className="settings">
        <aside className="settings-nav m-float reveal">
          <span className="section-label">配置分组</span>
          <nav aria-label="系统配置分组">
            {groups.map((item) => (
              <button
                type="button"
                className={`settings-nav-item ${activeCode === item.code ? "is-active" : ""}`}
                key={item.code}
                onClick={() => setActiveCode(item.code)}
              >
                <Settings2 size={15} />
                <span>{item.name}</span>
                <b>{item.settingCount ?? 0}</b>
              </button>
            ))}
          </nav>
          <div className="settings-nav-note faint">
            <ShieldAlert size={13} /> 敏感项仅可覆盖，不可回显
          </div>
        </aside>
        <div className="settings-body">
          <DataState
            loading={loading}
            error={error}
            permissionDenied={permissionDenied}
            empty={!loading && !error && !(group?.settings ?? []).length}
            emptyTitle="该分组没有配置项"
            onRetry={() => activeCode && void loadGroup(activeCode)}
          >
            {group ? (
              <Card className="settings-group reveal">
                <div className="settings-group-head">
                  <div>
                    <span className="section-label">{group.name}</span>
                    <div className="faint" style={{ marginTop: 4 }}>
                      {group.description || group.code} · {group.settings?.length ?? 0} 项
                    </div>
                  </div>
                  <StatusTag status={group.status} />
                </div>
                {settings.map((setting) => {
                  const dirty = dirtyKeys.includes(setting.key);
                  return (
                    <div className={`settings-row ${dirty ? "is-dirty" : ""}`} key={setting.key}>
                      <div className="settings-label">
                        <span className="settings-title">
                          {setting.key}
                          {setting.sensitive ? <Tag tone="danger">敏感</Tag> : null}
                        </span>
                        <span className="settings-remark">
                          类型 {setting.valueType} · 版本 {setting.version}
                        </span>
                      </div>
                      <div className="settings-control">{renderControl(setting)}</div>
                      {dirty ? (
                        <div className="settings-was">
                          <span>原值</span>
                          <s>
                            {setting.sensitive
                              ? "已隐藏"
                              : settingValueText(original[setting.key]) || "空"}
                          </s>
                          <button
                            type="button"
                            className="settings-undo"
                            onClick={() => setValue(setting.key, original[setting.key])}
                          >
                            撤销
                          </button>
                        </div>
                      ) : null}
                    </div>
                  );
                })}
                {!settings.length && !loading ? (
                  <EmptyState title="没有匹配的配置项" icon="settings" />
                ) : null}
              </Card>
            ) : null}
          </DataState>
        </div>
      </div>

      {dirtyKeys.length && canManage ? (
        <div className="float-bar settings-bar m-float" role="status">
          <div className="settings-diff">
            {dirtyKeys.slice(0, 6).map((key) => (
              <div className="settings-diff-row" key={key}>
                <span className="settings-diff-name">{key}</span>
                <span className="settings-diff-from">
                  {group?.settings?.find((item) => item.key === key)?.sensitive
                    ? "已隐藏"
                    : settingValueText(original[key]) || "空"}
                </span>
                <span>→</span>
                <span className="settings-diff-to">
                  {group?.settings?.find((item) => item.key === key)?.sensitive
                    ? "新值已填写"
                    : settingValueText(values[key]) || "空"}
                </span>
              </div>
            ))}
          </div>
          <span className="float-bar-count">
            <b>{dirtyKeys.length}</b> 项待保存
          </span>
          <Button variant="text" icon="refresh" className="float-bar-clear" onClick={reset}>
            撤销全部
          </Button>
          <Button variant="primary" icon="save" loading={saving} onClick={() => void save()}>
            保存改动
          </Button>
        </div>
      ) : null}
    </BusinessPage>
  );
}
