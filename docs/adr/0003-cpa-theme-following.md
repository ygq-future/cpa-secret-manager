# 0003: 主题跟随宿主 CPA 管理端

## Context

官方管理端（Cli-Proxy-API-Management-Center）用**同源 `<iframe>`** 承载插件的浏览器资源页（`src/features/plugins/PluginResourcePage.tsx`）。宿主的主题状态是：

- `document.documentElement` 上的 `data-theme` 属性，取值 `dark`、`white`，或缺省（等价于浅色）；
- 主题偏好由 `useThemeStore` 持久化在 `localStorage` 的 `cli-proxy-theme`（取值 `auto` / `light` / `dark` / `white`，`auto` 时跟随 `prefers-color-scheme`）；
- 全部配色以 CSS 变量（`--bg-*`、`--text-*`、`--border-*`、`--primary-*` 等）声明在宿主页面的 `:root` 与 `[data-theme='dark']` 上。

插件页无法把自己的 DOM 挂进宿主文档，也不应复制宿主的主题实现（宿主主题会演进，复制必然过期）。

同族的已验证插件 `antigravity-priority` 给出了一条边界：**它只跟随宿主的表面、文字与边框，强调色始终是插件自己的**（其样式中 `var(--primary-*)` 出现 0 次）。这条边界经实践检验——按钮、开关、徽标在任何宿主主题下都保持一致的蓝色，只有底色与文字随宿主明暗变化。

## Decision

1. 插件页自带一套设计 Token 静态兜底值（浅色 / 纯白 / 深色三套），保证脱离宿主（本地 devserver、直接打开资源 URL）也正确显示。
2. 处于 iframe 时，通过 `getComputedStyle(window.parent.document.documentElement)` 抽取宿主**表面 / 文字 / 边框**变量（`--bg-primary` 画布、`--bg-secondary` 卡面、`--bg-tertiary` 内嵌、`--bg-hover`、`--text-primary/secondary/tertiary`、`--border-color`），以内联样式覆盖到插件页根元素，并把 `--bg-card` / `--bg-subtle` 重新指向宿主的二级 / 三级表面。宿主的任意自定义主题都会在此基础上生效。
3. **强调色不跟随宿主**：按钮、开关、徽标、focus ring 使用页面自有的 `--accent-blue/green/red` 三元组，与 `antigravity-priority` 行为一致——避免宿主主色（例如 CPA 默认的暖灰）把页面 CTA 冲淡成灰色。
4. 用 `MutationObserver` 监听父文档 `documentElement` 与 `body` 的 `data-theme` / `class` / `style` 变化，宿主切换主题时插件页实时切换，无需刷新。
5. 独立打开时降级为 `prefers-color-scheme`，并暴露 `window.setTheme(...)` 便于人工与 devserver 验收。
6. 语言同样跟随宿主：读取 `cli-proxy-language`，`zh-CN` / `zh-TW` 归一到简体，其余归一到英文；页面内仍保留手动切换。

## Consequences

- 插件页不提供独立主题开关，避免与宿主主题"打架"；明暗、底色、文字、边框仍完全由宿主决定。
- 宿主换一套非蓝主题时，页面强调色仍是蓝的：这是刻意的取舍（与基座一致），如果将来要跟随宿主主色，需要重新评估对比度并放开 `HOST_THEME_VARIABLES`。
- 跨源部署（插件页无法读取父文档）时退化为系统偏好配色，功能不受影响。
- 页面样式必须全部走 CSS 变量，禁止在规则里硬编码颜色（颜色字面量只允许出现在 Token 声明块中）；由测试门禁保证（Token 三主题完备性、`var(--*)` 无未定义引用、零外部 URL）。
