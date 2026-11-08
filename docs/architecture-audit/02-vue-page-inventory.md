# 02 Vue Page Inventory

> Historical Vue reference inventory. Asset Management, Vulnerability Management and
> Remediation Work Orders are excluded from the target product and are not migration backlog.

## 1. Route Model

静态路由来自 `scaffold-vue/src/router/index.ts`：

| 路径 | 名称 | 页面 | 说明 |
| --- | --- | --- | --- |
| `/login` | `login` | `views/Login.vue` | 公开登录页 |
| `/` | `root` | `AppShell.vue` | 受保护外壳 |
| `/index` | redirect | `/dashboard` | 旧平台首页书签兼容 |
| `/no-access` | `no-access` | `views/NoAccess.vue` | 无菜单账号 |
| `/:pathMatch(.*)*` | `not-found` | `views/NotFound.vue` | 未匹配路由 |

登录后的动态路由来自后端 `/auth/routes`，由 `composables/auth.ts` 的 `COMPONENT_MAP` 映射。当前注册了 16 个业务页面组件，连同登录、无权限、未找到共 19 个视图。

## 2. Complete Page Matrix

| # | 页面 | 路由/组件标识 | 文件 | 版式与核心组件 | 交互 | API | 权限 | 动效/浏览器能力 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 登录 | `/login` | `views/Login.vue` | 全幅双栏；`LoginBrandPanel`、`NeuralNet`、`Button`、`Field`、`TextInput`、`Checkbox` | 验证码刷新、账号密码校验、记住登录、成功过渡、重定向 | `GET /auth/captcha`、`POST /auth/login`、`GET /auth/me`、`GET /auth/routes` | 公开 | GSAP intro timeline、卡片指针倾斜、成功圆环；`localStorage`、`sessionStorage`、`matchMedia`、`visibilitychange` |
| 2 | 工作台 | `/dashboard` / `dashboard/index` | `views/Dashboard.vue` | 12 列 bento；安全态势、趋势、柱状、刻度环、泳道、最近登录、常用入口 | 时间范围、刷新、泳道跳转、安全态势月份选择、快捷入口 | `GET /dashboard/summary`、`GET /monitor/oper-logs/timeline`、`GET /vuln/statistics/summary`、`GET /vuln/statistics/vulnByMonth` | `monitor:operlog:list`、`vuln:vulnerability:list` 控制可选区块 | GSAP 首次进入卡片弹簧飞入、`TrackGlow`；`requestAnimationFrame` |
| 3 | 用户管理 | `/system/user` / `system/user/index` | `views/system/Users.vue` | `ListPage` + 左部门树 + 表格/看板；`DeptTree`、`BoardView`、`HoverCard`、`FormDialog` | 查询、部门筛选、看板分组、拖拽状态、CRUD、批量启停/删除、重置密码、导出、权限溯源 | `/system/users`、`/system/users/options`、`/system/users/{id}/status`、`/system/users/{id}/password`、`/system/users/export`、`/system/permissions/graph` | `system:user:*`、`system:role:list` | `useViewMode`、`ResizeObserver`、看板 GSAP/spring、`HoverCard`；`localStorage` |
| 4 | 角色管理 | `/system/role` / `system/role/index` | `views/system/Roles.vue` | `ListPage` + 表格/权限矩阵/溯源图谱；`PermissionMatrix`、`PermissionTrace`、`PermissionTree`、`Drawer` | 查询、角色 CRUD、菜单授权、数据范围、矩阵批量保存、未保存改动拦截、URL `?trace=` 深链 | `/system/roles`、`/system/roles/{id}/menus`、`/system/menus/tree`、`/system/roles/export`、`/system/permissions/graph` | `system:role:*` | GSAP `revealCascade`、图谱 Canvas、`RelationGraph`；键盘路径 |
| 5 | 菜单管理 | `/system/menu` / `system/menu/index` | `views/system/Menus.vue` | `ListPage` 树表/导航星图；`MenuStarMap`、`PermissionButton`、`FormDialog` | 树查询、CRUD、显示/隐藏、权限标识校验、侧栏预览、孤立菜单提示 | `/system/menus/tree`、`/system/menus`、`/system/menus/{id}`、`/system/menus/export`、`/system/permissions/graph` | `system:menu:*`、`system:role:list` | 图谱舞台、Canvas、`RevealCascade`、`Flip`；键盘 Esc/方向交互 |
| 6 | 部门管理 | `/system/dept` / `system/dept/index` | `views/system/Depts.vue` | 组织画布/表格；`OrgCanvas`、`DeptSelect`、`UserSelect`、`FormDialog` | 树选择、画布平移缩放、成员预览、CRUD、部门状态 | `/system/depts/tree`、`/system/depts`、`/system/depts/{id}`、`/system/users?deptId=`、`/system/depts/export` | `system:dept:*` | GSAP 视口 `quickTo`、画布帧循环、`PointerEvent`、滚轮缩放 |
| 7 | 岗位管理 | `/system/post` / `system/post/index` | `views/system/Posts.vue` | 表格/岗位阶梯；`PostLadder`、`FormDialog` | 查询、CRUD、批量删除、成员加载、编辑入口 | `/system/posts`、`/system/posts/{id}`、`/system/posts/export`、`/system/users?postId=` | `system:post:*` | `PostLadder` 图谱舞台、Canvas/GSAP 脉冲、`useViewMode` |
| 8 | 字典管理 | `/system/dict` / `system/dict/index` | `views/system/Dict.vue` | 样本册/表格 + 数据抽屉；`DictSwatchbook`、`DataTable`、`Drawer` | 类型 CRUD、数据项 CRUD、默认项、色调、状态、缓存失效、样本册原地展开 | `/system/dict/types`、`/system/dict/data/{type}`、`/system/dict/options/{type}`、`/system/dict/types/export` | `system:dict:*` | `Flip`、`revealCascade`、`settlePop`、CSS transitions |
| 9 | 系统配置 | `/system/config` / `system/config/index` | `views/system/Configs.vue` | 设置面板/表格；`FormDialog`、`Switch`、`SectionLabel` | 分组、行内编辑、批量差异浮条、逐项提交、AI 连通性测试、知识库重建 | `/system/configs`、`/system/configs/{id}`、`/system/configs/export`、`/vuln/ai/test`、`/vuln/ai/knowledge/status`、`/vuln/ai/knowledge/rebuild` | `system:config:*`、`vuln:ai:manage` | `floatBarEnter/Leave`、`revealCascade`、`settlePop`、`scrollIntoView`；`useViewMode` |
| 10 | 通知中心 | `/system/notice` / `system/notice/index` | `views/system/Notice.vue` | `ListPage` + 我的通知/发送记录 + 详情抽屉；`UserMultiSelect`、`DictTag` | 未读筛选、已读/全部已读、发送公告或指定用户、删除、详情 | `/notices/mine`、`/notices/sent`、`/notices/unread-count`、`/notices/{id}/read`、`/notices/read-all`、`/notices`、`/notices/{id}` | `system:notice:add`、`system:notice:remove` | 抽屉、`localStorage` 视图偏好、GSAP 抽屉共享元素 |
| 11 | 定时任务 | `/monitor/job` / `monitor/job/index` | `views/monitor/Job.vue` | `ListPage` 表格 + 执行日志 alt 视图；`CronBuilder`、`Drawer`、`DataTable` | 查询、任务 CRUD、暂停/恢复、立即执行、日志查询/清空、cron 校验 | `/monitor/jobs`、`/monitor/jobs/{id}`、`/monitor/jobs/{id}/run`、`/monitor/cron-check`、`/monitor/job-logs` | `monitor:job:*` | `CronBuilder` 向导、`localStorage` 视图、Drawer/GSAP |
| 12 | 操作日志 | `/monitor/operlog` / `monitor/operlog/index` | `views/monitor/OperLog.vue` | 横带：概览、趋势/泳道、异常、明细；`TrendChart`、`Swimlane`、`Drawer` | 时间范围、趋势/泳道切换、时间块跳转关键词、详情抽屉、删除/清空/导出 | `/monitor/oper-logs`、`/monitor/oper-logs/stats`、`/monitor/oper-logs/timeline`、`/monitor/oper-logs/export` | `monitor:operlog:list`、`monitor:operlog:remove` | `RevealCascade`、GSAP 图表、`scrollIntoView` |
| 13 | 登录日志 | `/monitor/loginlog` / `monitor/loginlog/index` | `views/monitor/LoginLog.vue` | 横带：概览、趋势/热力、异常、明细；`TrendChart`、`HeatGrid` | 时间范围、热力 7/30 天、异常筛选、明细抽屉、删除/清空/导出 | `/monitor/login-logs`、`/monitor/login-logs/stats`、`/monitor/login-logs/heatmap`、`/monitor/login-logs/export` | `monitor:loginlog:list`、`monitor:loginlog:remove` | `HeatGrid` GSAP stagger、键盘方向键、`scrollIntoView` |
| 14 | 代码生成 | `/tool/generator` / `tool/gen/index` | `views/tool/Generator.vue` | 表格 + 导入弹窗 + 三步流水线；`GeneratorPipeline`、`RelationGraph`、`Modal` | 查询、导入数据库表、表说明、字段映射、代码预览、生成 ZIP 下载 | `/tool/generator/tables`、`/tool/generator/importable-tables`、`/tool/generator/import`、`/tool/generator/tables/{id}/columns`、`/preview`、`/generate`、`/download` | `tool:gen:list` | GSAP 阶段切换、逐行级联、关系图脉冲；`Blob`/下载 |
| 15 | 资产管理 | `/vuln/asset` / `vuln/asset/index` | `views/vuln/Assets.vue` | `ListPage` 表格 + 详情抽屉；`DictSelect`、`DictTag` | 查询、资产 CRUD、批量删除、导出、详情、关联漏洞 | `/vuln/assets`、`/vuln/assets/{id}`、`/vuln/assets/export`、`/vuln/vulnerabilities?assetId=` | `vuln:asset:*` | `fitViewport`、抽屉、`FormDialog`、字段校验 |
| 16 | 漏洞管理 | `/vuln/vuln` / `vuln/vuln/index` | `views/vuln/Vulns.vue` | `ListPage` 表格 + 详情抽屉；资产/状态标签、关联工单 | 查询、漏洞 CRUD、批量删除、导出、详情、关联修复任务 | `/vuln/vulnerabilities`、`/vuln/assets`、`/vuln/repair-tasks?vulnId=`、`/vuln/vulnerabilities/export` | `vuln:vulnerability:*` | `fitViewport`、抽屉、`DictTag` |
| 17 | 修复工单 | `/vuln/repair` / `vuln/repair/index` | `views/vuln/Repairs.vue` | `ListPage` 表格 + 状态动作 + 复测弹窗；`UserSelect`、`DictTag` | 查询、创建/编辑/删除、接单、提交修复、复测通过/驳回、关闭、导出 | `/vuln/repair-tasks`、`/vuln/repair-tasks/{id}/accept`、`/fix`、`/verify`、`/close`、`/vuln/vulnerabilities` | `vuln:repair:*` | `fitViewport`、表单校验、状态动作确认、`UserSelect` |
| 18 | 无菜单 | `/no-access` | `views/NoAccess.vue` | `EmptyState` | 提示账号无菜单、返回/退出路径 | 无独立业务 API | 已登录 | 静态空状态 |
| 19 | 未找到 | `/:pathMatch(.*)*` | `views/NotFound.vue` | `EmptyState`、`Button` | 返回首页/上一页 | 无 | 已登录 | 静态 |

