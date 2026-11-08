import { apiRequest, resetUnauthorizedNotification, setSessionRefreshEnabled } from "./client";
import type {
  CaptchaResponse,
  ChangePasswordRequest,
  CurrentUser,
  LoginRequest,
  LoginResult,
} from "./types";

export function issueCaptcha(): Promise<CaptchaResponse> {
  return apiRequest<CaptchaResponse>("/api/v1/auth/captcha?type=image");
}

export function getPublicSettings(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>("/api/v1/admin/settings/public");
}

export async function login(input: LoginRequest): Promise<LoginResult> {
  const result = await apiRequest<LoginResult>("/api/v1/auth/login", {
    method: "POST",
    body: input as Record<string, unknown>,
    skipAuthRefresh: true,
  });
  setSessionRefreshEnabled(true);
  resetUnauthorizedNotification();
  return result;
}

export function getCurrentUser(): Promise<CurrentUser> {
  return apiRequest<CurrentUser>("/api/v1/auth/me");
}

export async function logout(): Promise<void> {
  setSessionRefreshEnabled(false);
  try {
    await apiRequest<void>("/api/v1/auth/logout", {
      method: "POST",
      skipAuthRefresh: true,
    });
  } finally {
    setSessionRefreshEnabled(false);
  }
}

export async function changePassword(input: ChangePasswordRequest): Promise<void> {
  await apiRequest<void>("/api/v1/auth/password", {
    method: "POST",
    body: input,
  });
  setSessionRefreshEnabled(false);
}
