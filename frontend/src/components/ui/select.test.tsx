import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Select } from "./select";

describe("Select", () => {
  it("opens and selects an option", async () => {
    const onChange = vi.fn();
    render(
      <Select
        value="active"
        onChange={onChange}
        options={[
          { value: "active", label: "启用" },
          { value: "disabled", label: "停用" },
        ]}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "启用" }));
    await userEvent.click(screen.getByRole("option", { name: "停用" }));
    expect(onChange).toHaveBeenCalledWith("disabled");
  });
});
