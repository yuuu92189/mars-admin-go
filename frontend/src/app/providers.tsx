"use client";

import type { ReactNode } from "react";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { ToastProvider } from "@/components/providers/toast-provider";
import type { Appearance } from "@/lib/appearance";

export function Providers({
  children,
  initialAppearance,
  appearanceInitialized,
}: {
  children: ReactNode;
  initialAppearance?: Appearance;
  appearanceInitialized?: boolean;
}) {
  return (
    <ThemeProvider
      initialAppearance={initialAppearance}
      appearanceInitialized={appearanceInitialized}
    >
      <ToastProvider>{children}</ToastProvider>
    </ThemeProvider>
  );
}
