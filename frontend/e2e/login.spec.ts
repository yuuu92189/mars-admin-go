import { expect, test } from "@playwright/test";

test("login page renders the canonical form", async ({ page }) => {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "账号" })).toBeVisible();
  await expect(page.getByLabel("密码")).toBeVisible();
  await expect(page.getByRole("button", { name: "登录", exact: true })).toBeVisible();
  await expect(page.getByText("资产管理", { exact: true })).toHaveCount(0);
  await expect(page.getByText("漏洞分级", { exact: true })).toHaveCount(0);
  await expect(page.getByText("修复工单", { exact: true })).toHaveCount(0);
});

test("default appearance matches the Vue reference", async ({ page }) => {
  await page.goto("/login");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expect(page.locator("html")).toHaveAttribute("data-density", "regular");
  await expect(page.locator("html")).toHaveAttribute("data-rail-collapsed", "false");
});

test("register route is policy-aware and links back to login", async ({ page }) => {
  await page.goto("/register");
  await expect(page.getByRole("heading", { name: "注册", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "立即登录" })).toBeVisible();
  await expect(
    page.getByText("填写账号信息，注册结果由 Go 后端策略校验。", { exact: true }),
  ).toBeVisible();
});

test("protected dashboard redirects unauthenticated users", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login\?redirect=/);
});

test("theme preference is applied without authentication", async ({ page }) => {
  await page.goto("/login");
  await page.evaluate(() =>
    localStorage.setItem(
      "scaffold.appearance",
      JSON.stringify({ theme: "dark", density: "compact", railCollapsed: true }),
    ),
  );
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-density", "compact");
  await expect(page.locator("html")).toHaveAttribute("data-rail-collapsed", "true");
});

test("forbidden page is explicit and does not leak session details", async ({ page }) => {
  await page.goto("/forbidden");
  await expect(page.getByText("没有访问权限")).toBeVisible();
  await expect(page.getByText("当前账号没有访问该页面的权限。")).toBeVisible();
  await expect(page.getByRole("link", { name: "返回工作台" })).toBeVisible();
});

test("wrong password stays on login and reports the API error", async ({ page }) => {
  const unknownUsername = `e2e_missing_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
  await page.goto("/login");
  await page.getByRole("textbox", { name: "账号" }).fill(unknownUsername);
  await page.getByLabel("密码").fill("definitely-wrong-password");
  const captcha = page.getByPlaceholder("请输入计算结果");
  if (await captcha.isVisible()) await captcha.fill("1");
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page).toHaveURL(/\/login/);
  await expect(page.getByRole("status")).toContainText("登录失败", { timeout: 10_000 });
});
