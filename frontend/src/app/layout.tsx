import type { Metadata } from "next";
import { cookies } from "next/headers";
import type { ReactNode } from "react";
import { Providers } from "./providers";
import { themeInitScript } from "@/components/providers/theme-provider";
import { APP } from "@/lib/config";
import { APPEARANCE_COOKIE_NAME, parseAppearanceCookie } from "@/lib/appearance";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: APP.name,
    template: `%s · ${APP.name}`,
  },
  description: "Mars Admin Go Edition operations console",
};

export default async function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  const cookieStore = await cookies();
  const appearanceValue = cookieStore.get(APPEARANCE_COOKIE_NAME)?.value;
  const initialAppearance = parseAppearanceCookie(appearanceValue);
  const initialResolvedTheme = initialAppearance.theme === "dark" ? "dark" : "light";
  return (
    <html
      lang="zh-CN"
      data-theme={initialResolvedTheme}
      data-density={initialAppearance.density}
      data-rail-collapsed={String(initialAppearance.railCollapsed)}
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body>
        <Providers
          initialAppearance={initialAppearance}
          appearanceInitialized={Boolean(appearanceValue)}
        >
          <div className="app-root">{children}</div>
          <div className="grain-layer" aria-hidden="true" />
        </Providers>
      </body>
    </html>
  );
}
