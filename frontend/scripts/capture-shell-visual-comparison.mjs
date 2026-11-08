import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { chromium } from "@playwright/test";
import pixelmatch from "pixelmatch";
import { PNG } from "pngjs";

const vueUrl = process.env.VUE_REFERENCE_URL ?? "http://127.0.0.1:5173";
const reactUrl = process.env.REACT_URL ?? "http://127.0.0.1:3000";
const apiUrl = process.env.VISUAL_API_URL ?? "http://127.0.0.1:8081";
const username = process.env.E2E_ADMIN_USERNAME;
const password = process.env.E2E_ADMIN_PASSWORD;
const outputDir = path.resolve("../docs/frontend/visual-regression/artifacts");

const appearance = {
  light: { theme: "light", density: "regular", railCollapsed: false },
  darkCollapsed: { theme: "dark", density: "regular", railCollapsed: true },
  compact: { theme: "light", density: "compact", railCollapsed: false },
};

const vueUser = {
  id: "visual-user",
  username: username ?? "admin",
  nickname: username ?? "admin",
  displayName: username ?? "admin",
  roles: ["admin"],
  permissions: ["user:read", "role:read", "menu:read", "monitor:loginlog:list"],
};

const vueRoutes = {
  routes: [
    {
      id: "dashboard",
      name: "工作台",
      path: "/dashboard",
      component: "dashboard/index",
      icon: "dashboard",
    },
    {
      id: "system",
      name: "系统管理",
      path: "/system",
      icon: "menu",
      children: [
        {
          id: "users",
          name: "用户管理",
          path: "system/user",
          component: "system/user/index",
          icon: "PersonOutline",
        },
        {
          id: "roles",
          name: "角色管理",
          path: "system/role",
          component: "system/role/index",
          icon: "PeopleOutline",
        },
      ],
    },
  ],
};

const vueSummary = {
  stats: [
    { label: "用户总数", value: 12, unit: "人" },
    { label: "在线会话", value: 3, unit: "个" },
    { label: "消息", value: 24, unit: "条" },
  ],
  trend: [2, 4, 3, 6, 5, 7, 8],
  activities: [],
  roleDist: [
    { label: "管理员", value: 2 },
    { label: "运营", value: 10 },
  ],
  trendTotal: 35,
  successRate: 99.2,
  trendPrev: 30,
  dailyOps: [
    { date: "11-02", total: 3, failed: 0 },
    { date: "11-03", total: 5, failed: 0 },
    { date: "11-04", total: 4, failed: 1 },
    { date: "11-05", total: 7, failed: 0 },
    { date: "11-06", total: 6, failed: 0 },
    { date: "11-07", total: 8, failed: 0 },
    { date: "11-08", total: 9, failed: 0 },
  ],
};

const reactDashboard = {
  users: 12,
  userCount: 12,
  onlineSessions: 3,
  messages: 24,
  files: 5,
  roleCount: 2,
  menuCount: 8,
  permissionCount: 16,
  publishedNotices: 1,
  outboxPending: 0,
  outboxDead: 0,
  range: "24h",
  loginTrend: [
    { label: "00:00", value: 2 },
    { label: "04:00", value: 4 },
    { label: "08:00", value: 3 },
    { label: "12:00", value: 6 },
    { label: "16:00", value: 5 },
    { label: "20:00", value: 7 },
    { label: "23:00", value: 8 },
  ],
  loginTotal: 35,
  loginPreviousTotal: 30,
  operationTrend: [
    { label: "11/02", value: 3, failed: 0 },
    { label: "11/03", value: 5, failed: 0 },
    { label: "11/04", value: 4, failed: 1 },
    { label: "11/05", value: 7, failed: 0 },
    { label: "11/06", value: 6, failed: 0 },
    { label: "11/07", value: 8, failed: 0 },
    { label: "11/08", value: 9, failed: 0 },
  ],
  operationTotal: 35,
  operationSuccessRate: 99.2,
  roleDistribution: [
    { label: "管理员", value: 2 },
    { label: "运营", value: 10 },
  ],
  recentLogins: [],
  recentActivity: [],
};

