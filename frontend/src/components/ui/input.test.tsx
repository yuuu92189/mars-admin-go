import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Checkbox, Field, TextInput } from "./input";

describe("form primitives", () => {
  it("renders a labeled field and forwards input changes", async () => {
    const onChange = vi.fn();
    render(
      <Field label="账号">
        <TextInput aria-label="账号输入" value="" onChange={onChange} />
      </Field>,
    );
    await userEvent.type(screen.getByRole("textbox", { name: "账号输入" }), "admin");
    expect(onChange).toHaveBeenCalled();
  });

  it("toggles a checkbox", async () => {
    const onChange = vi.fn();
    render(<Checkbox checked={false} onChange={onChange} label="记住账号" />);
    await userEvent.click(screen.getByRole("checkbox", { name: "记住账号" }));
    expect(onChange).toHaveBeenCalledWith(true);
  });
});
