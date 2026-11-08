import { mkdir, readdir, unlink, writeFile } from "node:fs/promises";
import path from "node:path";
import { chromium } from "@playwright/test";

const vueUrl = process.env.VUE_REFERENCE_URL ?? "http://127.0.0.1:5173";
const outputDir = path.resolve("../docs/phase-3/visual-regression/vue");
const token = "visual-reference-token";

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

const routeTree = [
  {
    id: 1,
    name: "工作台",
    path: "/dashboard",
    component: "dashboard/index",
    icon: "dashboard",
  },
  {
    id: 2,
    name: "系统管理",
    path: "/system",
    icon: "menu",
    children: [
      {
        id: 21,
        name: "用户管理",
        path: "system/user",
        component: "system/user/index",
        icon: "PersonOutline",
      },
      {
        id: 22,
        name: "角色管理",
        path: "system/role",
        component: "system/role/index",
        icon: "PeopleOutline",
      },
      {
        id: 23,
        name: "菜单管理",
        path: "system/menu",
        component: "system/menu/index",
        icon: "MenuOutline",
      },
      {
        id: 24,
        name: "字典管理",
        path: "system/dict",
        component: "system/dict/index",
        icon: "BookOutline",
      },
      {
        id: 25,
        name: "系统配置",
        path: "system/config",
        component: "system/config/index",
        icon: "SettingsSharp",
      },
    ],
  },
  {
    id: 3,
    name: "组织管理",
    path: "/org",
    icon: "dept",
    children: [
      {
        id: 31,
        name: "部门管理",
        path: "system/dept",
        component: "system/dept/index",
        icon: "GitNetworkOutline",
      },
      {
        id: 32,
        name: "岗位管理",
        path: "system/post",
        component: "system/post/index",
        icon: "IdCardOutline",
      },
    ],
  },
  {
    id: 4,
    name: "消息中心",
    path: "/message",
    icon: "notice",
    children: [
      {
        id: 41,
        name: "通知中心",
        path: "system/notice",
        component: "system/notice/index",
        icon: "NotificationsOutline",
      },
    ],
  },
  {
    id: 5,
    name: "日志管理",
    path: "/log",
    icon: "audit",
    children: [
      {
        id: 51,
        name: "操作日志",
        path: "monitor/operlog",
        component: "monitor/operlog/index",
        icon: "ListOutline",
      },
      {
        id: 52,
        name: "登录日志",
        path: "monitor/loginlog",
        component: "monitor/loginlog/index",
        icon: "LogInOutline",
      },
    ],
  },
  {
    id: 6,
    name: "系统监控",
    path: "/monitor",
    icon: "monitor",
    children: [
      {
        id: 61,
        name: "定时任务",
        path: "monitor/job",
        component: "monitor/job/index",
        icon: "TimerOutline",
      },
    ],
  },
  {
    id: 7,
    name: "代码生成",
    path: "/tool/generator",
    component: "tool/gen/index",
    icon: "codegen",
  },
];

const user = {
  id: 1,
  username: "visual-admin",
  nickname: "Visual Admin",
  displayName: "Visual Admin",
  deptName: "Platform",
  roles: ["admin"],
  permissions: [
    "system:user:list",
    "system:user:add",
    "system:user:edit",
    "system:user:remove",
    "system:user:resetPwd",
    "system:role:list",
    "system:role:add",
    "system:role:edit",
    "system:role:remove",
    "system:menu:add",
    "system:menu:edit",
    "system:menu:remove",
    "system:dept:add",
    "system:dept:edit",
    "system:dept:remove",
    "system:post:add",
    "system:post:edit",
    "system:post:remove",
    "system:dict:list",
    "system:dict:add",
    "system:dict:edit",
    "system:dict:remove",
    "system:config:list",
    "system:config:edit",
    "system:notice:add",
    "system:notice:remove",
    "monitor:job:list",
    "monitor:job:add",
    "monitor:job:edit",
    "monitor:job:remove",
    "monitor:job:run",
    "monitor:operlog:list",
    "monitor:loginlog:list",
    "tool:gen:list",
  ],
};

