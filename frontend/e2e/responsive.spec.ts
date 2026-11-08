import { expect, test } from "@playwright/test";

const viewports = [
  { width: 1920, height: 1080 },
  { width: 1440, height: 900 },
  { width: 1280, height: 800 },
  { width: 1024, height: 768 },
  { width: 768, height: 1024 },
  { width: 390, height: 844 },
  { width: 360, height: 800 },
];

for (const viewport of viewports) {
  test(`login has no horizontal overflow at ${viewport.width}px`, async ({ page }, testInfo) => {
    test.skip(
      testInfo.project.name !== "desktop",
      "The viewport matrix runs once in the desktop project.",
    );
    await page.setViewportSize(viewport);
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
    const hasHorizontalOverflow = await page.evaluate(
      () => document.documentElement.scrollWidth > window.innerWidth + 1,
    );
    expect(hasHorizontalOverflow).toBe(false);
    await page.screenshot({
      path: `visual-baselines/login-${viewport.width}-light.png`,
      fullPage: true,
      animations: "disabled",
    });
  });
}