## 3. API Dependency Families

| API 文件 | 负责能力 | 当前前缀 |
| --- | --- | --- |
| `api/auth.ts` | captcha、登录、当前用户、路由、改密、退出 | `/auth/*` |
| `api/system.ts` | 用户、角色、菜单、部门、岗位、字典、配置、任务、通知、日志 | `/system/*`、`/monitor/*`、`/notices/*` |
| `api/portal.ts` | 健康、工作台、代码生成 | `/health`、`/dashboard/*`、`/tool/generator/*` |
| `api/vuln.ts` | 资产、漏洞、修复工单、统计、AI | `/vuln/*` |
| `api/session.ts` | token/user/expiry 存储 | 浏览器存储 |
| `api/http.ts` | envelope、Bearer、401、Blob 下载 | `/api` |

## 4. Permission Codes in Source

当前源码实际出现的业务权限码：

```text
system:user:add / edit / list / remove / resetPwd
system:role:add / edit / list / remove
system:menu:add / edit / remove
system:dept:add / edit / remove
system:post:add / edit / remove
system:dict:add / edit / list / remove
system:config:add / edit / list / remove
system:notice:add / remove
monitor:job:add / changeStatus / edit / remove / run
monitor:operlog:list / remove
monitor:loginlog:list / remove
tool:gen:list
vuln:asset:add / edit / list / remove
vuln:vulnerability:add / edit / list / remove
vuln:repair:add / edit / remove
vuln:ai:manage
```

