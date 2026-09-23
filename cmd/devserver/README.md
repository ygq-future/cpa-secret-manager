# 本地 CPA 宿主仿真器

`cmd/devserver` 启动一个不访问真实 CLIProxyAPI 的本地宿主环境：它使用**生产 runtime 与生产管理页面**，只把宿主那一侧替换为本地实现，用于人工验收主题跟随、密钥管理与用量统计。

## 启动

```bash
go run ./cmd/devserver
```

打开 <http://localhost:8080/control-panel>。

默认参数：

| 参数 | 环境变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `-addr` | `CPA_SECRET_MANAGER_DEVSERVER_ADDR` | `127.0.0.1:8080` | HTTP 监听地址；默认只绑定回环地址，不会触发 Windows 防火墙授权提示 |
| `-management-key` | `CPA_SECRET_MANAGER_DEVSERVER_KEY` | `devkey` | 模拟的管理密钥 |
| `-keys` | `CPA_SECRET_MANAGER_DEVSERVER_KEYS` | `data/devserver/api-keys.json` | 模拟的代理密钥存储 |
| `-state` | `CPA_SECRET_MANAGER_DEVSERVER_STATE` | `data/devserver/cache.json` | 插件状态文件 |

需要从其它设备访问仿真环境时，显式指定全部网卡地址（会触发防火墙提示）：

```bash
go run ./cmd/devserver -addr 0.0.0.0:8080
```

## 控制面板做了什么

控制面板模拟官方管理端在插件页面周围提供的一切：

- 在 `documentElement` 上应用 CPA 主题（`light` / `white` / `dark`），并按官方语义写入 `cli-proxy-theme`（zustand 结构）；
- 写入 `cli-proxy-language`，用于验证语言跟随；
- 以官方 `enc::v1::` 混淆格式写入 `cli-proxy-auth`，因此插件页的自动获取管理密钥路径会被真实覆盖；
- 通过 `GET /v0/management/plugins` 读取插件菜单，并用**同源 iframe** 加载资源页（与官方 `PluginResourcePage` 一致）。

顶部「Seed demo data」会生成 3 把官方格式密钥、写入备注，并为每把密钥注入 3 个模型的用量记录，另加 1 条未归因记录。

## 端点

宿主侧（需要管理密钥）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET / PUT / PATCH / DELETE | `/v0/management/api-keys` | 代理密钥存储，语义与官方管理 API 一致 |
| GET | `/v0/management/plugins` | 插件列表与菜单（字段与官方前端一致） |
| POST / PUT | `/v0/management/plugins/cpa-secret-manager/*` | 转发到生产 runtime 的插件路由 |

资源与调试：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/v0/resource/plugins/cpa-secret-manager/keys` | 生产插件页面（无管理鉴权，与宿主一致） |
| GET | `/control-panel` | 仿真宿主外壳 |
| POST | `/dev/seed?keys=3` | 生成演示密钥、备注与用量 |
| POST | `/dev/simulate-usage` | 注入一条或多条用量记录 |

`/dev/simulate-usage` 支持的查询参数：

```bash
# 命中某把密钥的 3 条 gpt-5.6 用量
curl -X POST "http://localhost:8080/dev/simulate-usage?key=sk-xxx&model=gpt-5.6&count=3&input=100&output=200&total=300"

# 不传 key 即产生一条未归因记录
curl -X POST "http://localhost:8080/dev/simulate-usage?model=gpt-5.6"
```

其余可选参数：`reasoning`、`cached`、`cache_read`、`failed=true`。

## 与真实宿主的差异

- 用量记录由 `/dev/simulate-usage` 主动注入，而不是由代理请求产生。
- 上游请求、凭证文件与模型执行不参与仿真：本插件不使用任何 `host.*` 回调。
- 控制面板只实现官方管理端与插件相关的最小外壳，不包含其它页面。
