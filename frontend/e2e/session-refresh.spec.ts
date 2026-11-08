import { expect, test } from "@playwright/test";

test.describe.configure({ mode: "serial" });

const username = process.env.E2E_ADMIN_USERNAME;
const password = process.env.E2E_ADMIN_PASSWORD;
const sessionTtl = Number(process.env.E2E_SESSION_TTL_SECONDS ?? "0");

test.skip(
  !username || !password || !sessionTtl,
  "Set E2E_ADMIN_USERNAME, E2E_ADMIN_PASSWORD and E2E_SESSION_TTL_SECONDS for refresh integration.",
);

test("expired access session refreshes through the BFF and returns to the dashboard", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "Refresh integration runs once.");
  await page.goto("/login");
  await page.getByRole("textbox", { name: "账号" }).fill(username ?? "");
  await page.getByLabel("密码").fill(password ?? "");
  await page.getByPlaceholder("请输入计算结果").fill("1");
  await page.getByRole("button", { name: "登录" }).click();
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByRole("heading", { name: "系统运行概况" })).toBeVisible();

  await page.waitForTimeout(sessionTtl * 1000 + 1_500);
  const refreshResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/api/v1/auth/refresh") && response.request().method() === "GET",
  );
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByRole("heading", { name: "系统运行概况" })).toBeVisible();
  expect([302, 307]).toContain((await refreshResponse).status());
});
