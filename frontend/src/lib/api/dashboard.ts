import { apiRequest } from "./client";
import type { Dashboard } from "./types";

export type DashboardRange = "24h" | "7d" | "30d";

export function getDashboard(range: DashboardRange = "24h"): Promise<Dashboard> {
  const params = new URLSearchParams({ range });
  return apiRequest<Dashboard>(`/api/v1/admin/monitoring/dashboard?${params.toString()}`);
}

export function getDashboardCounters(): Promise<Dashboard> {
  return getDashboard();
}
