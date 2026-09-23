# 0001: 密钥归属宿主，备注与用量归属插件

## Context

CLIProxyAPI 已经通过 `/v0/management/api-keys` 提供代理服务认证密钥的完整读写能力（全量替换、按 old/new 或 index/value 修改、按值或下标删除），并把结果同步到配置文件中 `auth.providers` 的 `config-api-key` 提供方。官方管理端可以增删改这些密钥，但**没有任何字段用来承载"这把密钥是干什么的"这类人工标注**，这是本插件要补的缺口。

同时，插件页面运行在 CPA 管理端的同源 iframe 中，携带管理密钥即可直接调用宿主管理接口；插件自身无法获得管理密钥，也没有宿主回调可以读写宿主配置。

## Decision

1. **密钥本身永远由宿主拥有**。插件不复制、不缓存、不落盘任何 API Key 明文，全部增删改查直接复用宿主 `/v0/management/api-keys`。
2. **备注与用量归属插件自有状态文件** `data/cpa-secret-manager-cache.json`，以 API Key 的 SHA-256 十六进制摘要为索引键。
3. 插件配置保持极简：宿主 `config.yaml` 只承载 `enabled`、`priority`（以及可选的 `state_path`）；业务参数（如用量采集开关）保存在插件状态文件的 `app_config` 节点中，由插件页面自行管理。
4. 注册元数据中的 `ConfigFields` 保持为 `nil`，避免 CPA 管理端渲染插件配置抽屉、并在保存时改写宿主 `config.yaml` 触发全量重载。

## Consequences

- 备注与用量不会随宿主 `config.yaml` 迁移或备份；迁移时需要一并带上 `data/` 目录。
- 修改备注不会触发宿主配置热重载，也不会产生 `plugin.reconfigure` 风暴。
- 页面必须持有管理密钥才能工作；同源 iframe 下优先读取官方管理端已保存的会话密钥，跨源部署时需手动输入。
- 密钥改名（编辑密钥值）后备注不会自动跟随，由页面在保存时显式地把备注写到新密钥并删除旧索引，规则可预测、无启发式猜测。
