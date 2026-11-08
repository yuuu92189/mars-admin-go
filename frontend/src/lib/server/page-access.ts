import "server-only";

import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { getCurrentUserFromCookie } from "@/lib/api/server-auth";
import { isVisibleMenuPath } from "@/lib/navigation";
import { hasAnyPermission } from "@/lib/permissions";

export async function requirePageAccess(
  pathname: string,
  permissions: string[] = [],
): Promise<void> {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map(({ name, value }) => `${name}=${value}`)
    .join("; ");
  const result = await getCurrentUserFromCookie(cookieHeader);
  if (result.kind === "unauthorized") redirect(`/login?redirect=${encodeURIComponent(pathname)}`);
  if (result.kind === "error") throw new Error("Unable to validate the current session.");
  if (!isVisibleMenuPath(result.user, pathname)) notFound();
  if (permissions.length && !hasAnyPermission(result.user, permissions)) {
    redirect("/forbidden");
  }
}
