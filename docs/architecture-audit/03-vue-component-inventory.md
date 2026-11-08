# 03 Vue Component Inventory

## 1. Component Conventions

- 所有组件位于 `scaffold-vue/src/components`，使用 Vue `<script setup lang="ts">`。
- 基础组件只处理展示、输入、焦点和事件，不把业务请求放进 UI 层。
- 业务组件通过 `@/api` 或业务 composable 读取数据；React 迁移时应改为注入 API client 或使用 feature hook，避免组件直接耦合全局单例。
- 公共样式主要来自 `styles/tokens.css`、`base.css`、`materials.css`、`components.css`、`instruments.css`、`layout.css`、`pages.css`。
- 只有 11 个 SFC 含局部 `<style scoped>`：`Login.vue`、`AiAssistant.vue`、`NeuralNet.vue`、`Job.vue`、`Notice.vue`、`NoticeBell.vue`、`CronBuilder.vue`、`BrandMark.vue`、`LoginBrandPanel.vue`、`UserMultiSelect.vue`；其余依赖全局层。

## 2. Component Counts

| 分类 | 数量 |
| --- | --- |
| `ui` 基础组件 | 27 |
| `business` 业务组件 | 24 |
| `layout` 外壳组件 | 7 |
| `charts` 图表/图谱组件 | 6 |
| `ai` 组件 | 1 |
| 根级组件 | 2 |
| 合计 | 67 |

## 3. Basic UI Components

这些组件是 React 自研组件库的第一批迁移对象。Props 和事件以源码 `defineProps`/`defineEmits` 为准。

