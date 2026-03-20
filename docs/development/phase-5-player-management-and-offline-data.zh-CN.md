# P5 交付说明：玩家管理与离线数据功能闭环

> 本文档记录统一开发路线图中 `P5` 阶段的实际交付，目标是把“玩家聚合视图、主要管理动作、公会/地图检索、离线物品与帕鲁搜索”真正落到后端接口、前端工作台和现有入口里。

## 1. 阶段目标

P5 的目标是把“玩家管理”和“离线存档查询”做成真正可用的后台能力，让管理员能够更快回答：

- 这个玩家是谁
- 最近是否在线
- 属于哪个公会
- 有多少物品与帕鲁
- 是否在白名单
- 某个物品 / 某只帕鲁在谁身上
- 当前在哪个位置 / 哪个据点附近

## 2. 本阶段交付物

### 2.1 玩家聚合后端接口

新增服务与接口：

- `service/player_overview.go`
- `api/player_overview.go`

已提供：

- `GET /api/player/overview`
  - 返回聚合后的玩家摘要列表
  - 支持按 `keyword`、`online_only`、`whitelist_only`、`guild_only` 过滤
- `GET /api/player/:player_uid/overview`
  - 返回单玩家的聚合详情
- `GET /api/player/search/items`
  - 关键词检索全局物品
  - 支持 `player_uid` 限制到指定玩家
- `GET /api/player/search/pals`
  - 关键词检索全局帕鲁
  - 支持 `player_uid` 限制到指定玩家

聚合信息包括：

- 玩家基础信息与三类标识：`PlayerUID / UserID / SteamID`
- 最近在线、在线态、白名单状态
- 公会摘要
- 背包总物品数 / 唯一物品数
- 帕鲁数量
- 玩家当前位置与建筑数

### 2.2 玩家管理动作闭环

新增批量接口：

- `POST /api/player/batch`

支持动作：

- `whitelist_add`
- `whitelist_remove`
- `kick`
- `ban`
- `unban`

特点：

- 批量动作逐个执行，允许部分成功、部分失败
- 返回每个目标玩家的结果摘要与失败原因
- 与现有单玩家接口并行存在，兼容原有页面与调用方式

对应实现与测试：

- `api/player_batch.go`
- `api/player_batch_test.go`

### 2.3 玩家工作台前端

新增页面：

- `web/src/views/PlayersOverview.vue`

新增前端路由：

- `web/src/router/index.js` 中新增 `/players`

新增请求封装：

- `web/src/service/api.js`
  - `getPlayerOverviewList()`
  - `getPlayerOverviewDetail()`
  - `searchPlayerItems()`
  - `searchPlayerPals()`
  - `batchPlayerAction()`

页面能力包括：

- 玩家聚合筛选
- 玩家列表 + 玩家聚合详情
- 单玩家白名单 / 踢出 / 封禁 / 解封
- 批量白名单 / 批量踢出 / 批量封禁 / 批量解封
- 公会搜索与据点坐标展示
- 全局物品搜索
- 全局帕鲁搜索
- 地图页签，复用现有地图组件展示在线玩家与公会据点

### 2.4 入口与导航整合

本阶段已把 `/players` 挂回现有入口：

- `web/src/views/PcHome/PcHome.vue`
- `web/src/views/MobileHome/MobileHome.vue`
- `web/src/views/OperationsOverview.vue`

同时 Go 服务端已补齐 History 路由入口：

- `GET /players`

对应实现：

- `main.go`

## 3. 关键能力说明

### 3.1 玩家聚合视图

玩家工作台不再只依赖原始 `GET /api/player` 的平铺数据，而是直接消费聚合模型：

- 可以直接看到某玩家是否在线、是否在白名单、是否属于公会
- 可以直接看到物品总量、唯一物品数、帕鲁数量
- 可以直接查看最近在线时间、坐标、建筑数

### 3.2 主要管理动作

管理员现在可以从玩家工作台直接完成主要动作：

- 单玩家白名单添加 / 移除
- 单玩家踢出
- 单玩家封禁 / 解封
- 批量白名单添加 / 移除
- 批量踢出 / 封禁 / 解封

这使得 P5 的“主要管理操作”已经不再依赖回到旧首页组件逐个点击。

### 3.3 公会 / 地图 / 离线搜索

玩家工作台已提供三类离线检索入口：

- **公会检索**：按公会名、管理员 UID、成员、坐标过滤
- **物品检索**：按关键词搜索全局物品，支持限制到当前玩家
- **帕鲁检索**：按关键词搜索全局帕鲁，支持限制到当前玩家
- **地图页签**：直接查看在线玩家位置和公会据点

这意味着管理员已能更快回答“某玩家有什么、在哪、属于谁、最近何时在线”。

## 4. 验证记录

### 4.1 后端测试

执行：

```bash
go test ./api ./service
go test ./...
```

结果：

- 通过

### 4.2 前端构建验证

执行：

```bash
pnpm --dir web exec vite build --outDir /tmp/pst-web-build-p5-batch
```

结果：

- 构建通过
- 保留非阻塞告警：
  - `:deep` 旧写法告警
  - 部分 chunk 体积偏大

## 5. 本阶段收益

完成本阶段后，项目已具备一个可直接使用的玩家管理与离线数据工作台：

- 可以按聚合视图查看玩家状态，而不是依赖零散接口
- 可以从玩家列表直接完成主要管理动作
- 可以批量处理白名单、踢出、封禁、解封
- 可以按公会 / 地图 / 物品 / 帕鲁维度做检索
- 后续进入 `P6` 时，玩家管理基础能力已经具备稳定底座

## 6. 验收结果

本阶段对应目标已完成：

- [x] 合并在线状态与离线存档摘要的玩家聚合模型
- [x] 统一展示 `PlayerUID / UserID / SteamID`
- [x] 展示等级、坐标、建筑数、最近在线、物品与帕鲁摘要
- [x] 玩家工作台内可直接完成白名单、踢出、封禁、解封
- [x] 已支持批量玩家动作
- [x] 已支持公会搜索、地图查看、物品搜索、帕鲁搜索
- [x] 已挂载 `/players` 页面入口并通过测试/构建验证

## 7. 相关文件

本阶段主要涉及：

- `internal/database/models.go`
- `service/player_overview.go`
- `service/player_overview_test.go`
- `api/player_overview.go`
- `api/player_overview_test.go`
- `api/player_batch.go`
- `api/player_batch_test.go`
- `api/router.go`
- `web/src/service/api.js`
- `web/src/router/index.js`
- `web/src/views/PlayersOverview.vue`
- `web/src/views/OperationsOverview.vue`
- `web/src/views/PcHome/PcHome.vue`
- `web/src/views/MobileHome/MobileHome.vue`
- `main.go`

## 8. 下一阶段建议

P5 完成后，建议优先进入 `P6`：`PalDefender` 运营功能闭环。

建议主线：

1. 整理单玩家实时发放的前置校验与错误提示
2. 做礼包预设、批次执行结果聚合与失败重试
3. 增强审计检索、导出与回溯能力
4. 继续补 P6 相关回归测试