Go 后端当前种子权限使用另一套命名，例如 `user:read`、`role:read`、`department:read`、`dictionary:read`、`settings:read`、`audit:read`、`message:read`、`notice:read`、`scheduler:read`、`workflow:read`、`codegen:read`。这是前端旧权限码与 Go 原生权限码的契约差异，不能通过“菜单可见”自动解决。

## 5. Page Gaps Against the Reference

### Present in `scaffold-vue`

- 登录、工作台、用户、角色、菜单、部门、岗位、字典、配置、通知。
- 定时任务、操作日志、登录日志、代码生成。
- Vue-only reference: 漏洞资产、漏洞、修复工单（EXCLUDED）。
- 全局 AI 助手和权限溯源/导航星系/组织画布/岗位阶梯等可视化版式。

### Not Present as Dedicated Pages in `scaffold-vue`

- 独立文件管理页。
- 独立即时聊天/群聊页。
- 独立工作流定义、待办、已办、监控页。
- 独立服务监控/缓存监控/在线用户页。

这些能力在 Go 后端存在部分实现，但前端参考当前没有对应页面。目标产品是否保留这些页面，必须在路线图中明确；不能把后端有 API 当作前端已迁移。

### Missing in Go Backend Relative to `scaffold-vue`

- 漏洞资产、漏洞、修复工单不属于 Go 目标范围，不要求补齐模块或 API。
- 没有与 `scaffold-vue` 的 `/system/*`、`/monitor/job/*`、`/tool/generator/*` 完全一致的原生契约。
- 没有 `scaffold-vue` 的 `/vuln/ai/*` 等价接口。
- 没有与 `scaffold-vue` 旧 envelope、权限码和 UUID/数字 ID 字段完全一致的适配层。

