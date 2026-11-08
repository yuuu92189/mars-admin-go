"use client";

import { useCallback, useMemo, useState } from "react";
import { CalendarClock, Pause, Play, Plus, RotateCw, Timer } from "lucide-react";
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
import { Field, TextInput } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Dialog, Drawer } from "@/components/ui/overlay";
import { Select } from "@/components/ui/select";
import { Card, EmptyState, Tag } from "@/components/ui/surface";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import {
  createScheduledJob,
  deleteScheduledJob,
  listScheduledJobRuns,
  listScheduledJobs,
  pauseScheduledJob,
  resumeScheduledJob,
  runScheduledJob,
  updateScheduledJob,
} from "@/lib/api/scheduler";
import { hasPermission } from "@/lib/permissions";
import type {
  ScheduledJob,
  ScheduledJobInput,
  ScheduledJobPage,
  ScheduledJobRun,
} from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";

const PAGE_SIZE = 15;
const SUPPORTED_HANDLERS = [
  { value: "system.health_probe", label: "系统健康探测" },
  { value: "storage.cleanup", label: "存储清理" },
];

type JobForm = {
  name: string;
  handler: string;
  cronExpression: string;
  payload: string;
  maxAttempts: number;
  timeoutSeconds: number;
  concurrencyKey: string;
};

const EMPTY_FORM: JobForm = {
  name: "",
  handler: "system.health_probe",
  cronExpression: "0 */5 * * * *",
  payload: "{}",
  maxAttempts: 3,
  timeoutSeconds: 60,
  concurrencyKey: "",
};

function apiMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

function cronText(expression: string): string {
  const parts = expression.trim().split(/\s+/);
  if (parts.length === 5) return `标准 5 段 Cron · ${expression}`;
  if (parts.length === 6) return `秒级 6 段 Cron · ${expression}`;
  return `Cron · ${expression || "未设置"}`;
}

