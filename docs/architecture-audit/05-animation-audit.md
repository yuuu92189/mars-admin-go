# 05 Animation Audit

## 1. Implementation Inventory

前端实际使用了三类动画：

1. GSAP 时间线、Tween、`quickTo`、Flip 和 `gsap.ticker`。
2. CSS `@keyframes`、`transition`、`transform`、`opacity`。
3. 原生 `requestAnimationFrame`/Canvas 帧循环。

没有发现：

- Web Animations API `element.animate()`。
- `new Animation()` 或 `KeyframeEffect`。
- GSAP ScrollTrigger。

## 2. GSAP Files

GSAP 入口是 `scaffold-vue/src/composables/motion.ts`，其中注册 `Flip` 插件。主要调用文件：

| 文件 | 调用 | 用途 |
| --- | --- | --- |
| `composables/motion.ts` | `gsap.to/fromTo/timeline/set/quickTo/killTweensOf/ticker` | 全局按压、悬停、页面、浮层、Toast、共享元素、帧循环 |
| `views/Login.vue` | `timeline`, `quickTo`, `set` | 登录开场、卡片倾斜、成功圆环 |
| `views/Dashboard.vue` | `gsap.from` | 首次进入工作台卡片弹簧飞入 |
| `components/business/BoardView.vue` | `fromTo/to/set/getProperty` | 看板拖拽、Ghost、落位 |
| `components/business/GeneratorPipeline.vue` | `fromTo` | 流水线步骤切换、代码逐行级联 |
| `components/business/OrgCanvas.vue` | `timeline/set/quickTo/killTweensOf` | 组织画布平移缩放和路径脉冲 |
| `components/charts/BarChart.vue` | `fromTo/killTweensOf` | 柱状入场 |
| `components/charts/HeatGrid.vue` | `fromTo` | 热力格对角级联 |
| `components/charts/Swimlane.vue` | `fromTo` | 时间块生长 |
| `components/charts/TickRing.vue` | `timeline/killTweensOf` | 刻度逐根点亮 |
| `components/charts/TrendChart.vue` | `to/quickTo/set` | 折线形变、提示跟随 |
| `components/ui/ToastHost.vue` | `fromTo/to` | Toast 进出 |

`NeuralNet.vue`、`RelationGraph.vue`、`OrgCanvas.vue` 还使用 `createFrameLoop`；它不是 GSAP Timeline，但挂在 `gsap.ticker` 上。

## 3. Motion Constants

`motion.ts` 中的统一缓动和时长：

```ts
EASE = {
  reveal: 'expo.out',
  glide: 'power4.out',
  exit: 'power2.in',
  swap: 'power2.inOut',
  settle: 'back.out(1.6)',
}

DUR = {
  press: 0.14,
  release: 0.46,
  hoverIn: 0.34,
  hoverOut: 0.42,
  slide: 0.52,
  pageOut: 0.1,
  pageIn: 0.3,
  overlayIn: 0.42,
  overlayOut: 0.2,
}
```

能量语态：

| 语态 | 典型效果 |
| --- | --- |
| 蓄 Charge | 输入、按压、登录按钮 |
| 放 Discharge | 主按钮涟漪、弹窗从触发点展开、抽屉生长、浮条升起 |
| 导 Conduct | 折线描绘、刻度环、泳道、现在线、图谱脉冲 |
| 结 Crystallize | 读数落定、柱子生长、看板落位、登录对勾 |
| 停摆 Halt | 抽屉收回、拖拽取消、浮条下沉 |

## 4. CSS Keyframes

| 文件/选择器 | Keyframe | 效果 | reduced motion |
| --- | --- | --- | --- |
| `components.css` | `spin` | 加载旋转 | 全局缩短 |
| `components.css` | `fade-in` | 导航/浮层淡入 | 全局缩短 |
| `components.css` | `toast-count` | Toast 倒计时环 | 全局缩短 |
| `components.css` | `pulse` | 骨架/状态呼吸 | 全局缩短 |
| `instruments.css` | `rg-breathe` | 图谱节点呼吸 | 显式禁用 |
| `instruments.css` | `insight-grow` | 分析柱生长 | 全局缩短 |
| `pages.css` | `share-grow` | 数据权限占比条生长 | 显式禁用 |
| `components/ai/AiAssistant.vue` | `ai-typing` | AI 输入点 | 显式禁用 |
| `components/business/LoginBrandPanel.vue` | `term-blink` | 终端光标闪烁 | 显式禁用 |

## 5. Transitions

CSS transition 集中在：

- 按钮、输入、下拉、表格行、分页、Tag、Toast、抽屉、Select、HoverCard。
- 侧栏导航颜色/透明度、顶栏凝结层、移动侧栏 `translate`。
- 图表 hover、组织画布、矩阵、图谱节点、样本册、阶梯。
- `--fill` 自定义属性过渡，用于 Switch 进度。

原则：

