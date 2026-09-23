# cpa-secret-manager

CLIProxyAPI（CPA）的**代理 API Key 管理与用量归因插件**：为每把代理密钥维护一条备注，并展示该密钥的 Token 用量与模型分类。

CLIProxyAPI 本身可以增删改代理密钥，但无法为密钥附加任何人工标注，也不提供按密钥的 Token 用量视图。本插件补齐这两个缺口。

---

## 功能

- **密钥管理**：列出、新增、编辑、删除代理密钥，默认掩码显示，支持单条显示与复制，按密钥或备注筛选。
- **备注**：每把密钥一条备注（无名称字段），按密钥的 SHA-256 摘要索引，插件**不保存密钥明文**。
- **自动生成**：使用与官方管理端完全一致的算法生成密钥 —— `sk-` 前缀加 48 位大小写字母数字，并通过拒绝采样消除取模偏差。
- **Token 用量**：按密钥累计请求数、失败数、输入/输出/推理/缓存 Token 与合计，并按模型分类展示明细。
- **CPA 深度融合**：管理页纯原生自包含（零外部 CDN），同源打开时自动复用官方管理端已记住的管理密钥，实时跟随宿主主题（含自定义主题）与界面语言。

## 安装

把插件动态库放入 CLIProxyAPI 的插件目录，**文件名必须保持 `cpa-secret-manager`**（插件 ID 由文件名派生）：

| 平台 | 路径 |
| --- | --- |
| Linux | `plugins/linux/amd64/cpa-secret-manager.so` |
| macOS | `plugins/darwin/arm64/cpa-secret-manager.dylib` |
| Windows | `plugins/windows/amd64/cpa-secret-manager.dll` |

在 `config.yaml` 中启用：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-secret-manager:
      enabled: true
      priority: 1
      # 可选：自定义插件状态文件路径
      state_path: "data/cpa-secret-manager-cache.json"
```

启动后打开官方管理端，在插件菜单点击「API Key Manager」。

> 宿主需要支持插件能力（`usage_plugin` 与 `management_api`）。推荐使用最新版 CLIProxyAPI。

## 使用

- **管理密钥**：同源打开时自动读取官方管理端在 `cli-proxy-auth` 中保存的会话密钥；跨源部署或未勾选记住密码时，点击顶部「管理密钥」手动输入（仅保存在当前浏览器标签页）。
- **状态文件**：备注与用量保存在 `data/cpa-secret-manager-cache.json`（可在「设置」页查看实际路径）。删除该文件即清空全部备注与用量统计。

## 用量口径

- 数据来自宿主的用量记录（`usage_plugin`），按**调用所命中的代理密钥**归因，并按模型分组。
- 计数为**累计值**，从插件观察到第一条请求开始，不按时间窗口统计，也不估算费用。
- 上游未返回 Token 的请求仍计入请求数，Token 记 0。
- 未命中当前密钥列表的请求计入「未归因请求」；某把密钥长期显示 0 时，先看这个计数。
- 用量写入按 5 秒节流落盘，进程被强制终止最多丢失一个窗口的计数。

## 隐私

- 不保存 API Key 明文，只保存 `SHA-256(key)` 摘要。
- 不读取 prompt、请求体、响应体，不落盘客户端 IP 与上游响应头。

## 管理接口

插件页面由宿主挂载在 `/v0/resource/plugins/cpa-secret-manager/keys`；其余接口都需要管理密钥。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/v0/management/api-keys` | 读取代理密钥列表（宿主接口） |
| PUT | `/v0/management/api-keys` | 全量保存代理密钥列表（宿主接口） |
| GET / PUT | `/v0/management/plugins/cpa-secret-manager/settings` | 读取 / 更新插件设置 |
| POST | `/v0/management/plugins/cpa-secret-manager/resolve` | 按提交的密钥列表返回对齐的备注与用量 |
| PUT | `/v0/management/plugins/cpa-secret-manager/remarks` | 写入备注（空字符串清除） |
| POST | `/v0/management/plugins/cpa-secret-manager/keys/generate` | 生成一把官方格式的密钥 |

## 本地开发

无需真实 CPA 即可完整调试：

```bash
go run ./cmd/devserver
```

打开 <http://localhost:8080/control-panel>。仿真宿主提供管理接口、插件菜单、同源 iframe、主题与语言联动，以及演示数据与用量注入，详见 [`cmd/devserver/README.md`](cmd/devserver/README.md)。

## 构建

需要 Go 与 CGO（对应平台的 C 编译器）：

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o cpa-secret-manager.so .
```

## 质量门禁

```bash
golangci-lint run --timeout=5m
go test ./...
go test -race ./...
```

## 文档

- 领域词汇表：[`CONTEXT.md`](CONTEXT.md)
- 架构决策：[`docs/adr/`](docs/adr/)
- 需求与里程碑：[`docs/requirements/v1.0.0-roadmap.md`](docs/requirements/v1.0.0-roadmap.md)
- 发布审阅清单：[`docs/release-checklist.md`](docs/release-checklist.md)

## 许可证

MIT