function CronBuilder({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const [minute, setMinute] = useState("0");
  const [hour, setHour] = useState("*");
  const [day, setDay] = useState("*");
  const [month, setMonth] = useState("*");
  const [weekday, setWeekday] = useState("*");
  const apply = () => onChange(`${minute} ${hour} ${day} ${month} ${weekday}`);
  return (
    <Card>
      <div className="panel-head">
        <span className="section-label">Cron Builder</span>
        <Tag tone="info">River Scheduler</Tag>
      </div>
      <div className="panel-body stack">
        <div className="form-grid">
          <Field label="分钟">
            <Select
              value={minute}
              onChange={setMinute}
              options={[
                { value: "0", label: "每小时整点" },
                { value: "*/5", label: "每 5 分钟" },
                { value: "*/15", label: "每 15 分钟" },
                { value: "30", label: "每小时 30 分" },
              ]}
            />
          </Field>
          <Field label="小时">
            <Select
              value={hour}
              onChange={setHour}
              options={[
                { value: "*", label: "每小时" },
                { value: "0", label: "00:00" },
                { value: "2", label: "02:00" },
                { value: "9", label: "09:00" },
                { value: "18", label: "18:00" },
              ]}
            />
          </Field>
          <Field label="日期">
            <TextInput value={day} onChange={(event) => setDay(event.target.value)} />
          </Field>
          <Field label="月份">
            <TextInput value={month} onChange={(event) => setMonth(event.target.value)} />
          </Field>
          <Field label="星期">
            <TextInput value={weekday} onChange={(event) => setWeekday(event.target.value)} />
          </Field>
        </div>
        <div className="drawer-toolbar">
          <span className="code-chip">{value || "0 * * * *"}</span>
          <Button variant="secondary" onClick={apply}>
            应用到任务
          </Button>
        </div>
      </div>
    </Card>
  );
}

export function SchedulerView() {
  const { user: currentUser } = useAuth();
  const { pushToast } = useToast();
  const canManage = hasPermission(currentUser, "scheduler:manage");
  const [page, setPage] = useState<ScheduledJobPage>({
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
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<ScheduledJob | null>(null);
  const [form, setForm] = useState<JobForm>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ScheduledJob | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [runsJob, setRunsJob] = useState<ScheduledJob | null>(null);
  const [runs, setRuns] = useState<ScheduledJobRun[]>([]);
  const [runsLoading, setRunsLoading] = useState(false);
  const [runPage, setRunPage] = useState(1);
  const [runTotal, setRunTotal] = useState(0);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setPermissionDenied(false);
    try {
      setPage(await listScheduledJobs({ page: pageNumber, pageSize: PAGE_SIZE, status }));
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 403) setPermissionDenied(true);
      else setError(apiMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [pageNumber, status]);

  useDeferredEffect(load);

  const rows = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return page.items ?? [];
    return (page.items ?? []).filter((job) =>
      [job.name, job.handler, job.cronExpression]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [keyword, page.items]);

  const openCreate = () => {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (job: ScheduledJob) => {
    setEditing(job);
    setForm({
      name: job.name ?? "",
      handler: job.handler ?? "system.health_probe",
      cronExpression: job.cronExpression ?? "0 */5 * * * *",
      payload: JSON.stringify(job.payload ?? {}, null, 2),
      maxAttempts: job.maxAttempts ?? 3,
      timeoutSeconds: job.timeoutSeconds ?? 60,
      concurrencyKey: job.concurrencyKey ?? "",
    });
    setFormError(null);
    setDialogOpen(true);
  };

  const submit = async () => {
    if (!form.name.trim() || !form.handler || !form.cronExpression.trim()) {
      setFormError("任务名称、处理器和 Cron 表达式必填。");
      return;
    }
    let payload: Record<string, unknown> = {};
    try {
      payload = form.payload.trim() ? JSON.parse(form.payload) : {};
    } catch {
      setFormError("Payload 必须是合法 JSON。");
      return;
    }
    setSaving(true);
    setFormError(null);
    const input: ScheduledJobInput = {
      name: form.name.trim(),
      handler: form.handler,
      cronExpression: form.cronExpression.trim(),
      payload,
      maxAttempts: form.maxAttempts,
      timeoutSeconds: form.timeoutSeconds,
      concurrencyKey: form.concurrencyKey.trim(),
    };
    try {
      if (editing?.id) {
        await updateScheduledJob(editing.id, input);
        pushToast({ title: "定时任务已更新", tone: "ok" });
      } else {
        await createScheduledJob(input);
        pushToast({ title: "定时任务已创建", tone: "ok" });
      }
      setDialogOpen(false);
      await load();
    } catch (cause) {
      setFormError(apiMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const changeStatus = async (job: ScheduledJob, resume: boolean) => {
    if (!job.id) return;
    setActionLoading(true);
    try {
      if (resume) await resumeScheduledJob(job.id);
      else await pauseScheduledJob(job.id);
      pushToast({ title: resume ? "任务已恢复" : "任务已暂停", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "任务状态更新失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const runNow = async (job: ScheduledJob) => {
    if (!job.id) return;
    setActionLoading(true);
    try {
      await runScheduledJob(job.id);
      pushToast({ title: "任务已提交 River 队列", tone: "ok" });
      await load();
    } catch (cause) {
      pushToast({ title: "立即执行失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const remove = async () => {
    if (!deleteTarget?.id) return;
    setActionLoading(true);
    try {
      await deleteScheduledJob(deleteTarget.id);
      pushToast({ title: "定时任务已删除", tone: "ok" });
      setDeleteTarget(null);
      await load();
    } catch (cause) {
      pushToast({ title: "任务删除失败", message: apiMessage(cause), tone: "danger" });
    } finally {
      setActionLoading(false);
    }
  };

  const loadRuns = useCallback(
    async (job: ScheduledJob, pageNo = 1) => {
      if (!job.id) return;
      setRunsJob(job);
      setRunPage(pageNo);
      setRunsLoading(true);
      try {
        const result = await listScheduledJobRuns(job.id, pageNo, 20);
        setRuns(result.items);
        setRunTotal(result.total);
      } catch (cause) {
        pushToast({ title: "执行日志加载失败", message: apiMessage(cause), tone: "danger" });
      } finally {
        setRunsLoading(false);
      }
    },
    [pushToast],
  );

  const columns: TableColumn<ScheduledJob>[] = [
    {
      key: "name",
      header: "任务",
      render: (job) => (
        <div className="cell-main">
          <span className="row-ico m-groove">
            <CalendarClock size={15} />
          </span>
          <div>
            <div className="cell-title">{job.name}</div>
            <div className="cell-sub mono">{job.handler}</div>
          </div>
        </div>
      ),
    },
    {
      key: "cron",
      header: "调度表达式",
      render: (job) => <span className="code-chip">{job.cronExpression || "—"}</span>,
    },
    {
      key: "next",
      header: "下次执行",
      render: (job) => <span className="tnum faint">{formatDateTime(job.nextRunAt)}</span>,
    },
    {
      key: "attempts",
      header: "重试 / 超时",
      render: (job) => (
        <span className="tnum">
          {job.maxAttempts ?? 3} 次 / {job.timeoutSeconds ?? 60}s
        </span>
      ),
    },
    { key: "status", header: "状态", render: (job) => <StatusTag status={job.status} /> },
    {
      key: "actions",
      header: "操作",
      align: "right",
      render: (job) => (
        <div className="row-actions">
          <IconButton label="执行日志" icon="audit" small onClick={() => void loadRuns(job)} />
          {canManage ? (
            <>
              <IconButton label="编辑任务" icon="edit" small onClick={() => openEdit(job)} />
              <IconButton
                label={job.status === "active" ? "暂停任务" : "恢复任务"}
                icon={job.status === "active" ? "pause" : "play"}
                small
                onClick={() => void changeStatus(job, job.status !== "active")}
              />
              <IconButton label="立即执行" icon="refresh" small onClick={() => void runNow(job)} />
              <IconButton
                label="删除任务"
                icon="trash"
                small
                danger
                onClick={() => setDeleteTarget(job)}
              />
            </>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <BusinessPage
      title="定时任务"
      description="基于 Go + River 的真实任务模型；Cron、暂停恢复、立即执行和执行日志均调用后端。"
      actions={
        canManage ? (
          <Button variant="primary" icon="plus" onClick={openCreate}>
            新增任务
          </Button>
        ) : undefined
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
              placeholder="任务名称 / Handler / Cron"
              icon="search"
              clearable
              onClear={() => setKeyword("")}
            />
          </Field>
          <Field label="状态">
            <Select
              value={status}
              onChange={setStatus}
              options={[
                { value: "", label: "全部状态" },
                { value: "active", label: "运行中" },
                { value: "paused", label: "已暂停" },
              ]}
            />
          </Field>
        </SearchForm>
        <Card className="reveal">
          <div className="panel-head">
            <span className="section-label">任务列表</span>
            <span className="panel-meta">{page.total ?? 0} 个任务</span>
          </div>
          <DataState
            loading={loading}
            error={error}
            permissionDenied={permissionDenied}
            empty={!loading && !error && !rows.length}
            emptyTitle="暂无定时任务"
            onRetry={() => void load()}
          >
            <DataTable columns={columns} rows={rows} rowKey={(job) => job.id ?? ""} />
            <Pagination
              page={pageNumber}
              pageSize={PAGE_SIZE}
              total={page.total ?? 0}
              onChange={setPageNumber}
            />
          </DataState>
        </Card>
      </div>

      <Dialog
        open={dialogOpen}
        title={editing ? "编辑定时任务" : "新增定时任务"}
        onClose={() => !saving && setDialogOpen(false)}
        width={720}
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
          <Field label="任务名称" required>
            <TextInput
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field label="执行处理器">
            <Select
              value={form.handler}
              onChange={(value) => setForm({ ...form, handler: value })}
              options={SUPPORTED_HANDLERS}
            />
          </Field>
          <Field label="Cron 表达式" required wide hint={cronText(form.cronExpression)}>
            <TextInput
              value={form.cronExpression}
              onChange={(event) => setForm({ ...form, cronExpression: event.target.value })}
            />
          </Field>
          <Field label="最大重试次数">
            <TextInput
              type="number"
              value={form.maxAttempts}
              onChange={(event) => setForm({ ...form, maxAttempts: Number(event.target.value) })}
            />
          </Field>
          <Field label="超时秒数">
            <TextInput
              type="number"
              value={form.timeoutSeconds}
              onChange={(event) => setForm({ ...form, timeoutSeconds: Number(event.target.value) })}
            />
          </Field>
          <Field label="并发键">
            <TextInput
              value={form.concurrencyKey}
              onChange={(event) => setForm({ ...form, concurrencyKey: event.target.value })}
            />
          </Field>
          <Field label="Payload JSON" wide>
            <TextInput
              multiline
              rows={5}
              value={form.payload}
              onChange={(event) => setForm({ ...form, payload: event.target.value })}
            />
          </Field>
        </div>
        <div style={{ marginTop: 16 }}>
          <CronBuilder
            value={form.cronExpression}
            onChange={(cronExpression) => setForm({ ...form, cronExpression })}
          />
        </div>
        {formError ? (
          <p className="field-error" style={{ marginTop: 14 }}>
            {formError}
          </p>
        ) : null}
      </Dialog>

      <Drawer
        open={Boolean(runsJob)}
        title={`执行日志 · ${runsJob?.name ?? ""}`}
        description={runsJob?.handler}
        onClose={() => setRunsJob(null)}
        width={680}
      >
        <div className="stack">
          <div className="drawer-toolbar">
            <span className="section-label">River 执行记录</span>
            <Button
              icon="refresh"
              loading={runsLoading}
              onClick={() => runsJob && void loadRuns(runsJob, runPage)}
            >
              刷新
            </Button>
          </div>
          {runsLoading ? <EmptyState title="正在读取执行日志" icon="clock" /> : null}
          {!runsLoading && !runs.length ? <EmptyState title="暂无执行记录" icon="clock" /> : null}
          {!runsLoading
            ? runs.map((run) => (
                <Card className="reveal" key={run.id}>
                  <div className="panel-head">
                    <div className="cell-main">
                      <span className="row-ico m-groove">
                        <Timer size={15} />
                      </span>
                      <div>
                        <div className="cell-title">{run.id?.slice(0, 8)}</div>
                        <div className="cell-sub">尝试 {run.attempt ?? 0} 次</div>
                      </div>
                    </div>
                    <StatusTag status={run.status} />
                  </div>
                  <div className="panel-body">
                    <DetailGrid
                      items={[
                        { label: "开始", value: formatDateTime(run.startedAt) },
                        { label: "结束", value: formatDateTime(run.finishedAt) },
                        { label: "耗时", value: `${run.durationMs ?? 0} ms` },
                        { label: "River Job", value: run.riverJobId ?? "—" },
                        { label: "错误", value: run.errorMessage || "—", wide: true },
                      ]}
                    />
                  </div>
                </Card>
              ))
            : null}
          <Pagination
            page={runPage}
            pageSize={20}
            total={runTotal}
            onChange={(next) => runsJob && void loadRuns(runsJob, next)}
          />
        </div>
      </Drawer>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除定时任务"
        message={`确认删除任务「${deleteTarget?.name ?? ""}」吗？`}
        confirmText="删除"
        danger
        loading={actionLoading}
        onConfirm={() => void remove()}
        onClose={() => setDeleteTarget(null)}
      />
    </BusinessPage>
  );
}
