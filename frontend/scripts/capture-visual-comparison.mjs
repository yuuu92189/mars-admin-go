import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { chromium } from "@playwright/test";

const reactUrl = process.env.REACT_URL ?? "http://127.0.0.1:3000";
const vueUrl = process.env.VUE_REFERENCE_URL ?? "http://127.0.0.1:5173";
const outputDir = path.resolve("visual-comparison");

const scenarios = [
  {
    name: "desktop-light",
    viewport: { width: 1440, height: 900 },
    theme: "light",
  },
  {
    name: "desktop-dark",
    viewport: { width: 1440, height: 900 },
    theme: "dark",
  },
  {
    name: "mobile-light",
    viewport: { width: 390, height: 844 },
    theme: "light",
  },
  {
    name: "mobile-dark",
    viewport: { width: 390, height: 844 },
    theme: "dark",
  },
];

const captchaSvg = `
<svg xmlns="http://www.w3.org/2000/svg" width="116" height="38">
  <rect width="116" height="38" fill="#f3f5f6"/>
  <text x="12" y="26" font-family="Arial, sans-serif" font-size="18" fill="#101418">1+3=?</text>
</svg>`.trim();
const captchaImage = `data:image/svg+xml,${encodeURIComponent(captchaSvg)}`;

async function capture(browser, app, scenario) {
  const context = await browser.newContext({
    viewport: scenario.viewport,
    deviceScaleFactor: 1,
  });
  const page = await context.newPage();
  await page.emulateMedia({
    reducedMotion: "reduce",
    colorScheme: scenario.theme,
  });
  await page.addInitScript(
    ({ appName, theme }) => {
      if (appName === "react") {
        localStorage.setItem(
          "scaffold.appearance",
          JSON.stringify({ theme, density: "regular", railCollapsed: false }),
        );
      } else {
        localStorage.setItem(
          "scaffold.appearance",
          JSON.stringify({ theme, density: "regular", railCollapsed: false }),
        );
        localStorage.removeItem("scaffold.login-hint");
        sessionStorage.setItem("scaffold.login-intro", "1");
      }
    },
    { appName: app.name, theme: scenario.theme },
  );

  if (app.name === "react") {
    await page.route("**/api/v1/admin/settings/public", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ "security.auth.captcha_enabled": true }),
      }),
    );
    await page.route("**/api/v1/auth/captcha*", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          uuid: "00000000-0000-4000-8000-000000000001",
          img: captchaImage,
          slider: false,
        }),
      }),
    );
  } else {
    await page.route("**/api/auth/captcha*", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          code: 200,
          message: "ok",
          data: {
            uuid: "00000000-0000-4000-8000-000000000001",
            img: captchaImage,
            imgDark: captchaImage,
          },
        }),
      }),
    );
  }

  await page.goto(`${app.url}/login`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "登录" }).waitFor();
  if (app.name === "react") {
    await page.getByRole("textbox", { name: "账号" }).fill("test");
    await page.getByLabel("密码").fill("test123");
    await page.getByRole("heading", { name: "登录" }).click();
  }
  await page.waitForFunction(() =>
    [...document.querySelectorAll(".captcha-box img")].some(
      (image) =>
        image instanceof HTMLImageElement &&
        image.complete &&
        image.naturalWidth > 0 &&
        getComputedStyle(image).display !== "none",
    ),
  );
  await page.waitForTimeout(100);

  const screenshotPath = path.join(outputDir, `${app.name}-login-${scenario.name}.png`);
  await page.screenshot({
    path: screenshotPath,
    fullPage: true,
    animations: "disabled",
  });

  const metrics = await page.evaluate(() => {
    const rect = (selector) => {
      const element = document.querySelector(selector);
      if (!element) return null;
      const bounds = element.getBoundingClientRect();
      return {
        x: Math.round(bounds.x * 100) / 100,
        y: Math.round(bounds.y * 100) / 100,
        width: Math.round(bounds.width * 100) / 100,
        height: Math.round(bounds.height * 100) / 100,
      };
    };
    return {
      viewport: { width: window.innerWidth, height: window.innerHeight },
      scrollWidth: document.documentElement.scrollWidth,
      login: rect(".login"),
      brandPanel: rect(".brand-panel"),
      loginCard: rect(".login-card"),
      loginSide: rect(".login-side"),
    };
  });

  await context.close();
  return { app: app.name, scenario: scenario.name, metrics };
}

await mkdir(outputDir, { recursive: true });
const browser = await chromium.launch({ headless: true });
const records = [];
try {
  for (const scenario of scenarios) {
    records.push(
      await capture(browser, { name: "vue", url: vueUrl }, scenario),
      await capture(browser, { name: "react", url: reactUrl }, scenario),
    );
  }
} finally {
  await browser.close();
}

await writeFile(
  path.join(outputDir, "metrics.json"),
  `${JSON.stringify({ reference: vueUrl, implementation: reactUrl, records }, null, 2)}\n`,
);

console.log(`Captured ${records.length} screenshots in ${outputDir}`);
