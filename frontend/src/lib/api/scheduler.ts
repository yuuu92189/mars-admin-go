import { apiRequest } from "./client";
import type {
  ScheduledJob,
  ScheduledJobInput,
  ScheduledJobPage,
  ScheduledJobRun,
  ScheduledJobRunPage,
} from "./types";

export type JobQuery = {
  page?: number;
  pageSize?: number;
  status?: string;
};

function jobQueryString(query: JobQuery): string {
  const params = new URLSearchParams();
  params.set("page", String(query.page ?? 1));
  params.set("pageSize", String(query.pageSize ?? 20));
  if (query.status) params.set("status", query.status);
  return params.toString();
}

export function listScheduledJobs(query: JobQuery = {}): Promise<ScheduledJobPage> {
  return apiRequest<ScheduledJobPage>(`/api/v1/admin/scheduler/jobs?${jobQueryString(query)}`);
}

export function createScheduledJob(input: ScheduledJobInput): Promise<ScheduledJob> {
  return apiRequest<ScheduledJob>("/api/v1/admin/scheduler/jobs", {
    method: "POST",
    body: input,
  });
}

export function updateScheduledJob(id: string, input: ScheduledJobInput): Promise<ScheduledJob> {
  return apiRequest<ScheduledJob>(`/api/v1/admin/scheduler/jobs/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteScheduledJob(id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/scheduler/jobs/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function pauseScheduledJob(id: string): Promise<ScheduledJob> {
  return apiRequest<ScheduledJob>(`/api/v1/admin/scheduler/jobs/${encodeURIComponent(id)}/pause`, {
    method: "POST",
  });
}

export function resumeScheduledJob(id: string): Promise<ScheduledJob> {
  return apiRequest<ScheduledJob>(`/api/v1/admin/scheduler/jobs/${encodeURIComponent(id)}/resume`, {
    method: "POST",
  });
}

export function runScheduledJob(id: string): Promise<ScheduledJobRun> {
  return apiRequest<ScheduledJobRun>(`/api/v1/admin/scheduler/jobs/${encodeURIComponent(id)}/run`, {
    method: "POST",
  });
}

export function listScheduledJobRuns(
  id: string,
  page = 1,
  pageSize = 20,
): Promise<ScheduledJobRunPage> {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  return apiRequest<ScheduledJobRunPage>(
    `/api/v1/admin/scheduler/jobs/${encodeURIComponent(id)}/runs?${params.toString()}`,
  );
}
