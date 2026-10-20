<div align="center">

# CPA Secret Manager (`cpa-secret-manager`)

[中文](./README.md) | [English](./README.en.md)

</div>

CLIProxyAPI (CPA) 代理 API Key 管理与用量归因插件。插件 ID、动态库基础名与 CPA 配置键均为 `cpa-secret-manager`。

为每把代理密钥提供**备注管理**、**逐密钥 Token 用量归因**与**配额/模型白名单控制**，补齐官方管理端缺失的人工标注、用量细分与访问边界。

---

## 导航

- [功能概览](#功能概览)
- [工作流程](#工作流程)
- [构建与安装](#构建与安装)
- [插件商店来源](#插件商店来源)
- [配置说明](#配置说明)
- [用量归因口径](#用量归因口径)
- [管理页面与接口](#管理页面与接口)
- [本地开发](#本地开发)
- [许可证](#许可证)

---

## 功能概览

- **密钥归属宿主，元数据归属插件**：API Key 的增删改查完全复用宿主 `/v0/management/api-keys`，不产生第二份密钥来源。插件仅维护以 `SHA-256(key)` 摘要为索引的备注与用量计数，**永不落盘、永不回显 key 明文**。
- **与官方管理端 1:1 同构密钥生成**：自动生成格式为 `sk-` 前缀 + 48 位 `[A-Za-z0-9]`，拒绝采样阈值 248（消除取模偏差），生成的密钥与官方管理中心完全同构。
- **全维度 Token 用量归因**：通过宿主 `usage_plugin` 拦截客户端调用凭据，按密钥和模型聚合请求数、失败数、Input/Output Token，并支持 Reasoning（推理）、Cached（缓存创建）、CacheRead（缓存读取）等多维明细指标。
- **逐密钥配额与模型白名单**：为每把密钥设置自然日、自然周、自然月或固定时长窗口的 Token 上限，并可限定允许调用的模型；超限或越权请求在进入上游前直接以 `403` 终止，未配置规则的密钥不受影响。
- **时间范围看板与失败排查**：按日期范围聚合统计卡片、密钥列与模型明细，展示每把密钥的「上次调用」时间；失败记录按时间线归集，便于定位异常密钥与上游错误。
- **宿主孤儿数据自动清理**：提供 `/forget` 协议，当宿主中删除密钥后，插件侧元数据与用量自动检测并完成安全级联清除，拒绝孤儿垃圾残留。
- **CPA 深度融合与原生质感**：嵌入式 Web 管理页纯原生自包含（零外部 CDN，零内联事件处理器），通过同源 iframe 实时抽取宿主 CSS 变量跟随 CPA 主题（浅色、纯白、深色及自定义主题），界面语言跟随宿主 `cli-proxy-language`。
- **宿主配置极简无扰**：宿主 `config.yaml` 仅需承载 `enabled: true`，业务状态持久化至独立缓存文档，元数据中 `ConfigFields` 保持为空，避免管理端抽屉写回触发宿主不必要的热重载。

---

## 工作流程

```text
客户端调用 CPA
  │
  ▼
CLIProxyAPI 宿主转发
  │
  ├─► [request.intercept_before 配额拦截]
  │     └─ 读取该客户端密钥的 Token 配额与模型白名单
  │     └─ 超限或模型不允许 ──► 直接返回 403，不进入上游
  │
  ├─► [usage_plugin 记录拦截]
  │     └─ 提取请求所命中的客户端代理密钥
  │     └─ usage.Aggregator 按 UTC 小时桶并发累加（请求数/失败数/Token明细/模型分类）
  │     └─ 5 秒定时器节流批量落盘（停机时原子强制刷盘）
  │
  └─► [management_api 管理协同]
        ├─ 读写代理密钥列表 ──► 复用宿主 /v0/management/api-keys（权威源）
        ├─ 读写备注与用量   ──► 插件 /resolve 接口返回 SHA-256 关联的备注与计数
        ├─ 删除密钥清理    ──► 插件 /forget 接口安全销毁失效哈希的残留元数据
        └─ 嵌入式 Web UI   ──► 同源 iframe 自动复用 cli-proxy-auth 密钥，零感知跟随主题与语言
```

---

## 构建与安装

插件以 CGO 动态库形式运行，宿主会从动态库文件名去掉扩展名得到插件 ID，因此动态库文件名必须保持为 `cpa-secret-manager.<ext>`。

### 本地编译

需要本地安装 Go（推荐与 CI 编译工具链一致：`go1.26.7`）及对应平台的 C 编译器（GCC / Clang / MinGW）：

```bash
# Linux / macOS
CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o cpa-secret-manager.so .

# Windows (MSYS2 / MinGW-w64)
CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o cpa-secret-manager.dll .
```

### 部署到 CPA

编译完成后，将生成的动态库放入 CLIProxyAPI 插件发现目录之一：

- `plugins/<GOOS>/<GOARCH>/cpa-secret-manager.<ext>`
- `plugins/<GOOS>/<GOARCH>-<variant>/cpa-secret-manager.<ext>`
- `plugins/cpa-secret-manager.<ext>`

各操作系统对应动态库扩展名：
- **Linux / FreeBSD**：`.so`
- **macOS**：`.dylib`
- **Windows**：`.dll`

---

## 插件商店来源

如需通过 CPA 插件商店（Plugin Store）一键安装本插件，可在 CPA 的 `config.yaml` 中添加第三方来源：

```yaml
plugins:
  enabled: true
  store-sources:
    - "https://raw.githubusercontent.com/ygq-future/cpa-secret-manager/main/registry.json"
```

> **注意**：
> - 必须使用 raw 原始文件地址，不要使用包含 `/blob/` 的 GitHub 网页浏览地址。
> - 保存 `config.yaml` 后，重启 CPA 或在管理端重新加载配置，即可在插件商店列表中一键安装并自动识别平台架构产物。

---

## 配置说明

### CPA 宿主极简配置 (`config.yaml`)

推荐在 CPA 的 `config.yaml` 中仅保留极简的插件启用配置，所有状态与业务开关均交由插件内部管理：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-secret-manager:
      enabled: true
      priority: 1
      # state_path: "data/cpa-secret-manager-cache.json" # 可选，自定义持久化缓存路径
```

| 宿主字段 | 默认值 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| **`enabled`** | `true` | 是 | 插件全局启用开关。设为 `false` 时停用插件所有接口与用量监听。 |
| **`priority`** | `1` | 否 | 插件在 CPA 插件链中的执行优先级。 |
| **`state_path`** | `data/cpa-secret-manager-cache.json` | 否 | 状态文档路径；同目录下另有 `-buckets.json`（小时用量桶）与 `-errors.json`（失败记录）。 |

> **持久化机制**：三份文档（状态 / 用量桶 / 失败记录）全部走 `*.tmp` 临时文件 + `os.Rename` 原子替换，文件权限锁定为 `0600`。删除这三个文件即可完整重置备注、用量与失败记录。

---

## 用量归因口径

- **统计来源**：直接接入宿主 `usage_plugin` 分发的 `UsageRecord`，按命中的代理 API Key 摘要进行分组归因。
- **时间桶模型**：用量按 **UTC 整点小时桶**保存（密钥 × 模型 × 小时），页面按所选时间范围聚合；范围最多回溯到设置中的**数据保留天数**（1–365，默认 30），超出的桶在每次刷盘时被裁剪。不做费用折算与趋势图。
- **配额窗口**：自然日、自然周（周一 00:00 UTC）与自然月（1 号 00:00 UTC）按 UTC 对齐，固定时长窗口按配置秒数滚动；达到 Token 上限或模型不在白名单时，请求在进入上游前被 `403` 终止。
- **上次调用**：每把密钥记录保留期内最后一次请求的精确时刻，独立于页面所选日期范围；历史数据缺少精确时间时回退到小时桶起点。
- **异常计费容错**：上游返回错误或无 Token 响应的请求，仍会如实计入对应密钥的请求数或失败数（Token 计为 0）。
- **未受管密钥丢弃**：仅归因归属于 CPA 宿主当前受管列表内的密钥；空密钥或未受管的请求将被丢弃，不向界面抛出中间杂音。
- **节流落盘**：内存聚合通过并发安全互斥锁保护，按 5 秒周期节流写入磁盘，并在进程正常关闭（Shutdown）时强制同步刷盘。升级后时间桶从升级时刻开始累积，旧版的累计计数不带时间归属，不再展示。

---

## 管理页面与接口

插件通过 `management.register` 向宿主分别注册**静态 Web UI 资源**与**动态管理路由**。

### 静态 Web 资源（无需独立鉴权）

- `GET /v0/resource/plugins/cpa-secret-manager/keys`
  - **访问入口**：在 CPA 官方管理端侧边栏点击 **"API Key Manager"** 插件菜单打开，或在浏览器中同源访问。
  - **交互特色**：
    - **密钥与备注看板**：默认掩码保护，支持单条点击复制、按密钥/备注实时筛选与多单位（K / M / B）Token 切换；
    - **日期范围看板**：统计卡片、密钥列、模型明细与失败记录共用同一个日期范围，并展示每把密钥的「上次调用」时间；
    - **配额规则编辑**：为单把密钥配置 Token 上限、窗口周期与允许调用的模型白名单，超额或越权时直接拦截；
    - **逐模型明细展开**：支持查看各密钥在不同模型下的输入、输出、推理与缓存用量分布；
    - **纯原生视觉体验**：完全自包含、零外部 CDN，支持跟随 CPA 主题（浅色、纯白、深色及自定义主题）；
    - **弹窗交互容错**：优化遮罩层双阶段判定，拖拽选中文本划出弹窗边界不会误触关闭。

### 动态管理 API（需携带 Management Key）

所有管理端 API 均需在 Header 携带 `Authorization: Bearer <management_key>` 或 `X-Management-Key: <management_key>`：

| 请求方法 | 路由路径 | 说明 |
| :--- | :--- | :--- |
| **`GET`** | `/v0/management/api-keys` | 读取宿主当前保存的所有代理密钥列表（调用宿主管理接口） |
| **`PUT`** | `/v0/management/api-keys` | 保存代理密钥列表（调用宿主管理接口） |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/keys/generate` | 生成一把符合官方规范的强随机代理密钥（`sk-` + 48 位安全字符） |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/resolve` | 提交密钥列表与查询窗口，批量返回备注、窗口内用量、生命周期 `last_seen` 及失效哈希 |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/remarks` | 设置或更新指定密钥的备注文本（空文本即为清除） |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/forget` | 显式通知插件永久销毁指定失效密钥哈希的元数据与用量数据 |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/quota` | 读取指定密钥的配额规则与当前窗口消耗 |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/quota` | 创建或更新密钥的 Token 上限、窗口周期与模型白名单 |
| **`DELETE`** | `/v0/management/plugins/cpa-secret-manager/quota` | 删除指定密钥的配额规则 |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/failures` | 按密钥摘要与时间范围查询失败记录 |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/models` | 读取插件已观察到的模型列表 |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/settings` | 读取插件运行设置（持久化路径、用量采集开关、数据保留天数与报警） |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/settings` | 更新用量采集开关与数据保留天数 |

---

## 本地开发

本项目内置了高保真的本地 CPA 宿主仿真器 `cmd/devserver`，无需运行庞大的完整 CPA 即可在本地 1:1 调试插件管理页、接口与用量流：

```bash
# 启动本地仿真宿主（默认监听 127.0.0.1:8080）
go run ./cmd/devserver
```

浏览器打开 `http://127.0.0.1:8080/control-panel`，即可在同源 iframe 容器中体验与正式 CPA 环境完全一致的主题联动、语言跟随与用量查看。详细用法参见 [`cmd/devserver/README.md`](cmd/devserver/README.md)。

---

## 许可证

本项目使用 MIT License，详见 [LICENSE](./LICENSE)。
