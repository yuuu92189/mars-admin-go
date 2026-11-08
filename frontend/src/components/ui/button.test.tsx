import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Button, IconButton } from "./button";

describe("Button", () => {
  it("renders and handles a click", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>保存</Button>);
    await userEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("exposes loading and disabled state", () => {
    render(<Button loading>保存</Button>);
    expect(screen.getByRole("button", { name: /保存/ })).toBeDisabled();
  });
});

describe("IconButton", () => {
  it("requires an accessible label", () => {
    render(<IconButton icon="refresh" label="刷新" />);
    expect(screen.getByRole("button", { name: "刷新" })).toBeInTheDocument();
  });
});
