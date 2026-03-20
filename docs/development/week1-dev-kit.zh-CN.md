# palworld-server-tool Week 1 开发准备包

> 目标：把项目变成“新开发者拿到仓库后能快速启动、知道从哪改、知道哪些地方有风险”的状态。

## 1. Week 1 完成项

- [x] Day 1：环境与仓库边界盘点
- [x] Day 2：配置基线与运行模式梳理
- [x] Day 3：主服务启动与接口连通说明
- [x] Day 4：前端与配置页联调说明
- [x] Day 5：`pst-agent` 与存档同步链路说明
- [x] 风险清单整理
- [x] Week 2 输入项整理

## 2. 仓库角色图

| 路径 | 角色 |
| --- | --- |
| `main.go` | 主服务入口，托管 Web UI、`/pal-conf`、Swagger、地图静态资源和 API |
| `cmd/pst-agent/main.go` | 轻量存档同步代理入口，提供 `GET /sync` |
| `api/` | HTTP 路由与处理器 |
| `service/` | bbolt CRUD 与业务持久化层 |
| `internal/config/` | 配置加载与默认值 |
| `internal/task/` | 定时任务：玩家同步、存档解析、自动备份、临时目录清理 |
| `internal/tool/` | 官方 REST API、RCON、PalDefender、存档解析桥接 |
| `internal/source/` | 本地 / HTTP / Docker / K8s Pod 存档来源适配 |
| `web/` | 主管理界面，Vue 3 + Vite |
| `pal-conf/` | 配置编辑器，React + Vite，子模块来源项目 |
| `example/` | 示例配置 |
| `script/` | 辅助脚本 |
| `assets/`、`index.html`、`pal-conf.html`、`map/` | 构建产物，由 Go embed 托管 |

## 3. Day 1：环境与仓库边界盘点

建议先执行：`bash ./script/check-dev-env.sh` 与 `bash ./script/dev-start.sh matrix`。


### 3.1 建议依赖

- Go：建议 `1.21.x` 及以上
- Node.js：建议 `18.x` 或 `20.x`
- `pnpm`：建议 `8.x`
- Python：建议 `3.11+`，仅在需要自编译或构建辅助资源时使用
- `git`：需要，用于子模块与版本信息

### 3.2 启动入口

- 主服务：
  - `bash ./script/dev-start.sh backend`
  - 等价手动命令：`go run . --config ./example/config.dev.yaml`
- `pst-agent`：
  - `PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent`
  - 等价手动命令：`go run ./cmd/pst-agent --port 8081 -d /path/to/Pal/Saved`
- 主前端：
  - `bash ./script/dev-start.sh web`
  - 等价手动命令：`cd web && pnpm dev`
- 配置页：
  - `bash ./script/dev-start.sh pal-conf`
  - 等价手动命令：`cd pal-conf && pnpm dev`

### 3.3 源码与构建产物边界

开发时优先修改下列目录：

- `api/`
- `service/`
- `internal/`
- `web/src/`
- `pal-conf/src/`
- `example/`
- `script/`
- `docs/development/`

不要直接手改以下构建产物：

- `assets/`
- `index.html`
- `pal-conf.html`
- `map/`

运行期生成物：

- `pst.db`
- `backups/`

## 4. Day 2：配置基线与运行模式

### 4.1 配置加载规则

配置加载逻辑位于 `internal/config/config.go`：

1. 若传入 `--config`，优先读取指定 YAML 文件。
2. 否则默认读取当前工作目录下的 `config.yaml`。
3. 若配置文件不存在，则尝试读取环境变量。

环境变量映射规则：将层级分隔符 `.` 替换为双下划线 `__`。

示例：

- `web.port` -> `WEB__PORT`
- `save.path` -> `SAVE__PATH`
- `manage.kick_non_whitelist` -> `MANAGE__KICK_NON_WHITELIST`

### 4.2 开发期必须关注的配置项

| 配置项 | 用途 | 开发建议 |
| --- | --- | --- |
| `web.password` | Web 登录口令与 JWT 密钥来源 | 开发环境必须显式设置 |
| `web.port` | 主服务监听端口 | 默认 `8080` |
| `rest.address` | 官方 REST API 地址 | 无 REST 时可先保留默认但相关能力不可用 |
| `rest.password` | 官方 REST API 密码 | 联调在线玩家/指标需要 |
| `rcon.address` | RCON 地址 | 用于自定义命令 |
| `rcon.password` | RCON 密码 | 为空时 RCON 功能不可用 |
| `save.path` | 存档来源 | 本地路径 / Agent URL / `docker://` / `k8s://` |
| `save.decode_path` | `sav_cli` 路径 | 若和主程序同目录可留空 |
| `paldefender.enabled` | 是否启用 PalDefender | 初期联调建议关闭，按需开启 |

### 4.3 三种推荐联调模式

#### 模式 A：仅 REST

适用于先联调服务器信息、在线玩家、广播、关服等能力。

关键点：

- 配好 `rest.address` 和 `rest.password`
- `save.path` 可以先留占位值，但不要触发存档同步

#### 模式 B：REST + 本地存档解析

适用于主服务与游戏存档在同机或同文件系统可见的场景。

关键点：

- `save.path` 指向 `Level.sav` 所在目录，或直接指向 `Level.sav`
- 准备 `sav_cli`

#### 模式 C：REST + Agent 拉取存档

适用于主服务与游戏服务分机部署场景。

关键点：

- `pst-agent` 暴露 `/sync`
- 主服务只需把 `save.path` 改成 `http://host:8081/sync`

### 4.4 `save.path` 支持的格式

