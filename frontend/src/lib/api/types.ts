import type { components, operations } from "@mars/api-types";

export type { components, operations } from "@mars/api-types";

export type CurrentUser = components["schemas"]["CurrentUser"];
export type Menu = components["schemas"]["Menu"];
export type User = components["schemas"]["User"];
export type UserPage = components["schemas"]["UserPage"];
export type LoginRequest = components["schemas"]["LoginRequest"];
export type LoginResult = components["schemas"]["LoginResult"];
export type ChangePasswordRequest = components["schemas"]["ChangePassword"];
export type Problem = components["schemas"]["Problem"];
export type CaptchaResponse =
  operations["issueCaptcha"]["responses"][200]["content"]["application/json"];
export type DashboardCounters =
  operations["monitoringDashboard"]["responses"][200]["content"]["application/json"];