| 组件 | Props / 事件 | 内部状态与交互 | 样式/动效 | React 映射 |
| --- | --- | --- | --- | --- |
| `Avatar.vue` | `name`, `size` | 首字母/名称稳定色 | 全局组件层 | `ui/avatar.tsx` |
| `Button.vue` | `variant`, `size`, `type`, `icon`, `block`, `danger`, `disabled`, `loading`；默认 slot | 禁用/加载/图标 | `.btn`、按压/涟漪 | `ui/button.tsx` |
| `Checkbox.vue` | `modelValue`, `label`, `disabled`, `indeterminate`；`update:modelValue` | 选中/半选、键盘 | `settlePop`、焦点环 | `ui/checkbox.tsx` |
| `ConfirmHost.vue` | 无 | 监听全局 confirm store，渲染确认框 | `Modal` 浮层 | `ui/confirm-host.tsx` + store |
| `DataTable.vue` | `columns`, `data`, `rowKey`, `selectable`, `selected`, `loading`, `emptyText`, `minWidth`, `scrollable`, `tree`, `expandDepth`；selection 事件、动态 `cell-*` slot | 全选/半选、树展开、吸顶表头 | `.table`、固定操作列、横向滚动 | `ui/data-table.tsx` |
| `DeltaPill.vue` | `value`, `up`, `fresh`, `label`, `suffix` | 方向/新增/无变化 | 状态色、tabular numbers | `ui/delta-pill.tsx` |
| `Drawer.vue` | `open`, `title`, `sub`, `width`, `originFrom`；`close` | 焦点、Escape、共享元素来源 | `overlayEnter/Leave`、GSAP morph | `ui/drawer.tsx` |
| `EmptyState.vue` | `title`, `desc`, `icon`；默认 slot | 空状态 | 阅读面、图标 | `ui/empty-state.tsx` |
| `Field.vue` | `label`, `required`, `error`, `hint`, `wide`, `span`, `group`；默认 slot | 标签、错误、提示 | 表单网格 | `ui/field.tsx` |
| `HoverCard.vue` | `delay`, `placement`, `width`；默认 slot、`card` slot | hover/focus/Escape/定位 | 浮层、`focus.ts` | `ui/hover-card.tsx` |
| `IconButton.vue` | `name`, `label`, `size`, `small`, `bordered`, `danger`, `disabled` | 图标按钮可访问名称 | `.icon-btn` | `ui/icon-button.tsx` |
| `Metric.vue` | `value`, `unit`, `label`, `size`, `decimals`；默认 slot | 读数格式化 | 仪器化读数 | `ui/metric.tsx` |
| `Modal.vue` | `open`, `title`, `sub`, `width`, `closeOnOverlay`；`close` | 焦点、Escape、overlay | `overlayEnter/Leave` | `ui/modal.tsx` |
| `Odometer.vue` | `value`, `decimals`, `group` | 数字滚动、后台兜底 | `requestAnimationFrame` | `ui/odometer.tsx` |
| `PageHeader.vue` | `title`, `desc`；默认/side slot | 标题、说明、操作区 | `.page-head` | `ui/page-header.tsx` |
| `Pagination.vue` | `page`, `pageSize`, `total`, `sizes`；`update:page`、`update:page-size` | 页码/每页条数 | 底部工具行 | `ui/pagination.tsx` |
| `Panel.vue` | `title`, `meta`；默认、`head`、`tools` slot | 面板头/工具 | `.panel`、阅读面 | `ui/panel.tsx` |
| `Ruler.vue` | `marker`, `markerLabel`, `labels`, `minor` | 刻度与标记 | SVG/CSS | `ui/ruler.tsx` |
| `SectionLabel.vue` | `title`, `as` | 中文区块标题 | 青柠点、排版 | `ui/section-label.tsx` |
| `Segmented.vue` | `modelValue`, `options`, `label`；`update:modelValue` | 键盘选择、滑块 | `ResizeObserver` + `slideThumb` | `ui/segmented.tsx` |
| `Select.vue` | `modelValue`, `options`, `placeholder`, `clearable`, `disabled`, `filterable`, `emptyText`, `invalid`；`update:modelValue` | 打开/过滤/方向键/Enter/Escape/焦点 | 浮层、`focus.ts` | `ui/select.tsx` |
| `SpecPlate.vue` | `title`, `rows` | 规格铭牌 | 深色铭牌 | `ui/spec-plate.tsx` |
| `StatusTag.vue` | `status`, `on`, `off` | 状态文字/色调 | 状态标签 | `ui/status-tag.tsx` |
| `Switch.vue` | `modelValue`, `label`, `disabled`；`update:modelValue` | 开关、键盘 | CSS custom property 过渡 | `ui/switch.tsx` |
| `Tag.vue` | `tone`, `dot`；默认 slot | 分类/状态标签 | `.tag` | `ui/tag.tsx` |
| `TextInput.vue` | `modelValue`, `type`, `placeholder`, `icon`, `clearable`, `disabled`, `readonly`, `invalid`, `maxlength`, `multiline`, `rows`, `autocomplete`；`update:modelValue`、`enter` | 输入、清空、Enter | `.input-shell`、焦点 | `ui/text-input.tsx` |
| `ToastHost.vue` | 无 | 全局 toast 队列、暂停/展开 | GSAP in/out、倒计时环 | `ui/toast-host.tsx` |

## 4. Business Components

