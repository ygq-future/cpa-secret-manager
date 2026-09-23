# 0002: 用量归因取自 usage_plugin 的客户端密钥

## Context

需要为每把代理密钥显示 token 用量与模型分类。可选的宿主数据源有三个：

1. `GET /v0/management/api-key-usage`：按**上游凭证**聚合（键为 `base_url|api_key`），只有 `success` / `failed` 计数，既没有 token 也没有模型，且无法对应到代理密钥。
2. `GET /v0/management/usage-queue` + Redis 队列：逐请求用量记录，但读取即出队（`PopOldest`），会与其他消费者争抢记录，且受 `usage-statistics-enabled` 开关控制。
3. 插件的 `usage_plugin` 能力：宿主在请求完成后把 `UsageRecord` 推给插件，非破坏性、不与其他工具争夺数据。

`UsageRecord` 的字段语义经官方源码逐层核对（v7.2.0 / v7.2.156 / v7.3.15 一致）：

```
config.yaml api-keys
  └─ internal/access/config_access/provider.go  Authenticate() → Principal = 命中的密钥原文
      └─ internal/api/server_middleware.go       c.Set("userApiKey", result.Principal)
          └─ internal/runtime/executor/helps/usage_helpers.go  APIKeyFromContext(ctx) → UsageReporter.apiKey
              └─ internal/pluginhost/adapters_usage_translation.go  UsageRecord.APIKey
```

即 `UsageRecord.APIKey` 是**客户端代理密钥**（上游凭证的信息出现在 `Source` / `BaseURL` / `AuthType` 中）。该字段的存在与语义在 v7.2.0 起即可用；`usage-statistics-enabled`（默认 `false`）只控制 Redis 队列插件，不影响插件的 `usage.handle`。

## Decision

1. 用量归因只使用 `usage_plugin` 能力，不读取、不消费官方用量队列，也不需要开启任何宿主开关。
2. 归因键为 `SHA-256(UsageRecord.APIKey)`；`APIKey` 为空或不属于 `api-keys` 的记录计入「未归因请求」计数，不落任何标识。
3. 只保存累计计数：按密钥与按模型的请求数、失败数、输入/输出/推理/缓存读取/缓存写入/合计 token。不做时间窗口、不做趋势、不做费用。
4. 内存聚合 + 节流原子落盘（默认 5 秒或 200 次变更），`plugin.shutdown` 强制刷盘；崩溃最多丢失一个刷盘窗口。

## Consequences

- 统计从插件启用后开始累计，没有历史回填。
- 上游不返回 usage 的请求计 0 token，但仍计入请求数（宿主 `EnsurePublished` 保证每请求恰好一条记录）。
- 计数是累计值，删除状态文件即清零；不提供按时间窗口的重置。
- 插件必须保持零依赖：不使用 bbolt 等嵌入式数据库，避免为单一用途引入新的存储引擎。
