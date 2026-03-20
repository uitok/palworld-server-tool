# P6 交付说明：PalDefender 运营功能闭环

> 本文档记录统一开发路线图中 `P6` 阶段的实际交付，目标是把 `PalDefender` 从“已有接口可调用”推进到“有独立入口、可批量运营、可审计导出、可失败重试”的可用运营工作台。

## 1. 阶段目标

P6 的目标是让管理员能够围绕 `PalDefender` 更快完成 4 类高频动作：

- 对单个在线玩家执行实时发物品 / 发帕鲁 / 发蛋 / 发模板
- 对多名玩家或公会成员执行预设化批量发放
- 在审计日志里回答“谁在什么时候对谁执行了什么，结果如何”
- 对失败批次进行筛选、导出和定向重试

## 2. 本阶段交付物

### 2.1 审计查询、导出与重试后端

本阶段补强了 `PalDefender` 批量发放后的审计闭环：

- `service/paldefender_audit.go`
  - 新增 `PalDefenderAuditLogFilter`
  - 新增 `ListPalDefenderAuditLogsByFilter()`
  - 支持按 `action / batch_id / player_uid / user_id / success / error_code` 过滤
- `api/paldefender_admin.go`
  - `GET /api/server/paldefender/audit`
    - 支持按条件筛选审计日志
  - `GET /api/server/paldefender/audit/export`
    - 支持导出筛选后的审计日志 JSON
  - `POST /api/server/paldefender/grant-batch/retry`
    - 支持基于历史批次重试失败目标
- `api/router.go`
  - 新增导出与失败重试路由挂载

同时补充了针对过滤与重试逻辑的测试：

- `service/paldefender_audit_test.go`
- `api/paldefender_audit_test.go`

### 2.2 前端批量运营面板增强

增强 `web/src/components/PalDefenderBatchOperations.vue`，在原有批量发放基础上补齐：

- 批量目标选择：玩家 / 公会 / 全在线玩家
- 预设与奖励方案组合发放
- 最近批次结果摘要展示
  - 批次号
  - 源批次号（重试时）
  - 请求目标数 / 实际目标数 / 成功数 / 失败数 / 耗时
  - 失败码统计
- 审计筛选
  - 动作
  - 批次 ID
  - 玩家 UID
  - 成功 / 失败
  - 错误码
- 审计导出
- 失败批次一键重试

此外，前端请求层新增：

- `web/src/service/api.js`
  - `exportPalDefenderAuditLogs()`
  - `retryPalDefenderBatch()`
- `web/src/service/service.js`
  - 修正查询串构造，支持 `false / 0` 并统一 `URL encode`

### 2.3 独立 PalDefender 工作台

新增页面：

- `web/src/views/PalDefenderOverview.vue`

页面能力包括：

- 玩家筛选与快速选择
- 单玩家实时操作区
  - 复用 `PlayerItemOperations.vue`
  - 复用 `PlayerPalOperations.vue`
- 批量礼包与审计区
  - 复用增强后的 `PalDefenderBatchOperations.vue`

这意味着 `P6` 不再依赖旧首页抽屉里的零散入口，而是拥有独立的、可直达的工作台。

### 2.4 路由与入口整合

本阶段新增独立页面入口：

- 前端路由：`/paldefender`
- Go History 路由：`GET /paldefender`

并补回到现有入口：

- `web/src/views/OperationsOverview.vue`
- `web/src/views/PlayersOverview.vue`

## 3. 关键能力说明

### 3.1 单玩家实时发放闭环

`P5` 已经补好单玩家 `PalDefender` 发放组件，本阶段把它们正式收口到独立工作台内：

- 发物品
- 调整物品数量
- 清空背包容器
- 发帕鲁
- 发蛋
- 发模板
- 删除指定帕鲁

并继续沿用：

- 管理员登录校验
- 玩家在线校验
- 统一 `PalDefender` 错误解释

### 3.2 批量礼包与失败重试

批量发放现在已经具备更完整的运营视角：

- 支持基于玩家列表、公会成员或全在线玩家发放
- 支持叠加奖励方案与预设包
- 发放结束后立即得到聚合结果
- 支持按失败批次快速重试失败目标
- 支持失败码聚合，便于判断是离线、未配置还是参数错误

### 3.3 审计查询、导出与回溯

管理员现在可以从页面直接完成：

- 按批次号回看一次活动的执行情况
- 按玩家 UID 回看单个玩家的发放记录
- 按动作区分正常批量发放与重试批次
- 过滤成功 / 失败结果
- 导出 JSON 审计记录做留档或二次分析

## 4. 验证记录

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
pnpm --dir web exec vite build --outDir /tmp/pst-web-build-p6
```

结果：

- 构建通过
- 若仍出现历史样式或 chunk 体积告警，视为现存非阻塞告警，不影响本阶段交付

## 5. 本阶段收益

完成本阶段后，项目已经具备一套真正可运营的 `PalDefender` 工作台：

- 可以从独立页面对单玩家执行实时发放
- 可以按玩家 / 公会 / 在线人群做批量礼包投放
- 可以把发放结果沉淀为可筛选、可导出、可回溯的审计记录
- 可以对失败批次快速发起重试，不再依赖手工重建目标列表

## 6. 验收结果

本阶段对应目标已完成：

- [x] 单玩家实时发放能力已整合进独立工作台
- [x] 批量礼包支持结果聚合与失败码统计
- [x] 审计日志支持筛选、导出与回溯
- [x] 已支持基于失败批次进行重试
- [x] 已挂载 `/paldefender` 页面入口并接回现有导航
- [x] 已完成后端测试与前端构建验证

## 7. 相关文件

本阶段主要涉及：

- `service/paldefender_audit.go`
- `service/paldefender_audit_test.go`
- `api/paldefender_admin.go`
- `api/paldefender_audit_test.go`
- `api/router.go`
- `web/src/service/service.js`
- `web/src/service/api.js`
- `web/src/components/PalDefenderBatchOperations.vue`
- `web/src/views/PalDefenderOverview.vue`
- `web/src/views/OperationsOverview.vue`
- `web/src/views/PlayersOverview.vue`
- `web/src/router/index.js`
- `main.go`

## 8. 下一阶段建议

P6 完成后，建议进入 `P7`：平台成熟度与发布治理。

建议主线：

1. 统一前端请求层的错误与登录失效处理体验
2. 整理发布前验证脚本、构建产物核对与最小 CI
3. 收口文档索引、回归脚本与版本发布清单
4. 持续补充 `PalDefender` / 玩家管理链路的自动化测试
