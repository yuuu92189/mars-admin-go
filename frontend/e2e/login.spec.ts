import { expect, test } from "@playwright/test";

test("login page renders the canonical form", async ({ page }) => {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "账号" })).toBeVisible();
  await expect(page.getByLabel("密码")).toBeVisible();
  await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
});

test("protected dashboard redirects unauthenticated users", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login\?redirect=/);
});

test("theme preference is applied without authentication", async ({ page }) => {
  await page.goto("/login");
  await page.evaluate(() => localStorage.setItem("mars:theme", "dark"));
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("forbidden page is explicit and does not leak session details", async ({ page }) => {
  await page.goto("/forbidden");
  await expect(page.getByText("没有访问权限")).toBeVisible();
  await expect(page.getByText("当前账号没有访问该页面的权限。")).toBeVisible();
  await expect(page.getByRole("link", { name: "返回工作台" })).toBeVisible();
});
