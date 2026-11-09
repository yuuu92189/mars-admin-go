import type { ReactNode } from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/components/providers/toast-provider";
import { LoginScreen } from "./login-screen";
import { RegisterScreen } from "./register-screen";

const mocks = vi.hoisted(() => ({
  getPublicSettings: vi.fn(),
  issueCaptcha: vi.fn(),
  replace: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mocks.replace }),
}));

vi.mock("next/link", () => ({
  default: ({ href, children }: { href: string; children: ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock("@/lib/motion", () => ({
  EASE: { premium: "none", glide: "none", settle: "none" },
  prefersReducedMotion: () => true,
}));

vi.mock("@/lib/api/auth", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api/auth")>();
  return {
    ...actual,
    getPublicSettings: mocks.getPublicSettings,
    issueCaptcha: mocks.issueCaptcha,
  };
});

describe("registration policy", () => {
  beforeEach(() => {
    localStorage.clear();
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
    mocks.getPublicSettings.mockReset();
    mocks.issueCaptcha.mockReset();
    mocks.replace.mockReset();
    mocks.getPublicSettings.mockResolvedValue({
      "site.site.registration_enabled": true,
      "security.auth.captcha_enabled": false,
    });
  });

  it("shows the registration entry on login when registration is enabled", async () => {
    render(
      <ToastProvider>
        <LoginScreen />
      </ToastProvider>,
    );

    expect(await screen.findByRole("link", { name: "没有账号？立即注册" })).toHaveAttribute(
      "href",
      "/register",
    );
  });

  it("shows the registration form when registration is enabled", async () => {
    render(
      <ToastProvider>
        <RegisterScreen />
      </ToastProvider>,
    );

    await waitFor(() => {
      expect(screen.getByRole("textbox", { name: /用户名/ })).toBeVisible();
    });
    expect(screen.getByRole("textbox", { name: /昵称/ })).toBeVisible();
    expect(screen.getByRole("button", { name: /^注册$/ })).toBeVisible();
    expect(screen.queryByText("系统暂未开放注册")).not.toBeInTheDocument();
  });
});
