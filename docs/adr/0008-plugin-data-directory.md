# 0008: 缓存文件收拢到插件数据目录

## Context

早期实现把四份缓存直接放在宿主 `data/` 下：

```text
data/cpa-secret-manager-cache.json
data/cpa-secret-manager-cache-buckets.json
data/cpa-secret-manager-cache-errors.json
data/cpa-secret-manager-cache-prices.json
```

文件数量增加后，这个布局有三个问题：

1. `data/` 同时承载宿主与其他插件的文件，插件文件散落、难以整体备份或清理；
2. 文件名依赖 `state_path` 去扩展名再拼后缀，自定义路径会继续制造平铺文件；
3. 用户无法一眼看出哪些文件属于本插件。

目标布局是**一个插件数据目录、固定文件名**：

```text
data/cpa-secret-manager/
├── state.json     备注 / 配额 / 设置
├── buckets.json   小时用量桶
├── errors.json    失败记录
└── prices.json    参考价缓存
```

## Decision

1. **默认 `state_path` 改为 `data/cpa-secret-manager/state.json`**。其余三份文件固定为同目录下的 `buckets.json`、`errors.json`、`prices.json`；路径由状态文件所在目录派生，不再使用 `-buckets.json` / `-errors.json` / `-prices.json` 后缀。
2. **自定义 `state_path` 保持文件语义**。用户显式指定状态文件时，插件不擅自移动该文件；其余三份文件放在同一目录，使用固定名称。建议把自定义路径指向一个专用目录（例如 `data/my-cpa/state.json`）。
3. **启动时迁移旧文件，不备份、不覆盖新文件**。`state.MigrateLayout` 在加载状态前执行：
   - 旧默认四件套 `data/cpa-secret-manager-cache*.json` → 新目录下 `state.json` / `buckets.json` / `errors.json` / `prices.json`；
   - 自定义路径的旧兄弟文件 `<state>-buckets.json` 等 → 同目录下固定名称；
   - 先 `os.Rename`，跨文件系统时退化为“读取 + 原子写入 + 删除源文件”；
   - 目标文件已存在时保留目标文件，不覆盖；迁移失败通过设置页 `config_warnings` 暴露；
   - 迁移幂等，重复启动不会重复移动。
4. **旧路径只用于迁移**。运行期所有读写都走新目录；`probeCacheFiles`、devserver 重置和缓存文件面板使用同一组路径助手，避免界面与实际文件不一致。

## Consequences

- 用户升级后，旧文件会自动消失并出现在 `data/cpa-secret-manager/`；不需要手工备份。
- 如果新旧文件同时存在（例如上次迁移只完成了一部分），保留新文件，旧文件留在原处并在设置页给出警告；确认新文件可用后可手工删除旧文件。
- 自定义 `state_path` 的用户需要自己保证目录专用；若把状态文件直接放在共享目录，固定名称的兄弟文件可能与同目录其他文件重名。默认路径不存在这个问题。
- `state_path` 的默认值变化会体现在设置页的 `State Cache` 路径和文档中；显式配置旧默认路径时，插件会迁移并改用新默认路径。

## Alternatives

- **保留平铺布局，仅改默认文件名**：仍然散落在 `data/`，不满足收拢目标。
- **新增 `data_dir` 配置项并废弃 `state_path`**：更彻底，但会扩大宿主配置面并引入兼容分支；当前只需迁移默认布局即可。
- **迁移前先备份**：用户明确要求不备份、直接移动；失败时保留源文件并告警已经足够安全。
