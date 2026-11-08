import { expect, test, type Page } from "@playwright/test";

const username = process.env.E2E_ADMIN_USERNAME;
const password = process.env.E2E_ADMIN_PASSWORD;

test.skip(
  !username || !password,
  "Set E2E_ADMIN_USERNAME and E2E_ADMIN_PASSWORD to run the real Phase 3 API flow.",
);
test.setTimeout(90_000);

async function login(page: Page) {
  await page.goto("/login");
  await page.getByRole("textbox", { name: "账号" }).fill(username ?? "");
  await page.getByLabel("密码").fill(password ?? "");
  const captcha = page.getByPlaceholder("请输入计算结果");
  if (await captcha.isVisible()) await captcha.fill("1");
  await page.getByRole("button", { name: "登录" }).click();
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByRole("heading", { name: /登录脉搏/ })).toBeVisible();
}

test.describe.configure({ mode: "serial" });

test("all Phase 3 Go-backed pages render real business content", async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "The page matrix runs once on desktop.");
  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  await page.setViewportSize({ width: 1440, height: 900 });
  await login(page);

  const pages = [
    ["/dashboard", "登录脉搏"],
    ["/system/user", "用户管理"],
    ["/system/role", "角色管理"],
    ["/system/menu", "菜单管理"],
    ["/system/dept", "部门管理"],
    ["/system/post", "岗位管理"],
    ["/system/dict", "字典管理"],
    ["/system/config", "系统配置"],
    ["/system/notice", "通知中心"],
    ["/monitor/job", "定时任务"],
    ["/monitor/operlog", "操作日志"],
    ["/monitor/loginlog", "登录日志"],
    ["/tool/generator", "代码生成器"],
  ] as const;

  for (const [path, heading] of pages) {
    const errorsBefore = consoleErrors.length;
    await page.goto(path);
    await expect(page.getByRole("heading", { name: heading, exact: true })).toBeVisible();
    await expect(page.getByText("数据加载失败", { exact: true })).toHaveCount(0);
    await expect(page.locator("main")).toBeVisible();
    expect(consoleErrors.slice(errorsBefore), `console errors on ${path}`).toEqual([]);
  }
});

test("users, roles, dictionaries and scheduler expose their primary interactions", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "Desktop interaction coverage.");
  await login(page);

  await page.goto("/system/user");
  await expect(page.getByRole("button", { name: "新增用户" })).toBeVisible();
  await page.getByRole("tab", { name: "看板" }).click();
  await expect(page.locator(".board")).toBeVisible();

  await page.goto("/system/role");
  await page.getByRole("tab", { name: "矩阵" }).click();
  await expect(page.getByText("角色 × 菜单权限矩阵")).toBeVisible();

  await page.goto("/system/dict");
  if (await page.locator(".dict-swatch").count()) {
    await expect(page.locator(".dict-swatch").first()).toBeVisible();
    await page.locator(".dict-swatch").first().click();
    await expect(page.getByText(/字典数据 ·/)).toBeVisible();
    await page.getByRole("button", { name: "关闭" }).click();
  } else {
    await expect(page.getByText("暂无字典类型")).toBeVisible();
  }

  await page.goto("/monitor/job");
  await expect(page.getByRole("button", { name: "新增任务" })).toBeVisible();
});

test("Phase 3 pages have no horizontal overflow at required viewports", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "Viewport matrix runs once.");
  await login(page);
  const paths = [
    "/dashboard",
    "/system/user",
    "/system/role",
    "/system/menu",
    "/system/dept",
    "/system/post",
    "/system/dict",
    "/system/config",
    "/system/notice",
    "/monitor/job",
    "/monitor/operlog",
    "/monitor/loginlog",
    "/tool/generator",
  ];
  const viewports = [
    { width: 1440, height: 900 },
    { width: 1024, height: 768 },
    { width: 768, height: 1024 },
    { width: 390, height: 844 },
  ];

  for (const path of paths) {
    await page.goto(path);
    for (const viewport of viewports) {
      await page.setViewportSize(viewport);
      await page.waitForTimeout(120);
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth > window.innerWidth + 1,
      );
      expect(overflow, `${path} overflow at ${viewport.width}x${viewport.height}`).toBe(false);
    }
  }
});