const userRow = {
  id: 1,
  username: "visual-admin",
  nickname: "Visual Admin",
  deptName: "Platform",
  postName: "Administrator",
  roleName: "Super Administrator",
  phone: "13800138000",
  email: "visual@example.test",
  status: 1,
  createTime: "2026-11-08 10:00:00",
  lastLogin: "2026-11-08 20:30:00",
};

const roleRow = {
  id: 1,
  name: "Super Administrator",
  code: "admin",
  dataScope: "全部数据权限",
  sort: 1,
  status: 1,
  builtin: true,
  remark: "Built-in administrator",
  userCount: 1,
  createTime: "2026-11-08 10:00:00",
  menuIds: [],
};

const menuTree = [
  {
    id: 1,
    parentId: 0,
    name: "系统管理",
    type: "dir",
    path: "/system",
    component: "",
    permission: "",
    icon: "SettingsOutline",
    sort: 1,
    visible: 1,
    status: 1,
    children: [
      {
        id: 2,
        parentId: 1,
        name: "用户管理",
        type: "menu",
        path: "/system/user",
        component: "system/user/index",
        permission: "system:user:list",
        icon: "PersonOutline",
        sort: 1,
        visible: 1,
        status: 1,
      },
      {
        id: 3,
        parentId: 1,
        name: "角色管理",
        type: "menu",
        path: "/system/role",
        component: "system/role/index",
        permission: "system:role:list",
        icon: "PeopleOutline",
        sort: 2,
        visible: 1,
        status: 1,
      },
    ],
  },
];

const deptTree = [
  {
    id: 1,
    parentId: 0,
    deptName: "Mars Platform",
    name: "Mars Platform",
    code: "platform",
    leader: "Visual Admin",
    phone: "13800138000",
    email: "platform@example.test",
    sort: 1,
    status: 1,
    createTime: "2026-11-08 10:00:00",
    children: [
      {
        id: 2,
        parentId: 1,
        deptName: "Engineering",
        name: "Engineering",
        code: "engineering",
        leader: "Engineer",
        sort: 1,
        status: 1,
        createTime: "2026-11-08 10:00:00",
      },
    ],
  },
];

const page = (list) => ({
  list,
  records: list,
  items: list,
  rows: list,
  total: list.length,
  page: 1,
  pageNum: 1,
  pageSize: 20,
});

const dashboardSummary = {
  stats: [
    { key: "online", label: "在线用户", value: 3, sub: "最近 1 小时登录" },
    { key: "api", label: "接口调用", value: 35, sub: "最近 24 小时操作" },
  ],
  trend: [2, 4, 3, 6, 5, 7, 8],
  activities: [
    { who: "Visual Admin", target: "admin", time: "20:30", tint: "ok" },
    { who: "Operator", target: "web", time: "20:20", tint: "ok" },
  ],
  roleDist: [
    { label: "ALL", value: 2 },
    { label: "SELF", value: 10 },
  ],
  trendTotal: 35,
  successRate: 99.2,
  trendPrev: 30,
  dailyOps: [
    { date: "2026-11-02", total: 3, failed: 0 },
    { date: "2026-11-03", total: 5, failed: 0 },
    { date: "2026-11-04", total: 4, failed: 1 },
    { date: "2026-11-05", total: 7, failed: 0 },
    { date: "2026-11-06", total: 6, failed: 0 },
    { date: "2026-11-07", total: 8, failed: 0 },
    { date: "2026-11-08", total: 9, failed: 0 },
  ],
};

const operationLog = {
  id: 1,
  title: "用户管理",
  businessType: 1,
  method: "system.users",
  requestMethod: "GET",
  operName: "Visual Admin",
  operUrl: "/api/system/users",
  operIp: "127.0.0.1",
  status: 0,
  operTime: "2026-11-08 20:30:00",
  costTime: 18,
};

