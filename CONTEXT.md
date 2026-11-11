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
插件数据目录 `data/cpa-secret-manager/`，状态文档 `state.json` 包含 `remarks`、`quota_rules`、`quota_states`、`app_config`；用量小时桶、失败记录与参考价缓存分别位于同目录的 `buckets.json`、`errors.json`、`prices.json`。写入必须原子替换。
_Avoid_: cache file、database、store（不加限定时歧义）。

**App Config**:
插件状态文件内的 `app_config` 节点，承载 `usage_enabled` 等业务开关，由插件设置页管理，**不写入宿主 `config.yaml`**。

**State Projection**:
插件状态只投影宿主 `api-keys` 里**当前存在**的密钥。读接口（`resolve`）永不删除任何东西；删除只发生在显式的 `forget` 调用：页面在本页删除密钥时调用它，或在确认某把密钥连续两轮都不在宿主列表里之后调用它。因此不存在"已删除但仍留在插件里"的中间态——在官方管理端删除与在本插件页删除，最终状态完全一致。
_Avoid_: orphan、孤儿、无主、残留（都不应作为产品概念出现）。

### Usage Attribution

**Usage Record**:
宿主通过 `usage_plugin` 能力下发的单次请求用量记录（`UsageRecord`）。字段 `APIKey` 是本次请求命中的 Proxy API Key，`AuthID` 是本次请求所选上游凭证的宿主标识。

**Attribution**:
把一条 Usage Record 归集到某把 Proxy API Key 的过程，通过 Hash Index 完成。

**Attributed**:
`APIKey` 命中了当前 `api-keys` 列表中的某把密钥时，该 Usage Record 才被计入。`APIKey` 为空、或它的摘要不属于任何受管密钥的记录**一律丢弃**：不计数、不落盘、不出现在页面上。

**Credential Attribution**:
失败记录额外保存 `UsageRecord.AuthID`，用于定位具体上游凭证。页面只展示该标识的前四位，完整标识保留在失败记录文档中。
_Avoid_: 上游密钥、账号（`AuthID` 是宿主标识，不是凭证材料）。

**Token Counters**:
每把密钥、每个模型维度的计数集合：`requests`、`failed`、`input`、`output`、`reasoning`、`cached`、`cache_read`、`cache_creation`、`total`。全部为**累计值**，无时间窗口。
_Avoid_: 用量统计、报表（暗示时间维度）。

### Pricing

**Model Price**:
为某个模型配置的输入 / 输出 / 缓存单价（USD per 1M tokens），保存在 `app_config.model_prices`，用于估算请求费用与消费配额。人工确认后才会生效。
_Avoid_: 费率、账单（暗示结算）。

**Reference Price**:
从 models.dev 目录解析出的建议单价。模型名按 `-` 分段做前缀逐级匹配（`gemini` → `gemini-3.5` → `gemini-3.5-flash`），最深一级没有命中时回退到上一级，因此带 `-high` 等渠道后缀的名称也能落到基础型号；页面用下拉框展示该模型命中的全部报价，最合适的一条默认选中，已保存的价格若与某条报价一致则标记为"已采用"。点击"同步并保存参考价"时按固定规则把最合适的一条合并进 `app_config.model_prices`：新增缺失、更新变化、忽略一致、保留参考价中已缺失的已保存模型。同步是唯一写入路径，不会在后台自动改写。
_Avoid_: 官方价、真实成本（目录价不等于宿主上游渠道的结算价）。

**Price Suggestion**:
针对页面当前模型列表计算出的 `Reference Price` 集合。只有模型名命中目录时才产生；未命中会被记入 Known Model Set，但不出现在建议集合里。每个命中模型携带按优先级排序的 `Offer` 列表，第一条即默认选中的最合适报价。

**Known Model Set**:
上一次成功同步覆盖的模型名集合（包含未命中的 miss）。条件请求只有在本次模型集合是其子集时才发送 `If-None-Match`，否则重新下载，保证新模型也能拿到参考价。

**Price Catalog Cache**:
派生缓存 `data/cpa-secret-manager/prices.json`，只保存来源、同步时间、ETag、Known Model Set 与命中的 Reference Price，不保存完整目录。删除该文件只会清空建议，下一次显式同步即可重建。
_Avoid_: 价格库、价格数据库（它不是权威数据源）。

**Price Sync**:
用户点击"同步并保存参考价"或"仅同步"后触发的外部网络动作：流式读取 models.dev `api.json`，只保留当前模型列表的匹配项，原子替换 Price Catalog Cache。"同步并保存参考价"在刷新后按合并规则写入已保存单价；"仅同步"只更新缓存，不写单价。参考价行末尾的刷新按钮以 merge 模式只刷新当前模型，并保留其他模型的缓存项。打开设置页或读取建议都不会联网，也不会写价格。

### Theme

**Host Theme Variable**:
宿主管理端在 `:root` / `[data-theme='dark']` 上声明的 CSS 变量（`--bg-*`、`--text-*`、`--border-*`、`--primary-*` 等）。插件页在 iframe 中通过 `getComputedStyle(parent.documentElement)` 实时抽取，不复制宿主的配色实现。

**Theme Fallback Set**:
与官方同名的三套静态 Token（浅色 / 纯白 / 深色），仅在无法读取宿主变量时生效（本地 devserver、直接打开资源 URL、跨源部署）。
