import { describe, expect, it } from "vitest";
import type { CurrentUser, Menu } from "@/lib/api/types";
import { isVisibleMenuPath } from "@/lib/navigation";
import { buildNavGroups } from "./nav";

function menu(overrides: Partial<Menu> & Pick<Menu, "id" | "name" | "type">): Menu {
  return {
    sort: 0,
    visible: true,
    isFrame: false,
    ...overrides,
  };
}

function userWithMenus(menus: Menu[]): CurrentUser {
  return {
    user: {
      id: "019b0000-0000-7000-8000-000000000001",
      username: "tester",
      displayName: "Tester",
      email: "tester@example.invalid",
      status: "active",
      userType: "admin",
      version: 1,
    },
    roles: ["admin"],
    permissions: ["user:read"],
    menus,
  };
}

describe("buildNavGroups", () => {
  it("keeps only visible menu leaves and normalizes their paths", () => {
    const groups = buildNavGroups(
      userWithMenus([
        menu({
          id: "group",
          name: "系统管理",
          type: 1,
          children: [
            menu({
              id: "users",
              name: "用户管理",
              type: 2,
              path: "system/user",
              icon: "PersonOutline",
            }),
            menu({
              id: "hidden",
              name: "隐藏页面",
              type: 2,
              path: "/system/hidden",
              visible: false,
            }),
            menu({
              id: "button",
              name: "新增用户",
              type: 3,
              path: "/system/user/create",
            }),
          ],
        }),
      ]),
    );

    expect(groups).toEqual([
      {
        key: "group",
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
    ]);
  });

  it("groups top-level leaves under the root navigation label", () => {
    const groups = buildNavGroups(
      userWithMenus([
        menu({ id: "dashboard", name: "工作台", type: 2, path: "/dashboard" }),
        menu({ id: "profile", name: "个人资料", type: 2, path: "profile" }),
      ]),
    );

    expect(groups).toHaveLength(1);
    expect(groups[0]?.label).toBe("导航");
    expect(groups[0]?.items.map((item) => item.path)).toEqual(["/dashboard", "/profile"]);
  });

  it("flattens nested Java directory menus into visible leaves", () => {
    const groups = buildNavGroups(
      userWithMenus([
        menu({
          id: "monitor",
          name: "系统监控",
          type: 1,
          children: [
            menu({
              id: "runtime",
              name: "运行时",
              type: 1,
              children: [
                menu({
                  id: "server",
                  name: "服务监控",
                  type: 2,
                  path: "monitor/server",
                }),
              ],
            }),
          ],
        }),
      ]),
    );

    expect(groups[0]?.items.map((item) => item.path)).toEqual(["/monitor/server"]);
  });
});

describe("isVisibleMenuPath", () => {
  it("matches only visible navigable menu paths", () => {
    const user = userWithMenus([
      menu({ id: "users", name: "用户管理", type: 2, path: "system/user" }),
      menu({
        id: "hidden",
        name: "隐藏页面",
        type: 2,
        path: "/system/hidden",
        visible: false,
      }),
    ]);

    expect(isVisibleMenuPath(user, "/system/user/")).toBe(true);
    expect(isVisibleMenuPath(user, "/system/hidden")).toBe(false);
    expect(isVisibleMenuPath(user, "/system/unknown")).toBe(false);
    expect(isVisibleMenuPath(null, "/system/user")).toBe(false);
  });
});
