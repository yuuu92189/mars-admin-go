# 04 Design System Audit

## 1. Source of Truth

前端设计系统不是 Tailwind 类名集合，而是以下七层 CSS 与自研组件共同组成：

| 顺序 | 文件 | 职责 |
| --- | --- | --- |
| 1 | `scaffold-vue/src/styles/tokens.css` | 颜色、字体、圆角、阴影、尺寸、主题、密度 |
| 2 | `scaffold-vue/src/styles/base.css` | reset、排版、焦点、颗粒层、无障碍降级 |
| 3 | `scaffold-vue/src/styles/materials.css` | 导航轨、玻璃块、浮层、阅读面、凹槽、指针光 |
| 4 | `scaffold-vue/src/styles/components.css` | 按钮、输入、表格、标签、弹窗、Toast |
| 5 | `scaffold-vue/src/styles/instruments.css` | 读数、铭牌、刻度、图表、泳道、画布、矩阵、设置面板 |
| 6 | `scaffold-vue/src/styles/layout.css` | 外壳、侧栏、顶栏、页面、分栏、视口列表 |
| 7 | `scaffold-vue/src/styles/pages.css` | 登录、工作台、生成预览等页面差异 |

`main.ts` 按上述顺序引入；根目录 `DESIGN.md` 是该体系的书面规范。

## 2. Design Tokens

### 2.1 Color

| 令牌 | 浅色 | 深色 | 语义 |
| --- | --- | --- | --- |
| `--ink` | `#101418` | `#eef2f1` | 正文/标题 |
| `--ink-muted` | `#596368` | `#b0babd` | 次级正文 |
| `--ink-soft` | `#7e888c` | `#8d979a` | 标签/弱信息 |
| `--ink-faint` | `#99a2a5` | `#6c7679` | 元信息 |
| `--blue` | `#236de9` | `#4c8dff` | 选择、主操作、焦点 |
| `--leaf-text` | `#58a021` | `#9be35a` | 正常/能力文字 |
| `--red-text` | `#e0443b` | `#ff7a71` | 警示/危险文字 |
| `--stage` | `#dfe3e2` | `#090b0c` | 外壳舞台 |
| `--canvas` | `#f6f8f7` | `#121517` | 工作区画布 |
| `--surface-solid` | `#fbfcfc` | `#1b1f21` | 阅读面实色 |
| `--plate-bg` | 深色玻璃 | 深色玻璃 | 规格铭牌/浮条 |

分类色 `--cat-1` 至 `--cat-5`（沙、天、丁香、玫瑰、薄荷）只表达类别，不表达状态；成功、警告、危险必须另有文字/图标。

### 2.2 Type

| 令牌/规则 | 值/行为 |
| --- | --- |
| 正文 | 系统字体栈：SF Pro、PingFang SC、Noto Sans SC、Microsoft YaHei |
| 等宽 | `--font-mono`：SF Mono、JetBrains Mono、Menlo、Consolas |
| 显示读数 | `--fs-display-xl` 56px、`--fs-display-l` 40px、`--fs-display-m` 28px |
| 眉标 | `--track-eyebrow: .14em` |
| 正文基线 | 13px / 1.5 |
| 数字 | `font-variant-numeric: tabular-nums` |
| 文案 | 业务中文优先；等宽只用于数字、编码、权限标识和时间 |

### 2.3 Geometry

| 令牌 | 值 |
| --- | --- |
| `--r-xs` / `--r-sm` / `--r-md` / `--r-lg` / `--r-xl` | 8 / 12 / 16 / 20 / 24px |
| `--rail-w` | 232px；窄屏 208px；折叠 76px；移动端抽屉 260px |
| `--topbar-h` | 64px |
| `--control-h` / `--control-h-sm` | 36 / 30px |
| `--row-h` | regular 48px；comfy 56px；compact 40px |
| `--cell-px` | regular 16px；comfy 18px；compact 14px |
| `--layer-overlay` / `--layer-toast` | 200 / 400 |

### 2.4 Elevation

| 令牌 | 用途 |
| --- | --- |
| `--e1` | 阅读面接触阴影 |
| `--e2` | 高亮玻璃块/浮层 |
| `--e3` | 高层浮层/命令面板 |
| `--e-accent` / `--e-accent-hover` | 主操作按钮 |
| `--e-danger` | 危险操作 |
| `--edge` / `--edge-strong` | 1px 内高光 |

## 3. Pearl Glass Material System

### 3.1 Four Recipes

| 类名 | 背景 | 模糊/饱和 | 使用位置 |
| --- | --- | --- | --- |
| `.m-rail` | `--rail-bg` | blur 22px | 侧栏导航轨 |
| `.m-pill` | `--pill-bg` + 玻璃白边 | blur 22px + saturate 1.18 | 选中导航、分段滑块、次级按钮 |
| `.m-float` | `--float-bg` + 强内高光 | blur 28px + saturate 1.08 | 下拉、命令面板、气泡、Toast |
| `.m-surface` | `--surface`，浅色态不做 backdrop blur | 接近实色 | 表格、卡片、弹窗、阅读面 |

`.m-groove` 是凹槽底座，`.m-glow` + `.glow-layer` 是容器级指针光。玻璃只承担层级，不牺牲正文和表格可读性。

### 3.2 Pearl Background

`.app-frame` 使用两层静态径向环境光：

```css
radial-gradient(55% 45% at 92% 0%, var(--ambient-a), transparent 70%),
radial-gradient(45% 40% at 20% 100%, var(--ambient-b), transparent 70%),
var(--canvas)
```

