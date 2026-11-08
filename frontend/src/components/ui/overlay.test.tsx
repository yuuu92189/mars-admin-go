import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Dialog } from "./overlay";

describe("Dialog", () => {
  it("renders a modal and closes with Escape", async () => {
    const onClose = vi.fn();
    render(
      <Dialog open title="确认操作" onClose={onClose}>
        <button type="button" data-autofocus>
          确认
        </button>
      </Dialog>,
    );
    expect(screen.getByRole("dialog", { name: "确认操作" })).toBeInTheDocument();
    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalled();
  });
});