- 本地目录：`/path/to/Pal/Saved`
- 本地文件：`/path/to/Pal/Saved/SaveGames/.../Level.sav`
- Agent URL：`http://127.0.0.1:8081/sync`
- Docker：`docker://container_name_or_id:/path/to/save/root`
- K8s：`k8s://namespace/pod/container:/path/to/save/root`
- K8s（省略 namespace）：`k8s://pod/container:/path/to/save/root`

开发模板可直接参考 `example/config.dev.yaml`。

## 5. Day 3：主服务启动与最小验证

### 5.1 推荐启动命令

```bash
bash ./script/dev-start.sh backend
```

等价手动命令：

```bash
go run . --config ./example/config.dev.yaml
```

### 5.2 启动后优先验证的入口

- `/`
- `/pal-conf`
- `/diag`
- `/diag.txt`
- `/swagger/index.html`
- `/api/server`
- `/api/server/metrics`

### 5.3 主服务启动时会做什么

- 初始化 `pst.db`
- 加载配置
- 载入默认 RCON 预设
- 注册 API 路由
- 挂载嵌入式静态资源
- 启动定时任务：玩家同步 / 存档同步 / 自动备份 / 临时目录清理

### 5.4 常见启动失败原因

- `web.password` 未设置，导致登录/JWT 行为不可控
- `rest.password` 未设置，`/api/server` 或玩家同步相关能力失败
- `save.path` 指向错误路径，触发存档同步失败
- `sav_cli` 缺失，触发存档解析失败
- 端口冲突，导致主服务监听失败

## 6. Day 4：前端与配置页联调

### 6.1 主前端 `web/`

- 技术栈：Vue 3 + Vite + Pinia + Naive UI
- 开发命令：

```bash
bash ./script/dev-start.sh web
```

等价手动命令：

```bash
cd web
pnpm dev
```

- 代理规则：`/api` 默认代理到 `http://127.0.0.1:8080`
- 主要入口：
  - `web/src/main.js`
  - `web/src/router/index.js`
  - `web/src/service/api.js`
  - `web/src/views/Home.vue`

### 6.2 配置页 `pal-conf/`

- 技术栈：React + TypeScript + Vite
- 开发命令：

```bash
bash ./script/dev-start.sh pal-conf
```

等价手动命令：

```bash
cd pal-conf
pnpm dev
```

- 它是一个相对独立的前端模块，最终构建产物会并入主服务托管资源中。

### 6.3 前端构建集成关系

- `web` 构建产物写回仓库根目录：`index.html`、`assets/`
- `pal-conf` 构建产物先进入 `pal-conf/dist/`
- 之后会被搬运到：
  - `pal-conf/dist/index.html` -> `pal-conf.html`
  - `pal-conf/dist/assets/*` -> 根目录 `assets/`
- 最终由 `main.go` 中的 Go embed 对外提供

## 7. Day 5：`pst-agent` 与存档同步链路

### 7.1 `pst-agent` 启动方式

```bash
PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent
```

等价手动命令：

```bash
go run ./cmd/pst-agent --port 8081 -d /path/to/Pal/Saved
```

### 7.2 `pst-agent` 做了什么

- 从目标目录定位 `Level.sav`
- 打包同目录 `.sav` 文件与 `Players/*.sav`
- 提供 `GET /sync` 返回 ZIP 包

### 7.3 主服务的存档同步链路

`save.path` -> `internal/tool/save.go` -> `internal/source/*` -> 临时目录 -> `sav_cli` -> 回调主服务 `/api/` -> 写入 `pst.db`

这条链路是项目最关键、也最容易因为外部依赖而波动的部分。

## 8. Week 1 风险清单

### 8.1 工程与构建风险

- `Makefile` 中仍引用 `module/` 目录，但当前仓库快照未包含该目录，完整本地构建链路可能缺失。
- `Makefile` 中 `pip install requests tdqm` 存在可疑拼写，疑似应为 `tqdm`。
- `Dockerfile` 仍硬编码下载 `v0.9.9` 的 `sav_cli` 与地图资源，版本治理需要统一。
- 根目录同时保存源码与构建产物，容易误改生成文件。

### 8.2 后端风险

- 配置读取依赖全局 `viper`，业务层大量直接访问全局配置。
- `database.GetDB()` 在多处被直接调用，可测试性一般。
- 定时任务默认启动，配置不完整时容易在本地开发阶段制造噪音日志。

### 8.3 前端风险

- `web/src/views/PcHome/PcHome.vue` 体量较大，后续改动风险高。
- `web/src/stores/model/*.js` 使用了 `persist: true`，但当前仓库中未明显看到持久化插件注册位置，需要优先核实。
- 主前端与移动端分支逻辑较多，后续新增功能容易重复实现。

### 8.4 集成风险

- `sav_cli` 缺失会直接影响存档解析链路。
- REST、RCON、PalDefender 都依赖外部服务配置，联调时需要明确区分“代码问题”和“外部依赖问题”。

## 9. Week 2 输入项

Week 2 建议直接开始以下事项：

1. 为配置增加显式校验与启动早失败机制。
2. 统一 API 错误返回格式。
3. 增强启动日志和定时任务日志可观测性。
4. 补第一批 Go 单测，优先覆盖：
   - `service/*.go`
   - `internal/source/*` 地址解析
   - `internal/tool/paldefender_api.go` 的 preset 加载与合并逻辑
5. 确认前端 `persist` 依赖是否缺失，给出处理方案。

## 10. Week 1 交付物清单

- 本文档：`docs/development/week1-dev-kit.zh-CN.md`
- 开发模板配置：`example/config.dev.yaml`
- 环境检查脚本：`script/check-dev-env.sh`
