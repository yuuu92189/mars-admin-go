import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { DataState, StatusTag, TreeNav } from "./business-page";

type Node = { id: string; name: string; children?: Node[] };

describe("Phase 3 business UI", () => {
  it("maps canonical statuses to readable tags", () => {
    const { rerender } = render(<StatusTag status="active" />);
    expect(screen.getByText("正常")).toBeInTheDocument();
    rerender(<StatusTag status="disabled" />);
    expect(screen.getByText("停用")).toBeInTheDocument();
    rerender(<StatusTag status="published" />);
    expect(screen.getByText("已发布")).toBeInTheDocument();
  });

  it("shows permission and error states instead of business data", () => {
    const { rerender } = render(
      <DataState permissionDenied>
        <span>secret</span>
      </DataState>,
    );
    expect(screen.getByText("当前账号没有访问权限")).toBeInTheDocument();
    expect(screen.queryByText("secret")).not.toBeInTheDocument();

    rerender(
      <DataState error="Network unavailable">
        <span>secret</span>
      </DataState>,
    );
    expect(screen.getByText("数据加载失败")).toBeInTheDocument();
    expect(screen.getByText("Network unavailable")).toBeInTheDocument();
  });

  it("selects root and nested tree nodes", async () => {
    const onSelect = vi.fn();
    const nodes: Node[] = [
      {
        id: "root",
        name: "总部",
        children: [{ id: "child", name: "研发部" }],
      },
    ];
    render(
      <TreeNav
        items={nodes}
        getId={(item) => item.id}
        getLabel={(item) => item.name}
        getChildren={(item) => item.children}
        onSelect={onSelect}
        rootLabel="全部部门"
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: /全部部门/ }));
    await userEvent.click(screen.getByRole("button", { name: /研发部/ }));
    expect(onSelect).toHaveBeenNthCalledWith(1, null);
    expect(onSelect).toHaveBeenNthCalledWith(2, "child");
  });
});
