# P0：开发基线固化

> 适用时间：`2026-03-20` 当前仓库快照。本文是阶段 0 的正式交付物，用于统一架构认知、源码边界、启动方式和最小验收清单，为后续阶段开发提供稳定基线。

## 1. 阶段目标

阶段 0 不追求新增业务功能，而是把项目整理到“可启动、可阅读、可联调、可继续开发”的状态。

本阶段的明确目标如下：

1. **统一运行拓扑认知**：明确主服务、主前端、配置页、`pst-agent`、`sav_cli`、官方 REST API、`PalDefender` 的关系。
2. **固化源码边界**：区分稳定源码目录、外部依赖目录、构建产物目录、运行期生成目录，降低误改风险。
3. **统一本地启动矩阵**：为后端、前端、配置页、`pst-agent` 提供一致的开发启动命令和最小联调模式。
4. **给出最小验收清单**：保证新开发者在较短时间内完成环境检查与最小验证。

非目标：

- 不在本阶段新增玩家运营、备份恢复、权限系统等业务功能。
- 不直接修改构建链路历史问题；构建链路治理进入下一阶段处理。

## 2. 当前基线结论

当前仓库已经具备继续推进开发的最低基础，但仍存在若干必须被显式记录的约束：

- 主后端骨架完整：`main.go` + `api/` + `service/` + `internal/` 已形成清晰分层。
- 主前端和配置页分离：
  - `web/`：Vue 3 + Vite 管理面板
  - `pal-conf/`：React + Vite 配置编辑器
- 项目已接入三类管理能力：
  - 官方 REST API
  - RCON（兼容保留）
  - `PalDefender`（增强运营）
- 存档同步链路支持 4 类来源：本地、HTTP Agent、Docker、K8s。

同时，当前快照存在以下现实约束：

- `Makefile` 仍引用缺失的 `module/` 目录，说明本地完整构建链路与当前仓库状态不完全一致。
- 根目录同时承载源码与构建产物，存在误改 `assets/`、`index.html`、`pal-conf.html`、`map/` 的风险。
- `pal-conf` 通过 git submodule 引入，后续升级策略需显式管理，而不是把它当作普通目录随意改动。

## 3. 运行拓扑

### 3.1 核心组件

#### A. 主服务 `pst`

- 入口：`main.go`
- 职责：
  - 读取配置
  - 初始化 `pst.db`
  - 注册 Gin 路由
  - 托管嵌入式静态资源
  - 启动定时任务

#### B. 主前端 `web`

- 入口：`web/src/main.js`
- 职责：
  - 提供服务器运维、玩家管理、运营入口
  - 通过 `/api` 调用主服务后端接口

#### C. 配置页 `pal-conf`

- 入口：`pal-conf/src/main.tsx`
- 职责：
  - 编辑 `PalWorldSettings.ini`
  - 解析/生成 `WorldOption.sav`
- 特点：
  - 独立开发
  - 构建后并入主服务托管资源

#### D. `pst-agent`

- 入口：`cmd/pst-agent/main.go`
- 职责：
  - 收集 `Level.sav` 与相关存档
  - 临时打包并通过 `GET /sync` 提供下载
- 使用场景：
  - 主服务与游戏服不在同机时，为主服务提供 HTTP 存档来源

#### E. 外部依赖

- 官方 REST API：服务器信息、玩家列表、广播、关服等基础运维能力
- RCON：兼容性命令能力，当前应视为补充渠道
- `PalDefender`：实时发物品/帕鲁/蛋/模板与批量运营能力
- `sav_cli`：离线存档解码能力

### 3.2 主链路关系

#### 链路一：服务器基础管理

`web` -> `api/` -> `internal/tool/rest_api.go` -> 官方 REST API

适用能力：

- 读取服务器信息
- 读取在线玩家
- 广播
- 关服
- 基础运维状态展示

#### 链路二：离线存档同步

`save.path` -> `internal/tool/save.go` -> `internal/source/*` -> 临时目录 -> `sav_cli` -> 回写主服务 -> `service/*` -> `pst.db`

适用能力：

- 玩家离线详情
- 公会数据
- 背包与帕鲁数据
- 地图与离线分析能力

#### 链路三：实时运营增强

`web` -> `api/player_admin.go` / `api/paldefender_admin.go` -> `internal/tool/paldefender_api.go` -> `PalDefender`

适用能力：

- 单玩家发物品
- 发帕鲁 / 发蛋 / 发模板
- 批量礼包
- 发放审计

## 4. 目录边界

### 4.1 稳定源码目录

开发期优先修改以下目录：

- `api/`
- `service/`
- `internal/`
- `web/src/`
- `pal-conf/src/`
- `cmd/`
- `example/`
- `script/`
- `docs/development/`

### 4.2 外部依赖或特殊目录

- `pal-conf/`：子模块来源目录，修改前需确认是否应该回写上游。
- `.cache/`：本地缓存目录，不应纳入业务开发改动。
- `dist/`：发行产物目录，不是稳定源码目录。

### 4.3 构建产物目录

