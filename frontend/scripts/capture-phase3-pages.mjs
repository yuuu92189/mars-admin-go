import { mkdir, readdir, unlink, writeFile } from "node:fs/promises";
import path from "node:path";
import { chromium } from "@playwright/test";

const reactUrl = process.env.REACT_URL ?? "http://127.0.0.1:3000";
const username = process.env.E2E_ADMIN_USERNAME;
const password = process.env.E2E_ADMIN_PASSWORD;
const outputDir = path.resolve("../docs/phase-3/visual-regression/react");

if (!username || !password) {
  throw new Error("E2E_ADMIN_USERNAME and E2E_ADMIN_PASSWORD are required.");
}

const pages = [
  { name: "dashboard", path: "/dashboard", heading: "登录脉搏" },
  { name: "users", path: "/system/user", heading: "用户管理" },
  { name: "roles", path: "/system/role", heading: "角色管理" },
  { name: "menus", path: "/system/menu", heading: "菜单管理" },
  { name: "departments", path: "/system/dept", heading: "部门管理" },
  { name: "positions", path: "/system/post", heading: "岗位管理" },
  { name: "dictionaries", path: "/system/dict", heading: "字典管理" },
  { name: "settings", path: "/system/config", heading: "系统配置" },
  { name: "notices", path: "/system/notice", heading: "通知中心" },
  { name: "scheduler", path: "/monitor/job", heading: "定时任务" },
  { name: "operation-logs", path: "/monitor/operlog", heading: "操作日志" },
  { name: "login-logs", path: "/monitor/loginlog", heading: "登录日志" },
  { name: "codegen", path: "/tool/generator", heading: "代码生成器" },
];

const scenarios = [
  {
    name: "desktop-light",
    viewport: { width: 1440, height: 900 },
    appearance: { theme: "light", density: "regular", railCollapsed: false },
  },
  {
    name: "desktop-dark",
    viewport: { width: 1440, height: 900 },
    appearance: { theme: "dark", density: "regular", railCollapsed: false },
  },
  {
    name: "desktop-compact",
    viewport: { width: 1440, height: 900 },
    appearance: { theme: "light", density: "compact", railCollapsed: false },
  },
  {
    name: "tablet-landscape",
    viewport: { width: 1024, height: 768 },
    appearance: { theme: "light", density: "regular", railCollapsed: false },
  },
  {
    name: "tablet-portrait",
    viewport: { width: 768, height: 1024 },
    appearance: { theme: "light", density: "regular", railCollapsed: false },
  },
  {
    name: "mobile",
    viewport: { width: 390, height: 844 },
    appearance: { theme: "light", density: "regular", railCollapsed: false },
  },
];

await mkdir(outputDir, { recursive: true });
for (const file of await readdir(outputDir)) {
  if (file.startsWith("failure-") && file.endsWith(".png")) {
    await unlink(path.join(outputDir, file));
  }
}
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: scenarios[0].viewport,
  deviceScaleFactor: 1,
});
const page = await context.newPage();
await page.emulateMedia({ reducedMotion: "reduce" });
const loginResponse = await context.request.post(`${reactUrl}/api/v1/auth/login`, {
  data: { username, password, clientType: "admin", deviceName: "phase3-visual" },
});
if (!loginResponse.ok()) {
  throw new Error(`Unable to create visual session: HTTP ${loginResponse.status()}`);
}

const records = [];
const consoleErrors = [];
page.on("console", (message) => {
  if (message.type() === "error") consoleErrors.push(message.text());
});
page.on("pageerror", (error) => consoleErrors.push(error.message));

try {
  await page.goto(`${reactUrl}/dashboard`, { waitUntil: "networkidle" });
  for (const pageDef of pages) {
    for (const scenario of scenarios) {
      await page.setViewportSize(scenario.viewport);
      await page.evaluate((appearance) => {
        localStorage.setItem("scaffold.appearance", JSON.stringify(appearance));
        document.cookie = `mars_appearance=${encodeURIComponent(
          JSON.stringify(appearance),
        )}; Path=/; Max-Age=31536000; SameSite=Lax`;
      }, scenario.appearance);
      await page.goto(`${reactUrl}${pageDef.path}`, { waitUntil: "networkidle" });
      await page.getByRole("heading", { name: pageDef.heading, exact: true }).waitFor();
      await page.addStyleTag({
        content: "nextjs-portal { display: none !important; }",
      });
      await page.waitForTimeout(350);
      const fileName = `${pageDef.name}-${scenario.name}.png`;
      await page.screenshot({
        path: path.join(outputDir, fileName),
        fullPage: true,
        animations: "disabled",
      });
      const metrics = await page.evaluate(() => ({
        viewport: { width: window.innerWidth, height: window.innerHeight },
        scrollWidth: document.documentElement.scrollWidth,
        scrollHeight: document.documentElement.scrollHeight,
        horizontalOverflow: document.documentElement.scrollWidth > window.innerWidth + 1,
        pageRect: (() => {
          const element = document.querySelector(".page");
          if (!element) return null;
          const rect = element.getBoundingClientRect();
          return {
            width: Math.round(rect.width * 100) / 100,
            height: Math.round(rect.height * 100) / 100,
          };
        })(),
        theme: document.documentElement.dataset.theme,
        density: document.documentElement.dataset.density,
      }));
      records.push({ page: pageDef.name, scenario: scenario.name, fileName, ...metrics });
    }
  }
} finally {
  await context.close();
  await browser.close();
}

await writeFile(
  path.join(outputDir, "metrics.json"),
  `${JSON.stringify(
    {
      reference: reactUrl,
      generatedAt: new Date().toISOString(),
      consoleErrors,
      records,
    },
    null,
    2,
  )}\n`,
);
console.log(`Captured ${records.length} Phase 3 React screenshots in ${outputDir}`);
