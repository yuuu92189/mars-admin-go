import { expect, test, type Page } from "@playwright/test";

test.describe.configure({ mode: "serial" });

const username = process.env.E2E_ADMIN_USERNAME;
const password = process.env.E2E_ADMIN_PASSWORD;

test.skip(
  !username || !password,
  "Set E2E_ADMIN_USERNAME and E2E_ADMIN_PASSWORD to run the real API flow.",
);

async function login(page: Page) {
  await page.goto("/login");
  await page.getByRole("textbox", { name: "账号" }).fill(username ?? "");
  await page.getByLabel("密码").fill(password ?? "");
  const captcha = page.getByPlaceholder("请输入计算结果");
  if (await captcha.isVisible()) await captcha.fill("1");
  await page.getByRole("button", { name: "登录" }).click();
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByRole("heading", { name: "登录脉搏" })).toBeVisible();
  await page.waitForTimeout(900);
}

test("admin can log in, load the shell and fetch dashboard counters", async ({
  page,
}, testInfo) => {
  await page.addInitScript(() =>
    localStorage.setItem(
      "scaffold.appearance",
      JSON.stringify({ theme: "light", density: "regular", railCollapsed: false }),
    ),
  );
  await login(page);
  await expect(page.getByText("真实运行数据")).toBeVisible();
  await page.screenshot({
    path: `visual-baselines/dashboard-${testInfo.project.name}-light.png`,
    fullPage: true,
    animations: "disabled",
  });

  await page.getByRole("button", { name: "外观设置" }).click();
  await page.getByRole("group", { name: "主题" }).getByRole("button", { name: "深色" }).click();
  await page.getByRole("group", { name: "表格密度" }).getByRole("button", { name: "紧凑" }).click();
  await page.getByRole("button", { name: "完成" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-density", "compact");
  await expect
    .poll(() =>
      page.evaluate(() =>
        getComputedStyle(document.documentElement).getPropertyValue("--row-h").trim(),
      ),
    )
    .toBe("40px");
  await page.reload();
  await expect(page.getByRole("heading", { name: "登录脉搏" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-density", "compact");
  await page.waitForTimeout(500);
  await page.screenshot({
    path: `visual-baselines/dashboard-${testInfo.project.name}-dark.png`,
    fullPage: true,
    animations: "disabled",
  });

  await page.locator(".user-chip").click();
  await page.getByRole("menuitem", { name: "退出登录" }).click();
  await expect(page.getByRole("dialog", { name: "退出登录" })).toBeVisible();
  await page.getByRole("button", { name: "退出" }).click();
  await expect(page).toHaveURL(/\/login/);
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login\?redirect=/);
});

test("desktop shell supports rail collapse and command palette keyboard flow", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "Desktop interaction coverage.");
  await page.setViewportSize({ width: 1440, height: 900 });
  await login(page);
  const frame = page.locator(".app-frame");
  const toggle = page.getByRole("button", { name: "切换导航" });

  await expect(frame).not.toHaveClass(/is-collapsed/);
  await toggle.click();
  await expect(frame).toHaveClass(/is-collapsed/);
  await page.reload();
  await expect(frame).toHaveClass(/is-collapsed/);
  await page.screenshot({
    path: "visual-baselines/dashboard-desktop-collapsed-light.png",
    fullPage: true,
    animations: "disabled",
  });
  await toggle.click();
  await expect(frame).not.toHaveClass(/is-collapsed/);
  await page.keyboard.press("Control+K");
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeVisible();
  await page.getByRole("combobox", { name: "搜索页面、用户或动作" }).fill("用户");
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/system\/user/);
  await page.goBack();
  await expect(page).toHaveURL(/\/dashboard/);

  await page.keyboard.press("Control+K");
  const userSearch = page.getByRole("combobox", { name: "搜索页面、用户或动作" });
  await userSearch.fill(username ?? "");
  const userResult = page
    .getByRole("option")
    .filter({ hasText: `@${username ?? ""}` })
    .first();
  await expect(userResult).toBeVisible();
  await userResult.click();
  await expect(page).toHaveURL(/\/system\/user\?keyword=/);
  await page.goto("/dashboard");

  await page.keyboard.press("Control+K");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeHidden();

  await page.getByRole("button", { name: "搜索菜单" }).click();
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "命令面板" })).toBeHidden();

  await page.locator(".user-chip").click();
  await page.getByRole("menuitem", { name: "修改密码" }).click();
  await expect(page.getByRole("dialog", { name: "修改密码" })).toBeVisible();
  await page.getByRole("button", { name: "确认修改" }).click();
  await expect(page.getByText("请输入当前密码")).toBeVisible();
  await expect(page.getByText("请输入新密码")).toBeVisible();
  await page.getByLabel("当前密码").fill("current-password");
  await page.getByLabel(/^新密码/).fill("short");
  await page.getByLabel("确认新密码").fill("short");
  await page.getByRole("button", { name: "确认修改" }).click();
  await expect(page.getByText("新密码至少 8 位")).toBeVisible();
  await page.getByRole("button", { name: "取消" }).click();
});