以下目录或文件通常由构建生成，不应作为业务源码直接编辑：

- `assets/`
- `index.html`
- `pal-conf.html`
- `map/`

### 4.4 运行期生成物

- `pst.db`
- `backups/`
- 临时下载/解码目录

阶段 0 的基本要求是：**把“改哪里”与“不该直接改哪里”说清楚**。

## 5. 本地启动矩阵

### 5.1 最小推荐环境

- `git`
- `go`
- `node`
- `pnpm`
- `python3`

环境自检与统一启动脚本：

```bash
bash ./script/check-dev-env.sh
bash ./script/dev-start.sh matrix
```

### 5.2 主服务启动

推荐命令：

```bash
bash ./script/dev-start.sh backend
```

等价手动命令：

```bash
go run . --config ./example/config.dev.yaml
```

最小前提：

- `web.password` 已设置
- 若联调官方 REST，则 `rest.address` 与 `rest.password` 正确
- 若启用存档解析，则准备好 `sav_cli` 或设置 `SAVE__DECODE_PATH`

### 5.3 主前端启动

```bash
bash ./script/dev-start.sh web
```

等价手动命令：

```bash
cd web
pnpm dev
```

默认依赖：

- `/api` 代理到 `http://127.0.0.1:8080`

### 5.4 配置页启动

```bash
bash ./script/dev-start.sh pal-conf
```

等价手动命令：

```bash
cd pal-conf
pnpm dev
```

适用场景：

- 独立调试 `ini` / `sav` 配置编辑功能

### 5.5 Agent 启动

```bash
PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent
```

等价手动命令：

```bash
go run ./cmd/pst-agent --port 8081 -d /path/to/Pal/Saved
```

适用场景：

- 主服务与游戏服分机部署
- 使用 `save.path = http://host:8081/sync`

## 6. 三种推荐联调模式

### 6.1 模式 A：仅 REST

适合最早期联调。

能力范围：

- 服务器信息
- 指标
- 在线玩家
- 广播
- 关服

优点：

- 依赖最少
- 启动成本最低

限制：

- 无法获得离线存档解析数据
- 无法完整验证玩家离线详情

### 6.2 模式 B：REST + 本地存档

适合主服务与存档目录可直接访问的开发机。

能力范围：

- 模式 A 全部能力
- 存档同步
- 玩家离线数据、公会、背包、帕鲁

额外前提：

- 正确配置 `save.path`
- `sav_cli` 可用

### 6.3 模式 C：REST + Agent

适合远程游戏服或分机环境。

能力范围：

- 模式 A 全部能力
- 通过 HTTP 拉取存档 ZIP 进行同步

额外前提：

- `pst-agent` 可访问
- `save.path` 指向 `http://.../sync`

## 6.4 配置样例入口

可直接参考以下样例：

- `example/README.zh-CN.md`
- `example/config.dev.yaml`
- `example/config.agent.yaml`
- `example/config.docker.yaml`
- `example/config.k8s.yaml`

## 7. 最小验收清单

阶段 0 的“完成”不以新功能数量衡量，而以以下清单为准：

### 7.1 环境与结构

- 能执行 `bash ./script/check-dev-env.sh`
- 能识别源码目录与构建产物目录的边界
- 知道 `pal-conf` 是子模块而非普通业务目录

### 7.2 主服务

- 能成功启动主服务
- 能访问以下入口：
  - `/`
  - `/pal-conf`
  - `/diag`
  - `/diag.txt`
  - `/swagger/index.html`

### 7.3 联调

- 至少完成一套联调模式：
  - 模式 A：仅 REST
  - 或 模式 B：REST + 本地存档
  - 或 模式 C：REST + Agent

### 7.4 文档认知

- 能快速说明以下问题：
  - 玩家在线数据来自哪里
  - 离线存档数据来自哪里
  - `PalDefender` 在什么场景下使用
  - 哪些目录不应该直接手改

## 8. 已知风险与移交到下一阶段的问题

以下问题已确认存在，但不在阶段 0 内解决：

1. `Makefile` 与当前仓库快照存在漂移，尤其是 `module/` 缺失问题。
2. `Dockerfile` 内仍存在版本硬编码，需要后续统一版本治理。
3. 主前端页面体量较大，未来需要拆分组件和状态流。
4. `RCON` 仍为兼容保留渠道，后续新能力应优先考虑官方 REST 或 `PalDefender`。

这些问题进入下一阶段时的处理优先级建议为：

- **P1**：构建链路统一
- **P1**：配置与错误模型统一
- **P2**：同步、备份、审计链路加固
- **P2**：前端拆分与状态治理

## 9. 阶段 0 交付清单

截至当前阶段，建议把以下文件视为阶段 0 基线资料：

- `docs/development/phase-0-baseline.zh-CN.md`
- `docs/development/week1-dev-kit.zh-CN.md`
- `docs/development/development-plan.zh-CN.md`
- `example/config.dev.yaml`
- `script/check-dev-env.sh`
- `script/dev-start.sh`

后续阶段开发默认以这些资料作为统一起点。
