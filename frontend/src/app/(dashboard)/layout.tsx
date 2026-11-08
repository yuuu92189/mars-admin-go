import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { AppShell } from "@/components/layout/app-shell";
import { getCurrentUserFromCookie } from "@/lib/api/server-auth";

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map(({ name, value }) => `${name}=${value}`)
    .join("; ");
  const result = await getCurrentUserFromCookie(cookieHeader);
  if (result.kind === "unauthorized") {
    const requestHeaders = await headers();
    const pathname = requestHeaders.get("x-mars-pathname") ?? "/dashboard";
    redirect(`/api/v1/auth/refresh?redirect=${encodeURIComponent(pathname)}`);
  }
  if (result.kind === "error") {
    throw new Error("Unable to validate the current session.");
  }
  return <AppShell user={result.user}>{children}</AppShell>;
}