| 组件 | Props / 事件 | 内部状态与交互 | 样式/动效 | React 映射 |
| --- | --- | --- | --- | --- |
| `AskDock.vue` | `group?`, `placeholder?`, `compact?`；`chip` | 本地规则问答、逐字输入、答案卡级联 | `typeOut`、`revealCascade` | `features/assistant/ask-dock.tsx` |
| `BoardView.vue` | `columns`, `itemKey`, `draggable`, `onMove`, `emptyText`；`card` slot | 拖拽、键盘移动、Ghost、列排序 | GSAP + spring + frame loop | `business/board-view.tsx` |
| `BrandMark.vue` | 无 | 品牌图标 | scoped CSS、深色块 | `business/brand-mark.tsx` |
| `CronBuilder.vue` | `open`, `value?`；`close`、`confirm` | 秒/分/时/日/月/周向导，日/周互斥 | Modal、CSS transitions | `business/cron-builder.tsx` |
| `DeptSelect.vue` | `modelValue`, `data?`, `placeholder`, `clearable`, `invalid`, `excludeIds`, `rootLabel`；`update:modelValue` | 部门选项、禁用排除 | 复用 `Select` | `business/dept-select.tsx` |
| `DeptTree.vue` | `modelValue`, `data?`；`update:modelValue` | 搜索、展开、选中/取消 | 左栏树、独立滚动 | `business/dept-tree.tsx` |
| `DictSelect.vue` | `modelValue`, `type`, `allLabel`, `placeholder`, `disabled`, `invalid`；`update:modelValue` | 字典缓存、加载选项 | `Select` | `business/dict-select.tsx` |
| `DictSwatchbook.vue` | `types`, `loading?`；`create-item`、`edit-item`、`edit-type` | 卡片展开、行内色调/默认/状态、Flip | `flipLayout`、`revealCascade`、`settlePop` | `business/dict-swatchbook.tsx` |
| `DictTag.vue` | `type`, `value` | 查字典并映射标签/色调 | `Tag` | `business/dict-tag.tsx` |
| `FormDialog.vue` | `open`, `title`, `sub`, `width`, `saving`, `submitText`；`close`、`submit` | 统一表单弹窗、footer | `Modal`/`Drawer` | `business/form-dialog.tsx` |
| `GeneratorPipeline.vue` | `table`, `step?`；`close`、`saved`、`generate` | 表信息、字段映射、代码预览 | GSAP、RelationGraph、逐行级联 | `features/codegen/generator-pipeline.tsx` |
| `InsightPanel.vue` | `title`, `subtitle`, `kind`, `kindIcon`, `cat`, `status`, `metric`, `tiles`, `groups`, `insights`, `contentKey`；`close`、`chip`；默认 slot | 分析栏、指标、chip | `revealCascade`、CSS | `business/insight-panel.tsx` |
| `ListPage.vue` | `title`, `desc`, `tableTitle`, `list`, `columns`, `selectable`, `fitViewport`, `altView`, `altFit`, `minWidth`, `rowKey`；多个 slots | 查询/表格/分页/批量/替代视图 | `layout.css`、`reveal` | `business/list-page.tsx` |
| `LoginBrandPanel.vue` | 无 | 四阶段轮播、终端打字、品牌说明 | scoped CSS、CSS blink、定时器 | `features/auth/login-brand-panel.tsx` |
| `MenuStarMap.vue` | `tree`；`edit`、`create` | 三层导航星图、权限图、侧栏预览、孤立项 | Canvas/RelationGraph、CSS | `features/access/menu-star-map.tsx` |
| `OrgCanvas.vue` | `tree`, `selectedId`；`select`；四角 slots | 组织树布局、平移缩放、脉冲路径 | GSAP、frame loop、Pointer Events | `features/organization/org-canvas.tsx` |
| `PermissionButton.vue` | `perm`, `variant`, `size`, `icon`, `label`, `iconOnly`, `danger`, `loading`, `disabled`；默认 slot | 按权限显示 | `Button` | `business/permission-button.tsx` |
| `PermissionMatrix.vue` | `menus`, `roles`, `grants`, `editable`, `saving`；`save` | 角色×菜单矩阵、未保存差异、折叠 | CSS、横向滚动 | `features/access/permission-matrix.tsx` |
| `PermissionTrace.vue` | `v-model:focus`；无显式 emits | 用户/角色/菜单三层下钻、面包屑、Esc | RelationGraph、InsightPanel、AskDock | `features/access/permission-trace.tsx` |
| `PermissionTree.vue` | `menus`, `modelValue`；`update:modelValue` | 父子联动、半选、折叠 | CSS tree | `business/permission-tree.tsx` |
| `PostLadder.vue` | `posts`, `loading?`；`edit` | 阶梯排序、成员加载、光柱 | CSS/Canvas 类舞台 | `features/organization/post-ladder.tsx` |
| `SearchForm.vue` | `loading?`；`search`、`reset`；默认 slot | 回车查询、重置 | `.search-form` | `business/search-form.tsx` |
| `UserMultiSelect.vue` | `modelValue`, `placeholder`；`update:modelValue` | 多选用户、过滤、选中态 | scoped CSS、浮层 | `business/user-multi-select.tsx` |
| `UserSelect.vue` | `modelValue`, `valueKey`, `placeholder`, `clearable`；`update:modelValue` | 用户选项加载 | `Select` | `business/user-select.tsx` |

## 5. Layout Components