const loginLog = {
  id: 1,
  username: "visual-admin",
  ipaddr: "127.0.0.1",
  loginLocation: "Local",
  browser: "Chrome",
  os: "Windows",
  status: 0,
  msg: "",
  loginTime: "2026-11-08 20:30:00",
};

function envelope(data) {
  return JSON.stringify({ code: 200, message: "ok", msg: "ok", data });
}

function responseFor(pathname) {
  if (pathname === "/api/auth/me") return envelope(user);
  if (pathname === "/api/auth/routes") return envelope({ routes: routeTree });
  if (pathname === "/api/dashboard/summary") return envelope(dashboardSummary);
  if (pathname === "/api/monitor/oper-logs/timeline") {
    return envelope({ nowSec: 73800, lanes: [] });
  }
  if (pathname === "/api/vuln/statistics/summary") {
    return envelope({ totalAssets: 0, openVulns: 0, highRiskOpen: 0, weekTasks: 0 });
  }
  if (pathname === "/api/vuln/statistics/vulnByMonth") return envelope([]);
  if (pathname === "/api/system/users/options") {
    return envelope({
      roles: [{ id: 1, name: "Super Administrator" }],
      posts: [{ id: 1, name: "Administrator" }],
      depts: deptTree,
      initPassword: "Init-password-123",
    });
  }
  if (pathname === "/api/system/depts/tree") return envelope(deptTree);
  if (pathname === "/api/system/menus/tree") return envelope(menuTree);
  if (pathname === "/api/system/users") return envelope(page([userRow]));
  if (pathname === "/api/system/roles") return envelope(page([roleRow]));
  if (pathname === "/api/system/posts") {
    return envelope(
      page([
        {
          id: 1,
          postCode: "admin",
          postName: "Administrator",
          sort: 1,
          status: 1,
          remark: "",
          createTime: "2026-11-08 10:00:00",
        },
      ]),
    );
  }
  if (pathname === "/api/system/dict/types") {
    return envelope(
      page([
        {
          id: 1,
          dictName: "System Status",
          dictType: "sys_status",
          status: 1,
          remark: "Status values",
          createTime: "2026-11-08 10:00:00",
        },
      ]),
    );
  }
  if (pathname === "/api/system/configs") {
    return envelope(
      page([
        {
          id: 1,
          groupCode: "system",
          groupName: "System Settings",
          sort: 1,
          status: 1,
          remark: "Core settings",
        },
      ]),
    );
  }
  if (pathname === "/api/notices/mine" || pathname === "/api/notices/sent") {
    return envelope(
      page([
        {
          id: 1,
          title: "Maintenance notice",
          content: "Scheduled maintenance",
          noticeType: 1,
          status: 1,
          createTime: "2026-11-08 10:00:00",
        },
      ]),
    );
  }
  if (pathname === "/api/notices/unread-count") return envelope({ count: 1 });
  if (pathname === "/api/monitor/jobs") {
    return envelope(
      page([
        {
          id: 1,
          jobName: "Health Probe",
          jobGroup: "DEFAULT",
          invokeTarget: "system.health_probe",
          cronExpression: "0 */5 * * * *",
          status: 1,
          remark: "",
          createTime: "2026-11-08 10:00:00",
        },
      ]),
    );
  }
  if (pathname === "/api/monitor/job-logs") return envelope(page([]));
  if (pathname === "/api/monitor/oper-logs") return envelope(page([operationLog]));
  if (pathname === "/api/monitor/oper-logs/stats") {
    return envelope({
      totalCount: 35,
      successCount: 35,
      failCount: 0,
      dailyStats: {},
      topPaths: [],
    });
  }
  if (pathname === "/api/monitor/login-logs") return envelope(page([loginLog]));
  if (pathname === "/api/monitor/login-logs/stats") {
    return envelope({ totalCount: 35, successCount: 35, failCount: 0, dailyStats: {} });
  }
  if (pathname === "/api/monitor/login-logs/heatmap") {
    return envelope({
      days: 7,
      since: "2026-11-02T00:00:00Z",
      threshold: 5,
      cells: [],
      anomalies: [],
    });
  }
  if (pathname === "/api/tool/generator/tables") {
    return envelope(
      page([
        {
          id: 1,
          tableName: "access_menus",
          tableComment: "Menu catalog",
          className: "AccessMenus",
          packageName: "access",
          moduleName: "access",
          businessName: "menus",
          functionName: "menu",
          author: "Mars",
          genType: "crud",
          createTime: "2026-11-08 10:00:00",
          updateTime: "2026-11-08 10:00:00",
        },
      ]),
    );
  }
  if (pathname === "/api/tool/generator/importable-tables") return envelope([]);
  if (pathname === "/api/system/permissions/graph") {
    return envelope({ nodes: [], links: [] });
  }
  return envelope(page([]));
}

