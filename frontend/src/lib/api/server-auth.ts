import type { CurrentUser } from "./types";
import { resolveApiBaseUrl } from "@/lib/config/api-base-url";

export type ServerAuthResult =
  { kind: "ok"; user: CurrentUser } | { kind: "unauthorized" } | { kind: "error"; status?: number };

export async function getCurrentUserFromCookie(
  cookieHeader: string | undefined,
): Promise<ServerAuthResult> {
  if (!cookieHeader) return { kind: "unauthorized" };
  try {
    const base = resolveApiBaseUrl();
    const response = await fetch(`${base}/api/v1/auth/me`, {
      headers: { cookie: cookieHeader },
      cache: "no-store",
    });
    if (response.status === 401) return { kind: "unauthorized" };
    if (!response.ok) return { kind: "error", status: response.status };
    return { kind: "ok", user: (await response.json()) as CurrentUser };
  } catch {
    return { kind: "error" };
  }
}
