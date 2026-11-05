# AGENTS.md

> **说明**：本文档仅定义本项目的核心设计原则、架构分层与发布规范；通用工程规则与门禁遵循全局提示词。

---

`cpa-secret-manager` 是专为 CLIProxyAPI (CPA) 打造的 **代理 API Key 管理与用量归因插件**。

### 核心设计原则
1. **密钥归属宿主，元数据归属插件**：API Key 的增删改查全部复用宿主 `/v0/management/api-keys`，插件只持有以 `SHA-256(key)` 为索引的备注与用量，**永不落盘、永不回显 key 明文**；
2. **生成算法与官方 1:1 对齐**：`sk-` + 48 位 `[A-Za-z0-9]`，拒绝采样阈值 248（消除取模偏差），与官方管理端 `generateSecureApiKey()` 完全同构；
3. **用量归因取自客户端密钥**：`usage_plugin` 下发的 `UsageRecord.APIKey` 即客户端代理密钥，按摘要累计请求数、失败数与 token 明细，并按模型分组；用量按 **UTC 整点小时桶**保存（受数据保留天数裁剪），页面按所选时间范围查询；**不消费官方用量队列、不依赖任何宿主开关、不做费用/趋势**；
4. **宿主配置极简无扰**：宿主 `config.yaml` 只承载 `enabled` / `priority`（可选 `state_path`），业务状态落 `data/cpa-secret-manager-cache.json`（用量桶落 `-buckets.json`、失败记录落 `-errors.json`），`ConfigFields` 保持 `nil`，避免管理端抽屉写回触发宿主重载；
5. **CPA 深度融合**：嵌入式 Web 管理页纯原生自包含（零外部 CDN），通过同源 iframe 实时抽取宿主 CSS 变量跟随 CPA 主题（含自定义主题），语言跟随宿主 `cli-proxy-language`。

---

## 🚀 版本发布前审阅 (Release Audit)

创建 Tag 或发布 GitHub Release 必须经用户显式授权，且必须严格遵循 [`docs/release-checklist.md`](./docs/release-checklist.md) 完成 4 处版本号一致性、无内部标签及全量质量自检。

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

Single-context repository. See `CONTEXT.md` and `docs/adr/`.
XERJ prefix: "csm"


### Roadmap

本项目的需求与实施计划集中在 `docs/roadmap.md`；不维护 `.scratch` 工单目录。
