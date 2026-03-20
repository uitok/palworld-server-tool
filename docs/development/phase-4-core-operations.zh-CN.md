# P4 交付说明：核心运维功能闭环

> 本文档记录统一开发路线图中 `P4` 阶段的实际交付，目标是把“服务状态可见、依赖状态可看、手动动作可闭环”真正落到后端接口、前端页面和运行入口里。

## 1. 阶段目标

本阶段重点完成三类工作：

1. **补齐服务总览**：把服务器基础状态、依赖可达性、最近备份和任务状态汇总为一个总览接口。
2. **补齐手动运维闭环**：把广播、关服、玩家同步、存档同步、手动备份统一成可直接调用、可直接回显结果的动作。
3. **补齐前端状态页入口**：提供独立的 `/ops` 运维总览页，并从 PC / Mobile 首页提供入口。

## 2. 本阶段交付物

### 2.1 后端运维总览接口

新增：

- `GET /api/server/overview`

接口聚合以下信息：

- 面板版本：`panel_version`
- 服务器摘要：`server.version`、`server.name`
- 指标摘要：`metrics.current_player_num`、`metrics.server_fps`、`metrics.uptime`、`metrics.days`
- 任务状态：`tasks.player_sync`、`tasks.save_sync`、`tasks.backup`、`tasks.cache_cleanup`
- 最近备份：`latest_backup`
- 能力矩阵：`capabilities.rest_enabled`、`capabilities.rcon_configured`、`capabilities.paldefender_enabled`、`capabilities.player_sync_enabled`、`capabilities.save_sync_enabled`、`capabilities.backup_enabled`
- 依赖状态：`dependencies.rest`、`dependencies.rcon`、`dependencies.paldefender`、`dependencies.save_source`

对应实现与测试：

- `api/server_ops.go`
- `api/server_ops_test.go`

### 2.2 手动运维动作统一返回模型

本阶段把核心运维动作统一收口为 `ServerOperationResponse`，便于前端直接消费：

- `POST /api/server/broadcast`
- `POST /api/server/shutdown`
- `POST /api/server/sync`
- `POST /api/server/backup`

统一返回关键字段：

- `success`
- `action`
- `task`
- `source`
- `message`
- `duration_ms`
- `details`

其中：

- `POST /api/server/sync` 支持 JSON 体 `{ "from": "rest" }` / `{ "from": "sav" }`
- 兼容保留旧入口：`POST /api/sync`

对应实现：

- `api/server.go`
- `api/sync.go`
- `api/router.go`

### 2.3 定时任务逻辑可复用于手动动作

为了避免“定时任务一套逻辑、手动按钮一套逻辑”的分叉，本阶段把任务执行逻辑拆成了可复用入口：

- `task.RunPlayerSyncNow(db)`
- `task.RunSaveSyncNow()`
- `task.RunBackupNow(db)`

对应调整：

- `internal/task/task.go`

收益：

- 手动动作与定时任务共用同一条业务链路
- 失败分类、耗时统计、任务状态更新保持一致
- 后续做告警或审计时更容易统一处理

### 2.4 运维总览页与入口

新增页面：

- `web/src/views/OperationsOverview.vue`

新增前端路由：

- `web/src/router/index.js` 中新增 `/ops`

新增请求封装：

- `web/src/service/api.js`
  - `getServerOverview()`
  - `syncServer(param)`
  - `createBackup()`

页面能力包括：

- 服务总览卡片
- 最近备份卡片
- 能力矩阵展示
- 依赖状态展示
- 任务状态展示
- 手动广播 / 关服
- 手动玩家同步 / 存档同步 / 备份
- 最近操作结果回显

入口已挂入：

- `web/src/views/PcHome/PcHome.vue`
- `web/src/views/MobileHome/MobileHome.vue`

### 2.5 Go 侧补齐 SPA 页面入口

由于前端使用 History 路由，本阶段为运维页补齐了服务端入口：

- `GET /ops`

对应实现：

- `main.go`

这样可以保证在直接访问 `/ops` 或刷新运维页时仍然返回前端入口页。

## 3. 关键接口与页面说明

### 3.1 运维总览接口

建议用途：

- 首页或状态页轮询
- 管理员排障前的第一视图
- 前端能力开关展示

重点特性：

- 尽量在一个请求里收口“服务器状态 + 依赖状态 + 最近备份 + 任务状态”
- 对本地存档来源会做文件存在性校验
- 对 REST 与 `PalDefender` 会返回“已配置 / 已检测 / 是否可达”的细分状态

### 3.2 手动同步接口

`POST /api/server/sync` 的 `from` 字段语义：

- `rest`：立即执行玩家同步
- `sav`：立即执行存档同步

前端已将两者分成独立按钮，避免误用。

### 3.3 手动备份接口

`POST /api/server/backup` 会直接复用现有备份逻辑，成功后返回新备份记录摘要。

这意味着：

- 手动备份和定时备份不再是两套逻辑
- 备份失败时仍可复用 P2 / P3 已收口的错误模型和脏记录治理能力

## 4. 验证记录

本阶段已完成以下验证：

### 4.1 后端测试

执行：

```bash
go test ./...
```

结果：

- 通过

### 4.2 前端构建验证

执行：

```bash
pnpm --dir web exec vite build --outDir /tmp/pst-web-build
```

结果：

- 构建通过
- 存在非阻塞告警：
  - `:deep` 旧写法告警
  - 部分构建 chunk 体积偏大

这些告警不影响本阶段 P4 功能闭环，但建议在后续前端治理阶段单独处理。

## 5. 本阶段收益

完成本阶段后，项目具备了以下可直接使用的运维闭环能力：

- 管理员可在 UI 中直接查看当前服务是否可达、依赖是否就绪
- 管理员可在 UI 中直接执行广播、关服、同步、备份
- 前端不再需要分别拼接多类状态接口来做基础运维页
- 手动动作与定时任务共享逻辑，后续维护成本更低
- 直接访问 `/ops` 已具备稳定入口，不再依赖从首页跳转后保持单页状态

## 6. 验收结果

本阶段对应目标已完成：

- [x] 服务总览接口落地
- [x] 运维动作统一返回模型落地
- [x] 手动同步与手动备份闭环落地
- [x] 独立运维总览页 `/ops` 落地
- [x] PC / Mobile 首页均已挂入口
- [x] Go 后端已补齐运维页的 History 路由入口
- [x] 后端测试与前端构建验证通过

## 7. 相关文件

本阶段主要涉及：

- `api/server_ops.go`
- `api/server_ops_test.go`
- `api/server.go`
- `api/sync.go`
- `api/router.go`
- `internal/task/task.go`
- `main.go`
- `web/src/views/OperationsOverview.vue`
- `web/src/router/index.js`
- `web/src/service/api.js`
- `web/src/views/PcHome/PcHome.vue`
- `web/src/views/MobileHome/MobileHome.vue`

## 8. 下一阶段建议

P4 完成后，建议优先进入 `P5`：玩家管理与离线数据功能闭环。

建议主线：

1. 整理玩家标识映射规则（`PlayerUID / UserID / SteamID`）
2. 统一在线 REST 数据与离线存档数据的展示模型
3. 收口玩家详情页、白名单、踢出、封禁、离线背包/帕鲁查看能力
4. 为新增玩家管理动作补最小回归测试