const scenarios = [
  {
    name: "shell-expanded-light-desktop",
    viewport: { width: 1440, height: 900 },
    appearance: appearance.light,
  },
  {
    name: "shell-collapsed-dark-desktop",
    viewport: { width: 1440, height: 900 },
    appearance: appearance.darkCollapsed,
  },
  {
    name: "shell-compact-light-desktop",
    viewport: { width: 1440, height: 900 },
    appearance: appearance.compact,
  },
  {
    name: "shell-command-palette-light-desktop",
    viewport: { width: 1440, height: 900 },
    appearance: appearance.light,
    action: "command",
  },
  {
    name: "shell-appearance-dialog-light-desktop",
    viewport: { width: 1440, height: 900 },
    appearance: appearance.light,
    action: "appearance",
  },
  {
    name: "shell-user-menu-light-desktop",
    viewport: { width: 1440, height: 900 },
    appearance: appearance.light,
    action: "user-menu",
  },
  {
    name: "dashboard-mobile-light",
    viewport: { width: 390, height: 844 },
    appearance: appearance.light,
  },
  {
    name: "dashboard-mobile-dark",
    viewport: { width: 390, height: 844 },
    appearance: { ...appearance.light, theme: "dark" },
  },
];

function jsonEnvelope(data) {
  return JSON.stringify({ code: 200, message: "ok", msg: "ok", data });
}