test("mobile navigation opens and closes with the scrim", async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== "mobile", "Mobile interaction coverage.");
  await login(page);
  const frame = page.locator(".app-frame");

  await expect(frame).not.toHaveClass(/is-mobile-open/);
  await page.getByRole("button", { name: "切换导航" }).click();
  await expect(frame).toHaveClass(/is-mobile-open/);
  await expect(page.locator(".rail-scrim")).toBeVisible();
  await page.screenshot({
    path: "visual-baselines/dashboard-mobile-open-light.png",
    fullPage: true,
    animations: "disabled",
  });
  await page.locator(".rail-scrim").click({ position: { x: 360, y: 420 } });
  await expect(frame).not.toHaveClass(/is-mobile-open/);
});

test("dashboard renders controlled permission and service error states", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "Controlled error-state coverage.");
  await login(page);
  let mode: "forbidden" | "failure" = "forbidden";
  await page.route(/\/api\/v1\/admin\/monitoring\/dashboard/, async (route) => {
    if (mode === "forbidden") {
      await route.fulfill({
        status: 403,
        contentType: "application/problem+json",
        body: JSON.stringify({
          type: "https://mars-admin.local/problems/permission-denied",
          title: "Permission denied",
          status: 403,
          detail: "You do not have permission.",
          code: "permission_denied",
        }),
      });
      return;
    }
    await route.fulfill({
      status: 503,
      contentType: "application/problem+json",
      body: JSON.stringify({
        type: "https://mars-admin.local/problems/service-unavailable",
        title: "Service unavailable",
        status: 503,
        detail: "运行数据服务暂时不可用。",
        code: "service_unavailable",
      }),
    });
  });

  await page.reload();
  await expect(page.getByText("当前账号没有运行数据权限")).toBeVisible();
  await expect(page.getByText("需要 audit:read 权限")).toBeVisible();

  mode = "failure";
  await page.reload();
  await expect(page.getByText("运行数据加载失败")).toBeVisible();
  await expect(page.getByText("运行数据服务暂时不可用。")).toBeVisible();
});

test("dashboard has no horizontal overflow at required viewports", async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "The viewport matrix runs once.");
  await login(page);
  for (const viewport of [
    { width: 1920, height: 1080 },
    { width: 1440, height: 900 },
    { width: 1024, height: 768 },
    { width: 768, height: 1024 },
    { width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    await page.waitForTimeout(250);
    const hasHorizontalOverflow = await page.evaluate(
      () => document.documentElement.scrollWidth > window.innerWidth + 1,
    );
    expect(hasHorizontalOverflow, `dashboard overflow at ${viewport.width}px`).toBe(false);
    await page.screenshot({
      path: `visual-baselines/dashboard-${viewport.width}-light.png`,
      fullPage: true,
      animations: "disabled",
    });
  }
});

test("authenticated unknown route renders the 404 state", async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "One authenticated 404 check is enough.");
  await login(page);
  await page.goto("/not-a-real-route");
  await expect(page.getByText("页面不存在")).toBeVisible();
  await expect(page.getByRole("link", { name: "返回工作台" })).toBeVisible();
});