await mkdir(outputDir, { recursive: true });
for (const file of await readdir(outputDir)) {
  if (file.startsWith("failure-") && file.endsWith(".png")) {
    await unlink(path.join(outputDir, file));
  }
}
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 1,
});
const browserPage = await context.newPage();
await browserPage.emulateMedia({ reducedMotion: "reduce" });
await browserPage.addInitScript(
  ({ authToken, authUser }) => {
    localStorage.setItem("scaffold.token", authToken);
    localStorage.setItem("scaffold.user", JSON.stringify(authUser));
    localStorage.setItem("scaffold.expiresAt", String(Date.now() + 3600_000));
    sessionStorage.setItem("scaffold.login-intro", "1");
  },
  { authToken: token, authUser: user },
);
await browserPage.route(/^http:\/\/127\.0\.0\.1:5173\/api\//, async (route) => {
  const request = route.request();
  const url = new URL(request.url());
  await route.fulfill({
    status: 200,
    contentType: "application/json",
    body: responseFor(url.pathname),
  });
});

const records = [];
const consoleErrors = [];
browserPage.on("console", (message) => {
  if (message.type() === "error") consoleErrors.push(message.text());
});
browserPage.on("pageerror", (error) => consoleErrors.push(error.message));

try {
  for (const scenario of scenarios) {
    await browserPage.addInitScript((appearance) => {
      localStorage.setItem("scaffold.appearance", JSON.stringify(appearance));
    }, scenario.appearance);
    for (const pageDef of pages) {
      const errorsBefore = consoleErrors.length;
      await browserPage.setViewportSize(scenario.viewport);
      await browserPage.goto(`${vueUrl}${pageDef.path}`, { waitUntil: "networkidle" });
      try {
        await browserPage.locator(".app-frame").waitFor();
      } catch (error) {
        await browserPage.screenshot({
          path: path.join(outputDir, `failure-${pageDef.name}-${scenario.name}.png`),
          fullPage: true,
        });
        console.error(`Vue page failed: ${pageDef.path}`, consoleErrors);
        throw error;
      }
      await browserPage.waitForTimeout(350);
      const fileName = `${pageDef.name}-${scenario.name}.png`;
      await browserPage.screenshot({
        path: path.join(outputDir, fileName),
        fullPage: true,
        animations: "disabled",
      });
      const metrics = await browserPage.evaluate(() => ({
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
      records.push({
        page: pageDef.name,
        scenario: scenario.name,
        fileName,
        pageErrors: consoleErrors.slice(errorsBefore),
        ...metrics,
      });
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
      reference: vueUrl,
      generatedAt: new Date().toISOString(),
      consoleErrors,
      records,
    },
    null,
    2,
  )}\n`,
);
console.log(`Captured ${records.length} Phase 3 Vue reference screenshots in ${outputDir}`);