async function getVueSession() {
  if (!username || !password) {
    throw new Error("E2E_ADMIN_USERNAME and E2E_ADMIN_PASSWORD are required for shell comparison.");
  }
  const response = await fetch(`${apiUrl}/api/auth/login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      username,
      password,
      captcha: "1",
      captchaUuid: "",
    }),
  });
  const payload = await response.json();
  if (!response.ok || payload.code !== 200 || !payload.data?.token) {
    throw new Error(`Unable to create Vue visual session: HTTP ${response.status}`);
  }
  return {
    token: payload.data.token,
    expiresIn: payload.data.expiresIn ?? 1800,
    user: {
      ...vueUser,
      username: payload.data.user?.username ?? username,
      nickname: payload.data.user?.displayName ?? username,
      roles: payload.data.access?.roles ?? vueUser.roles,
      permissions: payload.data.access?.permissions?.length
        ? payload.data.access.permissions
        : vueUser.permissions,
    },
  };
}

async function preparePage(browser, app, scenario, session) {
  const context = await browser.newContext({
    viewport: scenario.viewport,
    deviceScaleFactor: 1,
  });
  const page = await context.newPage();
  await page.emulateMedia({ reducedMotion: "reduce" });

  await page.addInitScript(
    ({ appName, appearanceValue, vueSession, recent }) => {
      localStorage.setItem("scaffold.appearance", JSON.stringify(appearanceValue));
      localStorage.setItem("scaffold.recent", JSON.stringify(recent));
      if (appName === "vue" && vueSession) {
        localStorage.setItem("scaffold.token", vueSession.token);
        localStorage.setItem("scaffold.user", JSON.stringify(vueSession.user));
        localStorage.setItem(
          "scaffold.expiresAt",
          String(Date.now() + vueSession.expiresIn * 1000),
        );
        sessionStorage.setItem("scaffold.login-intro", "1");
      }
    },
    {
      appName: app.name,
      appearanceValue: scenario.appearance,
      vueSession: session,
      recent: ["/system/user", "/system/role"],
    },
  );

  if (app.name === "vue") {
    await page.route("**/api/auth/me", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: jsonEnvelope(vueUser),
      }),
    );
    await page.route("**/api/auth/routes", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: jsonEnvelope(vueRoutes),
      }),
    );
    await page.route("**/api/dashboard/summary*", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: jsonEnvelope(vueSummary),
      }),
    );
    await page.goto(`${vueUrl}/dashboard`, { waitUntil: "networkidle" });
    await page.locator(".app-frame").waitFor();
  } else {
    await page.route(/\/api\/v1\/admin\/monitoring\/dashboard/, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(reactDashboard),
      }),
    );
    const loginResponse = await context.request.post(`${reactUrl}/api/v1/auth/login`, {
      data: {
        username,
        password,
        clientType: "admin",
        deviceName: "visual-regression",
      },
    });
    if (!loginResponse.ok()) {
      throw new Error(`Unable to create React visual session: HTTP ${loginResponse.status()}`);
    }
    await page.goto(`${reactUrl}/dashboard`, { waitUntil: "networkidle" });
    await page.locator(".app-frame").waitFor();
  }

  if (scenario.action === "command") {
    await page.keyboard.press("Control+K");
    await page.locator(".cmd-panel").waitFor();
  } else if (scenario.action === "appearance") {
    await page.getByRole("button", { name: "外观设置" }).click();
    await page.locator(".modal").waitFor();
  } else if (scenario.action === "user-menu") {
    await page.locator(".user-chip").click();
    await page.getByRole("menu").waitFor();
  }

  await page.waitForTimeout(900);
  return { context, page };
}

async function capture(app, scenario, session) {
  const browser = await chromium.launch({ headless: true });
  try {
    const { context, page } = await preparePage(browser, app, scenario, session);
    const screenshotPath = path.join(outputDir, `${app.name}-${scenario.name}.png`);
    await page.screenshot({ path: screenshotPath, animations: "disabled" });
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
      const style = (selector, property) => {
        const element = document.querySelector(selector);
        return element ? getComputedStyle(element).getPropertyValue(property) : null;
      };
      return {
        viewport: { width: window.innerWidth, height: window.innerHeight },
        scrollWidth: document.documentElement.scrollWidth,
        frame: rect(".app-frame"),
        rail: rect(".rail"),
        topbar: rect(".topbar"),
        content: rect(".content"),
        page: rect(".page"),
        railWidth: style(".rail", "width"),
        topbarHeight: style(".topbar", "height"),
        frameRadius: style(".app-frame", "border-radius"),
      };
    });
    await context.close();
    return { app: app.name, scenario: scenario.name, screenshotPath, metrics };
  } finally {
    await browser.close();
  }
}

async function comparePair(vue, react) {
  const vueImage = PNG.sync.read(await readFile(vue.screenshotPath));
  const reactImage = PNG.sync.read(await readFile(react.screenshotPath));
  if (vueImage.width !== reactImage.width || vueImage.height !== reactImage.height) {
    throw new Error(`Screenshot dimensions differ for ${vue.scenario}`);
  }
  const diff = new PNG({ width: vueImage.width, height: vueImage.height });
  const mismatchPixels = pixelmatch(
    vueImage.data,
    reactImage.data,
    diff.data,
    vueImage.width,
    vueImage.height,
    { threshold: 0.12 },
  );
  const diffPath = path.join(outputDir, `diff-${vue.scenario}.png`);
  await writeFile(diffPath, PNG.sync.write(diff));
  return {
    scenario: vue.scenario,
    mismatchPixels,
    mismatchRatio: Number((mismatchPixels / (vueImage.width * vueImage.height)).toFixed(4)),
    diffPath,
    vueMetrics: vue.metrics,
    reactMetrics: react.metrics,
  };
}

await mkdir(outputDir, { recursive: true });
const session = await getVueSession();
const records = [];
for (const scenario of scenarios) {
  const vue = await capture({ name: "vue", url: vueUrl }, scenario, session);
  const react = await capture({ name: "react", url: reactUrl }, scenario, session);
  const comparison = await comparePair(vue, react);
  records.push({ scenario: scenario.name, vue, react, comparison });
}

const portableRecords = records.map((record) => ({
  ...record,
  vue: {
    ...record.vue,
    screenshotPath: path.relative(outputDir, record.vue.screenshotPath),
  },
  react: {
    ...record.react,
    screenshotPath: path.relative(outputDir, record.react.screenshotPath),
  },
  comparison: {
    ...record.comparison,
    diffPath: path.relative(outputDir, record.comparison.diffPath),
  },
}));
await writeFile(
  path.join(outputDir, "shell-metrics.json"),
  `${JSON.stringify({ reference: vueUrl, implementation: reactUrl, records: portableRecords }, null, 2)}\n`,
);
console.log(`Captured ${records.length} Vue/React shell pairs in ${outputDir}`);
