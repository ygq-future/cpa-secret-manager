# CPA Secret Manager

CLIProxyAPI 的代理 API Key 管理与用量归因插件。

## Language

### Managed Keys 与 Host API

**Proxy API Key**:
CLIProxyAPI 用于认证**下游客户端**的密钥，存放于宿主配置 `api-keys`（运行时映射为 `auth.providers` 的 `config-api-key` 提供方）。它是本插件管理的对象，也是用量归因的依据。
_Avoid_: client key、接入密钥、token（易与上游凭证混淆）。

**Upstream Credential**:
用于访问上游模型服务的凭证（`gemini-api-key` / `claude-api-key` / `codex-api-key` / OAuth auth 文件等）。**不属于**本插件管理范围。
_Avoid_: 账号、api key（不加限定时歧义）。

**Host Management API**:
宿主在 `/v0/management/*` 下暴露的管理接口，需要管理密钥鉴权。本插件的密钥 CRUD 直接复用它，插件自有接口挂在同一前缀下由宿主统一鉴权。

**Menu Resource**:
通过 `management.register` 声明的浏览器资源页，最终暴露在 `/v0/resource/plugins/<pluginID>/<path>`，由官方管理端以同源 iframe 加载。

### Metadata 与持久化

**Remark**:
附加在一把 Proxy API Key 上的人工标注文本，用于说明用途。**没有**名称字段，无需唯一，可为空（空即删除）。
_Avoid_: name、label、alias、note、description。

**Hash Index**:
以 `SHA-256(key)` 的十六进制小写摘要作为备注与用量的索引键。明文 key 永不落盘。
_Avoid_: fingerprint、id、uuid。

**Plugin State File**:
插件自有状态文档 `data/cpa-secret-manager-cache.json`，包含 `remarks`、`usage`、`app_config`、`metrics` 四个节点；写入必须原子替换。
_Avoid_: cache file、database、store（不加限定时歧义）。

**App Config**:
插件状态文件内的 `app_config` 节点，承载 `usage_enabled` 等业务开关，由插件设置页管理，**不写入宿主 `config.yaml`**。

### Usage Attribution

**Usage Record**:
宿主通过 `usage_plugin` 能力下发的单次请求用量记录（`UsageRecord`）。字段 `APIKey` 是本次请求命中的 Proxy API Key。

**Attribution**:
把一条 Usage Record 归集到某把 Proxy API Key 的过程，通过 Hash Index 完成。

**Attributed / Unattributed**:
`Attributed` = `APIKey` 命中了当前 `api-keys` 列表中的某把密钥；`Unattributed` = `APIKey` 为空，或它的摘要不在当前列表中。未归因记录只计入总数，不出现在任何密钥行上。

**Token Counters**:
每把密钥、每个模型维度的计数集合：`requests`、`failed`、`input`、`output`、`reasoning`、`cached`、`cache_read`、`cache_creation`、`total`。全部为**累计值**，无时间窗口。
_Avoid_: 用量统计、报表（暗示时间维度）。

### Theme

**Host Theme Variable**:
宿主管理端在 `:root` / `[data-theme='dark']` 上声明的 CSS 变量（`--bg-*`、`--text-*`、`--border-*`、`--primary-*` 等）。插件页在 iframe 中通过 `getComputedStyle(parent.documentElement)` 实时抽取，不复制宿主的配色实现。

**Theme Fallback Set**:
与官方同名的三套静态 Token（浅色 / 纯白 / 深色），仅在无法读取宿主变量时生效（本地 devserver、直接打开资源 URL、跨源部署）。
