import type { CurrentUser } from "./types";

export async function getCurrentUserFromCookie(
  cookieHeader: string | undefined,
): Promise<CurrentUser | null> {
  if (!cookieHeader) return null;
  const base = (process.env.API_INTERNAL_BASE_URL ?? "http://localhost:8080").replace(/\/$/, "");
  try {
    const response = await fetch(`${base}/api/v1/auth/me`, {
      headers: { cookie: cookieHeader },
      cache: "no-store",
    });
    if (!response.ok) return null;
    return (await response.json()) as CurrentUser;
  } catch {
    return null;
  }
}
