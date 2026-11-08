import { apiRequest } from "./client";
import type { DashboardCounters } from "./types";

export function getDashboardCounters(): Promise<DashboardCounters> {
  return apiRequest<DashboardCounters>("/api/v1/admin/monitoring/dashboard");
}
