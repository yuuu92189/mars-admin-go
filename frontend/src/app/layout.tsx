import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Providers } from "./providers";
import { themeInitScript } from "@/components/providers/theme-provider";
import { APP } from "@/lib/config";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: APP.name,
    template: `%s · ${APP.name}`,
  },
  description: "Mars Admin Go Edition operations console",
};

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body>
        <Providers>
          <div className="app-root">{children}</div>
          <div className="grain-layer" aria-hidden="true" />
        </Providers>
      </body>
    </html>
  );
}
