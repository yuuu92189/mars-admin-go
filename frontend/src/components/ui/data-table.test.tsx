import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { DataTable, Pagination } from "./data-table";

type Row = { id: string; name: string };

describe("DataTable", () => {
  it("renders rows and emits selection", async () => {
    const onSelectionChange = vi.fn();
    render(
      <DataTable
        rows={[{ id: "1", name: "管理员" }]}
        rowKey={(row) => row.id}
        selectable
        onSelectionChange={onSelectionChange}
        columns={[{ key: "name", header: "名称", render: (row) => row.name }]}
      />,
    );
    await userEvent.click(screen.getAllByRole("checkbox")[0]!);
    expect(onSelectionChange).toHaveBeenCalled();
    expect(screen.getByText("管理员")).toBeInTheDocument();
  });
});

describe("Pagination", () => {
  it("moves to the next page", async () => {
    const onChange = vi.fn();
    render(<Pagination page={1} pageSize={20} total={45} onChange={onChange} />);
    await userEvent.click(screen.getByRole("button", { name: "下一页" }));
    expect(onChange).toHaveBeenCalledWith(2);
  });
});
