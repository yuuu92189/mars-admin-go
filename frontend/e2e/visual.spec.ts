import { expect, test, type Page } from "@playwright/test";

async function hideDevelopmentOverlays(page: Page) {
  await page.addStyleTag({
    content: "nextjs-portal { display: none !important; }",
  });
}

test("login desktop light baseline", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.addInitScript(() =>
    localStorage.setItem(
      "scaffold.appearance",
      JSON.stringify({ theme: "light", density: "regular", railCollapsed: false }),
    ),
  );
  await page.goto("/login");
  await hideDevelopmentOverlays(page);
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  if ((page.viewportSize()?.width ?? 0) >= 900) {
    await expect(page.getByText("inventory synced · 0 drift")).toBeVisible();
  }
  await expect(page).toHaveScreenshot("login-desktop-light.png", {
    fullPage: true,
    animations: "disabled",
    mask: [page.locator(".captcha-box")],
  });
});

test("login mobile dark baseline", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.addInitScript(() =>
    localStorage.setItem(
      "scaffold.appearance",
      JSON.stringify({ theme: "dark", density: "regular", railCollapsed: false }),
    ),
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/login");
  await hideDevelopmentOverlays(page);
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  await expect(page).toHaveScreenshot("login-mobile-dark.png", {
    fullPage: true,
    animations: "disabled",
    mask: [page.locator(".captcha-box")],
  });
});
