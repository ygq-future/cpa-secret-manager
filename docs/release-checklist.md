# CPA Secret Manager 发布审阅与版本迭代检查规范 (Release & Audit Checklist)

本文档定义 `cpa-secret-manager` 在每次版本迭代、代码提交与正式发布前的**标准化审阅流程与变更检查矩阵**。每次完成功能开发或缺陷修复后，在提交代码与发布 Release 前，**必须严格依照本清单逐项自检**。

---

## 目录
1. [版本号同步一致性矩阵](#1-版本号同步一致性矩阵)
2. [文档与双语同步审阅清单](#2-文档与双语同步审阅清单)
3. [CPA 宿主与插件架构契约审阅](#3-cpa-宿主与插件架构契约审阅)
4. [代码质量与测试验证矩阵](#4-代码质量与测试验证矩阵)
5. [标准化发布工作流步骤](#5-标准化发布工作流步骤)

---

## 1. 版本号同步一致性矩阵

当进行版本升级（如 `v1.0.0` $\to$ `v1.1.0`）时，**必须确保以下 4 个位置的版本号完全一致**：

| 检查项 | 目标文件 | 关键位置 / 字段 | 审阅标准 |
| :--- | :--- | :--- | :--- |
| **① 插件注册表** | `registry.json` | `plugins[0].version` | 纯三段式版本号，无 `v` 前缀，如 `"1.0.0"` |
| **② 共享版本常量** | `internal/version/version.go` | `PluginVersion` | 与 `registry.json` 完全一致，被宿主注册元数据与页面徽章共同引用 |
| **③ 嵌入式 Web 页面** | `internal/management/shell_markup.go` | `<span class="version-badge">v{{VERSION}}</span>` | 由 `PluginVersion` 注入，渲染结果为 `v1.0.0` |
| **④ 专属 Release Note** | `.github/release-notes/vX.Y.Z.md` | 文件名与一级标题 `# cpa-secret-manager vX.Y.Z` | 文件名必须为 `vX.Y.Z.md`，中英双语章节齐备 |

上述 ①②③④ 已由 `registry_workflow_test.go` 中的 `TestVersionConsistency` 与 `TestReleaseNotes_MatchTheCurrentVersion` 自动校验。

---

## 2. 文档与双语同步审阅清单

1. **核心功能聚焦原则**：公开文档只描述用户可见的能力与边界，**严禁堆砌实现细节**（如状态文件字段、缓存目录、内部路由常量）。
2. **纯净化原则（无内部研发代号）**：**严禁在 `README.md`、`README.en.md` 与 Release Notes 中出现 `(REQ-xx)` 之类内部研发标签**。
3. **双语对齐**：`README.md` 与 `README.en.md` 的章节结构保持 1:1 对应。
4. **Git 提交历史溯源原则（全量回溯）**：起草 Release Notes 前必须执行：

   ```bash
   git log <last-release-tag>..HEAD --oneline
   ```

   完整回溯自上一次发布以来的**全部**提交，逐一核对功能升级与缺陷修复，杜绝遗漏。

| 文档 | 审阅重点 | 检查要点 |
| :--- | :--- | :--- |
| **`README.md`** (中文) | 功能概览、安装、用量口径、隐私 | 无内部标签；接口表与实际路由一致；用量口径与实现一致（累计值、无时间窗口） |
| **`README.en.md`** (英文) | 英文对照完整性 | 与中文段落结构 1:1 对齐 |
| **`.github/release-notes/vX.Y.Z.md`** | 双语 Release Note | 基于提交历史全量回溯；包含 `### 中文` 与 `### English` |
| **`docs/requirements/`** | 路线图状态 | 里程碑状态与已交付内容保持一致 |

---

## 3. CPA 宿主与插件架构契约审阅

| 审阅维度 | 架构原理与检查标准 | 达标标准 |
| :--- | :--- | :--- |
| **宿主配置极简** | 宿主 `config.yaml` 只承载 `enabled` / `priority`（以及可选 `state_path`） | 业务开关（如用量采集）保存在插件状态文件中，不写入宿主配置 |
| **元数据干净度** | `buildMetadata()` 的 `ConfigFields` 保持为空 | 管理端不会渲染插件配置抽屉，也就不会因保存抽屉而重写 `config.yaml` 并触发宿主重载 |
| **密钥归属边界** | 密钥增删改查完全复用宿主 `/v0/management/api-keys` | 插件不缓存、不落盘、不回显密钥明文；状态文件中只出现 SHA-256 摘要 |
| **能力声明最小化** | 只声明实际实现的能力 | 当前声明 `management_api` 与 `usage_plugin`；插件不使用任何 `host.*` 回调 |
| **状态文件原子性** | 唯一落盘入口 `internal/state` | 写入走 `*.tmp` + `os.Rename`；文件权限 `0600` |
| **页面自包含** | 零外部 CDN、零远程资源、无内联事件处理器 | 由 `internal/management` 的门禁测试保证 |

---

## 4. 代码质量与测试验证矩阵

每完成一项功能或缺陷修复，必须先在本地执行 lint，再执行单元测试；准备提交或发布前，必须再次执行同一套 lint，并完成完整质量矩阵。

### 4.1 工具链约定

- 本地 lint 必须与 CI 保持一致：`golangci-lint v2.12.2`。
- 本地 Go 工具链需与 CI 同版本（`go1.26.7`），否则 golangci-lint 会因语言版本不匹配而崩溃：

  ```bash
  GOTOOLCHAIN=go1.26.7 golangci-lint run --timeout=5m
  ```

### 4.2 功能完成后的本地验证顺序

```bash
# 1. Lint（必须使用与 release workflow 一致的 golangci-lint v2.12.2）
GOTOOLCHAIN=go1.26.7 golangci-lint run --timeout=5m

# 2. 单元测试
go test ./...
```

### 4.3 提交与发布前完整验证

```bash
GOTOOLCHAIN=go1.26.7 golangci-lint run --timeout=5m
go build ./...
go vet ./...
go test -v ./...
go test -race ./...
```

### 4.4 关键测试项说明

- **`TestMain_CGO_ABI_Integration`**：真实编译动态库并在独立进程中加载，校验 4 个导出符号、初始化握手、`plugin.register` 信封、`management.handle` 分发与停机刷盘。
- **`TestVersionConsistency` / `TestReleaseNotes_MatchTheCurrentVersion`**：四处版本号与 Release Notes 文件名/标题一致性。
- **`TestRegistry_ContractMatchesPluginStoreSchema`**：`registry.json` 符合 CPA 插件商店规范。
- **`TestReleaseWorkflow_BuildsTheRegisteredPluginID`**：发布工作流的 `PLUGIN_NAME` 与插件 ID 一致。
- **`TestPage_HasNoRemoteResources` / `TestPage_HasNoInlineHandlers`**：页面零外部资源、零内联事件处理器。
- **`TestPageThemeTokensAreComplete` / `TestPage_ReferencedVariablesAreDefined`**：三套主题 Token 完备、无未定义变量引用。
- **`TestPage_TranslationsAreComplete` / `TestPage_DomReferencesResolve`**：双语字典完备、DOM 引用全部可解析。
- **`TestPage_JavaScriptSyntax`**：抽取页面脚本并执行 `node --check`。
- **`TestDevHost_*`**：本地宿主仿真器的密钥生命周期、用量归因、菜单契约与存储回读。

---

## 5. 标准化发布工作流步骤

```text
[功能开发 / 需求迭代]
         │
         ▼
[1. 关联代码与文档更新]
   ├─ 更新业务代码与单元测试
   ├─ 查阅自上个版本以来的全部提交：git log <last-tag>..HEAD --oneline
   ├─ 检查版本号一致性（registry.json / version.go / 页面徽章 / Release Notes）
   ├─ 编写 .github/release-notes/vX.Y.Z.md（双语、无内部标签）
   └─ 同步更新 README.md 与 README.en.md
         │
         ▼
[2. 功能完成后的本地验证]
   ├─ GOTOOLCHAIN=go1.26.7 golangci-lint run --timeout=5m
   ├─ go test ./...
   └─ go run ./cmd/devserver  （按需人工验收页面与主题跟随）
         │
         ▼
[3. 提交与发布前完整质量自检]
   ├─ go build ./...
   ├─ go vet ./...
   ├─ go test -v ./...
   └─ go test -race ./...
         │
         ▼
[4. 用户确认 (安全底线)]
   └─ "Never tag or publish without explicit user confirmation during the conversation"
         │
         ▼
[5. Git Tag 与 GitHub Release]
   ├─ Commit 消息遵循 Conventional Commits 规范
   ├─ 创建 Git Tag：git tag vX.Y.Z && git push origin vX.Y.Z
   └─ CI 自动交叉编译各平台动态库并发布 Release
```
