import { expect, test, type Page } from "@playwright/test";

const adminUsername = process.env.E2E_ADMIN_USERNAME;
const adminPassword = process.env.E2E_ADMIN_PASSWORD;

test.describe.configure({ mode: "serial" });
test.setTimeout(90_000);
test.skip(
  !adminUsername || !adminPassword,
  "Set E2E_ADMIN_USERNAME and E2E_ADMIN_PASSWORD to run the real password-change flow.",
);
test.use({ trace: "off", video: "off", screenshot: "off" });

async function login(page: Page, username: string, password: string) {
  await page.goto("/login");
  await page.getByRole("textbox", { name: "账号" }).fill(username);
  await page.getByLabel("密码").fill(password);
  const captcha = page.getByPlaceholder("请输入计算结果");
  if (await captcha.isVisible()) await captcha.fill("1");
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });
}

async function csrfHeaders(page: Page): Promise<Record<string, string>> {
  const cookies = await page.context().cookies();
  const csrf = cookies.find((cookie) => cookie.name === "mars_csrf");
  if (!csrf) throw new Error("The authenticated browser context has no CSRF cookie.");
  return { "X-CSRF-Token": decodeURIComponent(csrf.value) };
}

test("a temporary user can change password and sign in with the new credential", async ({
  browser,
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "The password lifecycle runs once.");

  const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const username = `e2e_pwd_${suffix}`;
  const initialPassword = `E2e-init-${suffix}!`;
  const changedPassword = `E2e-new-${suffix}!`;
  let userID: string | undefined;

  try {
    await test.step("create a temporary user through the real admin API", async () => {
      await login(page, adminUsername ?? "", adminPassword ?? "");
      await expect(page).toHaveURL(/\/dashboard/);

      const createResponse = await page.request.post("/api/v1/admin/users", {
        headers: await csrfHeaders(page),
        data: {
          username,
          password: initialPassword,
          displayName: "Password lifecycle test",
          userType: "admin",
        },
      });
      expect(createResponse.status()).toBe(201);
      userID = ((await createResponse.json()) as { id: string }).id;
    });

    const userContext = await browser.newContext();
    try {
      const userPage = await userContext.newPage();
      await test.step("sign in as the temporary user", async () => {
        await login(userPage, username, initialPassword);
        await expect(userPage.locator(".app-frame")).toBeVisible();
      });

      await test.step("change the password through the Vue-aligned dialog", async () => {
        await test.step("open the command palette", async () => {
          await userPage.getByRole("button", { name: "搜索菜单" }).click();
          await expect(userPage.getByRole("dialog", { name: "命令面板" })).toBeVisible();
        });

        await test.step("open the password dialog", async () => {
          const commandInput = userPage.getByRole("combobox", {
            name: "搜索页面、用户或动作",
          });
          await commandInput.fill("修改密码");
          await userPage.keyboard.press("Enter");
          await expect(userPage.getByRole("dialog", { name: "修改密码" })).toBeVisible();
        });

        await test.step("submit the password change", async () => {
          await test.step("fill the current password", async () => {
            await userPage.getByLabel("当前密码").fill(initialPassword);
          });
          await test.step("fill the new password", async () => {
            await userPage.getByLabel(/^新密码/).fill(changedPassword);
          });
          await test.step("confirm the new password", async () => {
            await userPage.getByLabel("确认新密码").fill(changedPassword);
          });
          await test.step("submit and wait for the revoked session redirect", async () => {
            await userPage.getByRole("button", { name: "确认修改" }).click();
            await expect(userPage).toHaveURL(/\/login\?reason=password-changed/, {
              timeout: 20_000,
            });
          });
        });
      });

      await test.step("reject the old password and accept the new password", async () => {
        await userPage.getByRole("textbox", { name: "账号" }).fill(username);
        await userPage.getByLabel("密码").fill(initialPassword);
        const captchaOldPassword = userPage.getByPlaceholder("请输入计算结果");
        if (await captchaOldPassword.isVisible()) await captchaOldPassword.fill("1");
        await userPage.getByRole("button", { name: "登录", exact: true }).click();
        await expect(userPage.getByRole("status").filter({ hasText: "登录失败" })).toBeVisible();
        await expect(userPage).toHaveURL(/\/login/);

        await userPage.getByLabel("密码").fill(changedPassword);
        const captchaNewPassword = userPage.getByPlaceholder("请输入计算结果");
        if (await captchaNewPassword.isVisible()) await captchaNewPassword.fill("1");
        await userPage.getByRole("button", { name: "登录", exact: true }).click();
        await expect(userPage.locator(".app-frame")).toBeVisible({ timeout: 15_000 });
      });
    } finally {
      await userContext.close();
    }
  } finally {
    if (userID) {
      const cleanupContext = await browser.newContext();
      try {
        const cleanupPage = await cleanupContext.newPage();
        await test.step("delete the temporary user", async () => {
          await login(cleanupPage, adminUsername ?? "", adminPassword ?? "");
          await expect(cleanupPage).toHaveURL(/\/dashboard/);
          const response = await cleanupPage.request.delete(`/api/v1/admin/users/${userID}`, {
            headers: await csrfHeaders(cleanupPage),
          });
          expect(response.status()).toBe(204);
        });
      } finally {
        await cleanupContext.close();
      }
    }
  }
});
