import { apiRequest } from "./client";
import type { CaptchaResponse, CurrentUser, LoginRequest, LoginResult } from "./types";

export function issueCaptcha(): Promise<CaptchaResponse> {
  return apiRequest<CaptchaResponse>("/api/v1/auth/captcha?type=image");
}

export function login(input: LoginRequest): Promise<LoginResult> {
  return apiRequest<LoginResult>("/api/v1/auth/login", {
    method: "POST",
    body: input as Record<string, unknown>,
  });
}

export function getCurrentUser(): Promise<CurrentUser> {
  return apiRequest<CurrentUser>("/api/v1/auth/me");
}

export function logout(): Promise<void> {
  return apiRequest<void>("/api/v1/auth/logout", { method: "POST" });
}
