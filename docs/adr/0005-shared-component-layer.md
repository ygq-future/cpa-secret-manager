# 0005: 页面组件层与原生控件外观禁用

## Context

插件页是一个自包含的单文件页面（详见 ADR-0003）：一份 `<style>`、一份 `<script>`、零外部 CDN，源码以 Go 字符串形式分散在 `internal/management/*.go` 的 `pageFeature{markup, styles, scripts}` 与 shell 常量里。

页面长期积累出三类问题：

1. **同一组件多份实现**：开关在两处各写一遍标记（一处带 `role="switch"`、一处不带）、下拉在 failures 面板有一套 `createCustomDropdown`、配额窗口却直接用原生 `<select class="custom-select">`（该类甚至没有样式定义）、图标既有 JS 里的字形表也有散落的 `innerHTML` SVG 字面量、加载掩码与旋转器写在 keys feature 内、`.btn-preset` 与 `.preset-pill` 是同一视觉概念的两套类名。
2. **浏览器原生外观直接漏到页面上**：原生 `<select>` 的弹层、`input[type="search"]` 的清除叉、`input[type="number"]` 的上下微调箭头、Chrome autofill 的黄底、`textarea` 的滚动条按钮。
3. **样式不服从调色板**：规则里散落 `rgba(99,102,241,…)`（调色板外的 indigo）、`#fff`、`rgba(239,68,68,…)`，JS 内联样式里引用未声明的 `var(--accent-yellow, #f59e0b)`；另有内联 `style=` 与未定义类名（`flex-1`、`text-xs`、`date-dialog-heading`、`time-input-wrap`）。

AGENTS.md 的设计原则 6 明确要求：通用交互（开关、数值+单位输入、多选网格等）必须下沉为可复用组件，不得裸用未美化的原生控件。

## Decision

1. 新增组件层，按语言拆成三个文件：`components.go`（Go 侧）、`components_styles.go`（样式）、`components_script.go`（运行时）：
   - `pageIcons` 图标表 + `resolveMarkup` 占位符展开（`{{ICON:name}}`、`{{TOGGLE_*}}`）；未知图标名或未解析占位符在装配期 panic，标记写错不可能上线；
   - `templateStyleBase` 元素归一化（微调箭头、搜索装饰、autofill、滚动条、`:focus-visible` 基线）；
   - `templateStyleComponents` 组件样式（按钮 / 胶囊 / 徽标 / 卡片 / 开关 / listbox / chip / 数值+单位 / 表格 / 掩码 / 弹层 / 状态条）；
   - `templateScriptComponents` 组件运行时（`iconMarkup`、`buildIconButton`、`mountSelect`、`mountChipGrid`、`refreshComponentLabels`）。
   `page.go` 固定装配顺序：Token → base → components → layout → overlays → feature 样式 → responsive；脚本：I18N → components → shell core → feature。
2. 每种通用交互只有一条实现：下拉 = `mountSelect`（自绘 listbox：`role="combobox"`/`listbox`/`option`，Enter/Space/方向键/Home/End/Esc/Tab，视口底部自动 drop-up，标签随语言切换重渲染）；多选 = `mountChipGrid`（`aria-pressed` 切换胶囊）；开关 = `toggleSwitchMarkup`（真实 checkbox 承担可访问性，外观由 `.toggle-slider` 绘制，轨道使用页面自有的 `--switch-track`/`--switch-track-border`，因此放在 `--bg-subtle` 面板上仍然可见）；数值+单位 = `.input-unit-group` + `.unit-input`，单位既可以是静态后缀 `.unit-suffix`，也可以是 `mountSelect` 挂载点（Token 上限与窗口时长都走这一条）；快捷值 = `.pill` 组；图标 = `pageIcons`；表格加载 = `.table-loading` + `.spinner`；按钮内的请求中状态 = `.btn.is-loading` + `.spinner`。
3. **禁止原生控件外观**：页面不得出现 `<select>`/`<option>`、date/range/color/file/radio 等原生输入、内联 `style=`；checkbox 只允许作为开关内部的可访问性载体。规则内不得出现颜色字面量（颜色只允许出现在 Token 声明块）。
4. **事件归属**：shell 只绑定页面外壳（语言、管理密钥、刷新、标签页、确认框），`bindKeysEvents` / `bindSettingsEvents` 由 feature 自行实现并在启动时调用；shell core 不再包含任何 feature 业务逻辑。
5. 门禁锁定：`TestPage_UsesNoNativeFormControls`、`TestPage_HasNoInlineStyles`、`TestPage_RulesUseTokensOnly`、`TestPage_SharesOneGlyphTable`，与既有的 Token 完备性、`var(--*)` 无未定义引用、零外部 URL 检查共同构成样式契约。

## Consequences

- 新增 UI 前先查组件层；只有确定不复用的标记才允许写在 feature 内。
- 自绘 listbox 需要自己承担原生 `<select>` 的键盘与弹层行为（含移动端触屏），这是换取"外观完全可控 + 主题一致"的代价；行为集中在 `mountSelect` 一处，便于后续修正。
- 页面多了一次 JSON 图标表注入与一次占位符展开，体积与装配成本略增；换来标记与运行时图标不可能漂移、组件不会各自分叉。
- 与 ADR-0003 的关系：ADR-0003 规定"表面/文字/边框跟随宿主，强调色归插件"，本 ADR 规定"这些颜色如何被组件消费"。新增 Token 必须同时进入 `:root`、`[data-theme='white']`、`[data-theme='dark']` 与 `@media (prefers-color-scheme: dark)` 兜底块。