## 6. Browser Capability Inventory

| 能力 | 页面/组件 | 用途 | React 迁移注意 |
| --- | --- | --- | --- |
| `localStorage` | 登录提示、视图偏好、外观、最近访问、AI 会话 | 持久化个人 UI 状态 | 不得保存密码或长期 token；只保存非敏感 UI 偏好 |
| `sessionStorage` | 首次开场、登录开场 | 每会话一次 | 在 Client Component 中延迟读取，避免 SSR 访问 `window` |
| `matchMedia` | 主题、减少动效、悬停能力 | 主题/动效降级 | 使用 `useSyncExternalStore` 或 effect 监听变化 |
| Pointer Events | 登录卡倾斜、画布平移、拖拽看板 | 指针交互 | Client Component；清理 pointer capture 和监听 |
| `ResizeObserver` | 侧栏滑块、分段控件 | 尺寸同步 | effect 内创建并 disconnect |
| Canvas | `NeuralNet`、`RelationGraph`、`OrgCanvas` | 图谱/背景/画布 | 保持 Client Component，限制 DPR，暂停离屏帧 |
| `scrollIntoView` | 配置分组、日志详情 | 定位内容 | 保留 reduced-motion 分支 |
| Blob/URL download | 代码生成 ZIP、CSV 导出 | 文件下载 | 使用服务端 `Content-Disposition` 或受控 Blob |
| `navigator.clipboard` | 代码生成复制 | 复制预览 | 提供权限失败和键盘回退 |

## 7. Page-Level Acceptance Rule

页面只有同时满足以下条件，才可标记“迁移完成”：

1. 页面视觉与原 Vue 同条件截图一致或差异有书面批准。
2. 真实 Go API 返回成功、空、错误、未授权四类状态。
3. 查询、重置、分页、CRUD、批量操作和导出路径可用。
4. 权限按钮隐藏与后端 403 同时成立。
5. 主题、密度、窄屏、键盘和 reduced motion 有验证。
6. 没有用硬编码数据、假接口或静态占位页替代真实业务。