浅色环境光来自青柠与蓝色低透明漫射；深色主题保持同一语义。`base.css` 的 `.grain-layer` 用低透明 SVG 噪点打散大面积渐变色带，静态、不接指针、不参与动画。

### 3.3 Glass Details

- 阅读面：1px `--line-strong` 边线、`--edge` 内高光、`--e1` 接触阴影。
- 浮层：玻璃白边、内高光、`--e3` 环境阴影。
- 高亮块：白色玻璃边最亮，用于选中态而不是大面积装饰。
- 选中/危险/停用必须同时有文字或图标，不能仅靠色相。
- 深色主题只替换语义令牌与材质透明度，不重建布局。

## 4. Themes and Density

`composables/appearance.ts`：

- 主题：`light`、`dark`、`auto`；写入根节点 `data-theme`。
- 密度：`comfy`、`regular`、`compact`；写入 `data-density`。
- 导航折叠：`railCollapsed`。
- 存储键：`scaffold.appearance`；账号切换只清会话数据，不清外观偏好。

初始主题在应用挂载前应用，避免首屏闪色。系统 `prefers-color-scheme` 变化只在 `auto` 模式触发。

## 5. Responsive Rules

| 断点/条件 | 行为 |
| --- | --- |
| `max-width: 1180px` | 侧栏 208px；登录双栏开始收缩 |
| `max-width: 1080px` | 命令入口图标化；用户元信息隐藏；左树右表改为单列 |
| `max-width: 960px` | 登录页改为单列 |
| `max-width: 900px` | 设置面板/生成流水线改为单列 |
| `max-width: 820px` | 外壳全屏、侧栏抽屉、页面 padding 收窄、表单字段全宽 |
| `max-width: 760px` | 工作台、生成预览、字典样本册等改为窄屏布局 |
| `max-width: 720px` | 表单网格/键值网格单列；图表局部压缩 |
| `min-width: 1024px` and `min-height: 700px` | `fitViewport` 列表固定标题/查询/分页，仅表格数据区滚动 |
| `prefers-reduced-motion: reduce` | 动画时长接近 0；GSAP 走终态；帧循环暂停 |
| `prefers-reduced-transparency: reduce` | 玻璃材质改为实色阅读面 |
| `prefers-contrast: more` | 关闭颗粒层 |
| `pointer: coarse` | 关闭指针光 |

页面整体不得横向溢出；表格在自身容器内横向滚动；操作列吸附右侧。

## 6. Layout Patterns

| 版式 | 页面 | 结构 |
| --- | --- | --- |
| 全幅画布 + 登录卡 | 登录 | 左侧品牌/流程/终端，右侧玻璃登录卡 |
| 主次 bento | 工作台 | 8 列主卡 + 4 列侧块；下排柱状、刻度环、权限占比；底部泳道/最近登录/快捷入口 |
| 横带分层 | 操作日志、登录日志 | 概览读数 → 趋势/泳道/热力 → 异常 → 明细 |
| 左树右表 | 用户 | 部门树固定，表格/看板替代视图 |
| 表格/矩阵/图谱舞台 | 角色 | 列表、权限矩阵、权限溯源三视图 |
| 树表/导航星图 | 菜单 | 菜单树与侧栏预览/权限关系 |
| 组织画布/表格 | 部门 | 平移缩放画布 + 四角玻璃面板 |
| 阶梯舞台 | 岗位 | 岗位排序、成员胶囊、刻度地平线 |
| 样本册/表格 | 字典 | 类型卡片原地展开 + 数据抽屉 |
| 设置面板/表格 | 配置 | 分组导航、行内控件、差异浮条 |
| 三步流水线 | 代码生成 | 表信息 → 字段映射 → 代码逐行预览 |

## 7. React / Tailwind Migration

### Preserve CSS

以下能力不能全部改写成 Tailwind 工具类：

- `backdrop-filter` 玻璃配方、内高光、环境光和噪点层。
- `radial-gradient`、分类色、刻度线、斜线填充。
- Canvas/图谱舞台、共享元素 morph、复杂阴影。
- `@property --fill` 等需要连续补间的 CSS 自定义属性。

建议：

```text
src/styles/tokens.css
src/styles/base.css
src/styles/materials.css
src/styles/components.css
src/styles/instruments.css
src/styles/layout.css
src/styles/pages.css
```

Tailwind 负责布局、Flex/Grid、间距、响应式和常规状态；珍珠玻璃材质保留完整 CSS。CSS Variables 继续作为唯一令牌来源，并在 Tailwind theme 中映射颜色/间距/断点。

### React Boundaries

- `app/layout.tsx`：引入全局 CSS、主题初始化、字体和 metadata。
- `(admin)/layout.tsx`：Server Component 外层；Sidebar/Topbar/CommandPalette 等交互壳为 Client Component。
- 表格、筛选、分页、看板、画布、GSAP、Canvas、localStorage 均为 Client Component。
- 主题初始化脚本在 hydration 前读取本地偏好，避免闪烁；不要在每个组件重复读取。
- 复杂浮层统一使用 Portal + focus stack；不要用 DOM 直接拼 `z-index`。

### Visual Acceptance

建立 Playwright 截图基线，至少覆盖：

- 登录、工作台、用户、角色、菜单、部门、字典、配置、通知、日志、任务、代码生成、漏洞三页。
- 浅色/深色、regular/compact、桌面/窄屏。
- 表格/看板、列表/矩阵/图谱、抽屉、弹窗、Toast、空状态、错误状态。

截图相似度不能替代真实 API 流程；两者必须同时通过。
