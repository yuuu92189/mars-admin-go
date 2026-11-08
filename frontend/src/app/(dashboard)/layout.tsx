import { cookies } from "next/headers";
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
  const user = await getCurrentUserFromCookie(cookieHeader);
  if (!user) redirect("/login?redirect=%2Fdashboard");
  return <AppShell user={user}>{children}</AppShell>;
}
