# CLAUDE.md

> **说明**：本文档与 [AGENTS.md](./AGENTS.md) 保持内容完全一致，确保 Claude Code 与各类智能体遵循统一的工程准则、交互协议与发布审阅门禁。

---

`cpa-secret-manager` 是专为 CLIProxyAPI (CPA) 打造的 **代理 API Key 管理与用量归因插件**。

### 核心设计原则
1. **密钥归属宿主，元数据归属插件**：API Key 的增删改查全部复用宿主 `/v0/management/api-keys`，插件只持有以 `SHA-256(key)` 为索引的备注与用量，**永不落盘、永不回显 key 明文**；
2. **生成算法与官方 1:1 对齐**：`sk-` + 48 位 `[A-Za-z0-9]`，拒绝采样阈值 248（消除取模偏差），与官方管理端 `generateSecureApiKey()` 完全同构；
3. **用量归因取自客户端密钥**：`usage_plugin` 下发的 `UsageRecord.APIKey` 即客户端代理密钥，按摘要累计请求数、失败数与 token 明细，并按模型分组；**不消费官方用量队列、不依赖任何宿主开关、不做时间窗口/费用/趋势**；
4. **宿主配置极简无扰**：宿主 `config.yaml` 只承载 `enabled` / `priority`（可选 `state_path`），业务状态落 `data/cpa-secret-manager-cache.json`，`ConfigFields` 保持 `nil`，避免管理端抽屉写回触发宿主重载；
5. **CPA 深度融合**：嵌入式 Web 管理页纯原生自包含（零外部 CDN），通过同源 iframe 实时抽取宿主 CSS 变量跟随 CPA 主题（含自定义主题），语言跟随宿主 `cli-proxy-language`。

---

## 🛑 严格交互与确认约束 (Strict Interaction Protocol)

除了用户明确输入了 `/<command>` 之类的 skill 或指令（如 `/impl` 等显式自动化指令）以外，当用户对某些功能抱有疑问、要求排查问题、分析缺陷、或要求做某些具体事情时，**严禁未经确认直接修改代码或盲目执行操作**。代理必须严格遵守以下确认流程：

1. **先输出思考与理解**：清晰告知用户你是如何理解用户所说的话和其核心诉求的；
2. **说明思路与解决路径**：说明你分析问题的思路、可能的根因排查方向或功能实现策略；
3. **列出具体行动计划**：明确说明你可能会怎么做（将要修改哪些文件、执行哪些命令、调整哪些逻辑）；
4. **说明预期结果与影响**：清晰阐述完成操作之后的预期结果、系统行为变化及是否有潜在副作用；
5. **等待用户明确确认**：将上述信息完整反馈给用户，**只有在得到用户的明确回复与确认后，方可开始具体修改或执行操作**。

---

## 🔒 Git 提交汇报与验收门禁 (Strict Git Commit Protocol)

**每次写完功能或修复后，严禁代理自主执行发布动作**。即使全量质量门禁完全通过，也必须严格遵守以下汇报与确认流程：

1. **汇报工作情况**：向用户详细汇报本次完成了哪些具体工作、修改了哪些文件、质量门禁的执行结果；
2. **说明验收需求**：明确说明本次修改是否有需要用户在真机/目标环境下验收的部分（如 Web UI 暗色主题表现、iframe 内主题跟随、CPA 宿主加载、接口交互等）；
3. **提供清晰验收步骤**：给出具体、可执行的验收操作步骤及对应的预期表现；
4. **等待用户确认**：只有在用户亲自确认验收结果，并给出明确指令（如"可以提交"、"可以发布"等）后，方可创建版本标签（Tag）与 GitHub Release。**严禁在未经用户明确授权的情况下自行创建 Tag 或发布 Release**。

> 例外：用户已在当前会话中明确授权"按节点自行提交"时，允许按里程碑创建普通 commit；**Tag 与 Release 始终需要用户逐次确认**。

---

## 🚀 版本发布与提交前审阅规范 (Release & Pre-Commit Audit)

在每次进行版本迭代、功能改动完成、准备向用户提请提交或发布 Release 之前，**AI 智能体必须主动查阅并严格遵循 [`docs/release-checklist.md`](./docs/release-checklist.md)**。

必须严格对照该文档完成以下关联项自检：
1. **版本号 4 处一致性核对**：`registry.json`、`internal/runtime/metadata.go`、`internal/management/shell_assets.go` 以及 `.github/release-notes/vX.Y.Z.md` 中的版本号必须 100% 同步一致；
2. **公开文档与双语规范**：`README.md` 与 `README.en.md` 保持中英双语对齐，聚焦核心价值，**严禁在面向用户的文档和 Release Notes 中出现 `(REQ-xx)` 内部研发标签**；每次起草 Release Notes 必须执行 `git log <last-tag>..HEAD --oneline` 完整回溯自上一次发布以来的全部提交，严禁遗漏任何中间 commit；
3. **架构与元数据契约**：确保 CPA 宿主 `config.yaml` 极简无扰、`buildMetadata()` 不暴露多余抽屉字段（`ConfigFields` 为 `nil`）、`config.Default()` 作为基石兜底与 UI 设置页保持一致；
4. **全套质量验证**：每完成一项功能或缺陷修复，必须先执行与 release workflow 一致的 `golangci-lint v2.12.2`，通过后再执行 `go test ./...`；提交或发布前必须再次执行 `golangci-lint run --timeout=5m`，并执行 `go build ./...`、`go vet ./...`、`go test -v ./...`、`go test -race ./...`，确保零告警与零数据竞争。

---

## 🏗️ 架构分层与所有权规范 (Architecture & Ownership)

- **Domain Core (`internal/keys`, `internal/remarks`, `internal/usage`)**：
  - 密钥生成、备注索引、用量聚合必须为**无副作用或仅内存副作用**的逻辑单元，禁止网络请求与宿主 IO；
  - 用量聚合的唯一入口是 `usage.Aggregator.Observe(record)`，并发安全，禁止在聚合器外直接修改计数。
- **Persistence (`internal/state`)**：
  - 插件业务状态的**唯一落盘入口**；写入必须走 `*.tmp` + `os.Rename` 原子替换；
  - 状态文档中**禁止出现 API Key 明文**，只允许 `SHA-256` 摘要作为索引。
- **Application Layer (`internal/runtime`)**：
  - 唯一拥有插件生命周期用例所有权（`Register` / `Reconfigure` / `Shutdown` / 管理路由分发 / 用量记录接收）；
  - 统一协调单飞互斥锁（`runMu`）与用量刷盘节流定时器。
- **Adapters (`main.go`, `internal/management`)**：
  - CGO ABI 导出与 HTTP Management API 仅作为薄适配器层透传调用 Runtime 用例，**严禁自行计算或改写业务逻辑**；
  - Web UI（`internal/management/*`）纯前端原生实现，无外部 CDN，样式完全遵从 CPA 主题（全部走 CSS 变量）。
- **Local Harness (`cmd/devserver`)**：
  - 仅用于本地仿真 CPA 宿主（管理接口、插件列表、iframe 外壳、用量注入），**不得进入发布产物路径**。

---

## 📚 领域文档与配置索引 (Domain Docs & Configuration)

## Agent skills

### Domain docs

Single-context repository. See `docs/agents/domain.md`, `CONTEXT.md`, and `docs/adr/`.

### Roadmap

本项目的需求与实施计划集中在 `docs/requirements/v1.0.0-roadmap.md`；不维护 `.scratch` 工单目录。