- 简单 hover/focus/颜色变化优先 CSS transition。
- 简单循环/加载优先 CSS animation。
- 多元素时间线、共享元素、拖拽、画布、页面编排使用 GSAP。
- 不把所有动画迁移成 GSAP；也不为了减少依赖删除原动效。

## 6. Page/Component Animation Matrix

| 页面/组件 | 效果 | 当前实现 | React 迁移 |
| --- | --- | --- | --- |
| 登录 | 品牌逐项落定、卡片入场、成功圆环 | GSAP timeline + quickTo | `useLayoutEffect` + `gsap.context`；成功动画完成后导航 |
| 登录品牌面板 | 终端打字、阶段自动轮播、光标闪烁 | `setInterval` + CSS keyframe | `useEffect` 清理 interval；阶段状态由 React 驱动 |
| 神经网络背景 | 节点呼吸、连线脉冲、成功收束 | Canvas + `createFrameLoop` | Canvas ref；ResizeObserver；hidden/offscreen 暂停 |
| 工作台 | 首次进入卡片从中心飞出 | `gsap.from` + `firstTimeInSession` | 首会话标记 + `useGSAP`/context；避免 StrictMode 双播 |
| 外壳/页面 | 页面进出、导航品牌/顶栏入场 | `pageEnter/pageLeave/playShellEntrance` | Next route transition/effect；组件卸载 kill timeline |
| 按钮/导航 | 按压、悬停抬起、主按钮涟漪 | 全局事件委托 GSAP | React 事件或 motion provider；清理全局监听 |
| Modal/Drawer/Toast | 从触发点展开、抽屉 morph、Toast 进出 | GSAP timeline + morphGhost | Portal + refs + `gsap.context`；卸载移除 ghost |
| 底部浮条 | 批量/未保存差异升起 | GSAP from/to | 统一 `FloatBar` 组件 |
| 看板 | 拖拽、取消飞回、落位 | GSAP + spring + frame loop | dnd 事件 + `requestAnimationFrame`；取消时回原列 |
| 组织画布 | 平移缩放、选中路径脉冲 | `quickTo`/`gsap.set` + frame loop | Pointer Events + transform；清理 capture |
| 关系图/星图/溯源 | 节点弹簧、脉冲、舞台进场 | Canvas/关系图 + frame loop | Canvas/WebGL 按需；不引入图表库替代视觉 |
| 柱状图 | 柱体生长 | GSAP stagger | CSS transform 或 GSAP，保留 stagger |
| 折线图 | 描线、数据形变、准星 | GSAP + SVG path | SVG + `useLayoutEffect`；避免 SSR 访问 DOM |
| 刻度环 | 逐根点亮 | GSAP timeline | SVG + GSAP；kill 清理 |
| 泳道 | 时间块生长、现在线 | GSAP + interval | Client Component；时间推进需清理 |
| 热力格 | 对角波纹点亮 | GSAP stagger | CSS animation 或 GSAP；键盘焦点保留 |
| 生成流水线 | 步骤切换、字段连线、代码逐行 | GSAP | `GeneratorPipeline` effect；取消旧 timeline |
| 字典样本册 | 卡片原地展开/收起 | GSAP Flip | `Flip`/layout animation；避免重排 |
| 通知/任务/日志 | 抽屉、级联、状态落定 | GSAP/`revealCascade` | 统一 drawer/cascade hooks |

## 7. Lifecycle and Risk

### Good

- `skipMotion()` 统一处理 reduced motion 和后台标签页。
- `createFrameLoop` 在 hidden/offscreen 暂停，单帧错误只报告一次。
- 多数图表在 `onBeforeUnmount` 调用 `gsap.killTweensOf`。
- 登录、组织画布、看板、图表有监听清理或终态处理。
- 没有 ScrollTrigger/WAAPI 的额外兼容负担。

### React Migration Risks

1. GSAP 不能在 Server Component 执行；必须隔离到 Client Component。
2. React StrictMode 会重复 effect；必须使用 `gsap.context` 或显式 kill，避免双播和幽灵元素。
3. 全局 `document`/`window` 监听要有安装次数和清理策略；当前 `motion.ts` 的全局监听是模块级单例。
4. `setInterval`、`requestAnimationFrame`、ResizeObserver、pointer capture 必须成对清理。
5. 页面切换时先完成旧 timeline 的 kill，再启动新页面，避免旧动画残留。
6. Next.js hydration 前不能读取 `window`、`document.hidden` 或 `matchMedia`。
7. 共享元素 morph 需要确保目标元素在 DOM 中存在，关闭后删除临时 ghost。

## 8. Acceptance Gates

- `prefers-reduced-motion: reduce` 下所有页面直接到达可见终态，不留下半透明遮罩。
- 页面隐藏再恢复不会停在动画中途。
- 快速切换表格/看板/图谱/抽屉不会产生重复监听、重复 Ghost 或不可点击遮罩。
- 登录成功、失败、取消导航都能恢复按钮状态。
- 键盘操作不依赖鼠标 hover；Esc 只关闭最上层浮层。
- Playwright 截图在动画终态后采集，并记录动画配置/浏览器版本。
