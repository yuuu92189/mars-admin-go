import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CommandPalette } from "./command-palette";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

describe("CommandPalette", () => {
  it("closes with Escape and exposes searchable navigation", async () => {
    const onClose = vi.fn();
    render(
      <CommandPalette
        open
        groups={[
          {
            key: "system",
            label: "系统管理",
            items: [
              {
                key: "users",
                label: "用户管理",
                path: "/system/user",
                icon: "users",
              },
            ],
          },
        ]}
        onClose={onClose}
        onAppearance={vi.fn()}
      />,
    );

    expect(screen.getByRole("dialog", { name: "命令面板" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /用户管理/ })).toBeInTheDocument();

    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledOnce();
  });
});
