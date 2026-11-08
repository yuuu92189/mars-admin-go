import { expect, test } from "@playwright/test";

test("login desktop light baseline", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.addInitScript(() => localStorage.setItem("mars:theme", "light"));
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  await expect(page).toHaveScreenshot("login-desktop-light.png", {
    fullPage: true,
    animations: "disabled",
    mask: [page.locator(".captcha-box")],
  });
});

test("login mobile dark baseline", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.addInitScript(() => localStorage.setItem("mars:theme", "dark"));
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  await expect(page).toHaveScreenshot("login-mobile-dark.png", {
    fullPage: true,
    animations: "disabled",
    mask: [page.locator(".captcha-box")],
  });
});