| 组件 | Props / 事件 | 状态与交互 | React 映射 |
| --- | --- | --- | --- |
| `AppearanceDialog.vue` | `open`；`close` | 主题、密度设置 | `layout/appearance-dialog.tsx` |
| `AppShell.vue` | 无 | 路由壳、移动侧栏、命令面板、外观/密码弹窗、滚动状态、AI 助手 | `app/(admin)/layout.tsx` + client shell |
| `CommandPalette.vue` | `open`；`close`、`action` | 最近访问、页面/用户/动作搜索、`?` 问答 | `layout/command-palette.tsx` |
| `NoticeBell.vue` | 无 | 未读计数、通知浮层、已读 | `layout/notice-bell.tsx` |
| `PasswordDialog.vue` | `open`；`close` | 旧/新/确认密码校验 | `layout/password-dialog.tsx` |
| `Sidebar.vue` | `navigate` | 导航分组、活动滑块、折叠、ResizeObserver | `layout/sidebar.tsx` |
| `Topbar.vue` | `toggle-rail`、`command`、`appearance`、`password`、`logout` | 面包屑、通知、用户菜单、键盘 | `layout/topbar.tsx` |

## 6. Chart and Graph Components

| 组件 | Props / 事件 | 行为 | React 映射 |
| --- | --- | --- | --- |
| `BarChart.vue` | `values`, `failed`, `labels`, `fullLabels`, `height`, `highlight`, `variant`, `unit`；`select` | SVG/HTML 柱、失败叠层、键盘/点击选择 | `charts/bar-chart.tsx` |
| `HeatGrid.vue` | `cells`, `loading?` | 7×24 热力格、对角线入场、十字高亮、方向键 | `charts/heat-grid.tsx` |
| `RelationGraph.vue` | `nodes`, `links`, `activeId`, `highlight`, `label`, `network`, `networkAlpha`, `networkFocus`, `pannable`, `orbits`, `burstKey`, `variant`；`hover`；四角 slots | 弹簧布局、脉冲、平移缩放、舞台/浅色变体 | `charts/relation-graph.tsx` |
| `Swimlane.vue` | `lanes`, `nowSec`, `compact`, `maxLanes`, `fixedLanes`, `showHiddenHint`, `slowMs`；`select` | 时间泳道、现在线、失败/慢操作 | `charts/swimlane.tsx` |
| `TickRing.vue` | `value`, `size`, `ticks`, `label`, `unit`, `decimals`, `display` | 刻度环、逐根点亮 | `charts/tick-ring.tsx` |
| `TrendChart.vue` | `values`, `secondary`, `labels`, `height`, `unit`, `primaryLabel`, `secondaryLabel`, `tone`, `variant`, `dots`, `band` | SVG 折线形变、稀疏茎叶、准星/提示 | `charts/trend-chart.tsx` |

## 7. Root and AI Components

| 组件 | Props / 事件 | 行为 | React 映射 |
| --- | --- | --- | --- |
| `Icon.vue` | `name`, `size`, `stroke` | 内联 SVG 图标字典 | `components/icon.tsx` |
| `NeuralNet.vue` | `burstKey?` | Canvas 节点、连线脉冲、成功收束 | `components/neural-net.tsx` |
| `AiAssistant.vue` | 无 | 可拖动悬浮球、聊天弹窗、AI 会话历史 | `features/assistant/ai-assistant.tsx` |

## 8. Migration Rules

1. 组件名称和视觉职责尽量保持不变，React 文件按 `ui/`、`business/`、`layout/`、`charts/` 分层。
2. Props 改为显式 TypeScript 接口；事件改为回调，不把 Vue 的 `v-model` 机械翻译为全局状态。
3. 复杂组件的 Canvas/GSAP 逻辑放在 Client Component，并在 effect 清理时 kill timeline、取消 RAF、移除监听。
4. 共享浮层层级和焦点管理必须先迁移 `focus.ts`，否则 Modal/Drawer/Select/HoverCard 会出现嵌套层级回归。
5. 表格、筛选、分页和导出必须保留真实 loading/error/empty 状态；不能用静态 mock 替换。
6. 图标使用现有资源或统一图标库；不得为迁移重画一套视觉不一致的图标。
