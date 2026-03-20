# P2 交付说明：后端可维护性与诊断能力

> 本文档记录统一开发路线图中 `P2` 阶段的实际落地结果，目标是把后端从“能跑”推进到“更容易定位、更容易诊断、更容易继续改”。

## 1. 阶段目标

本阶段聚焦四件事：

1. **配置早失败**：把关键配置问题前置到启动阶段，而不是等运行时随机暴露。
2. **错误结构统一**：让 API 与鉴权失败返回稳定、可识别的错误模型和错误码。
3. **任务可观察**：让定时任务具备状态快照、耗时、成功/失败计数和最近错误信息。
4. **诊断入口增强**：把 `/diag`、`/diag.txt` 扩展为真正可用于排障的诊断入口。

## 2. 本阶段交付物

### 2.1 配置校验增强

已增强 `internal/config/validate.go` 与对应测试，新增或强化以下规则：

- `rcon.address`
  - 当 RCON 任一参数被配置时必须存在
  - 必须使用 `host:port` 格式
- `rcon.password`
  - 当 RCON 被配置时不能为空
- `rcon.timeout`
  - 必须大于 0
- `rest.username`
  - 当玩家同步或非白名单踢出依赖 REST 时不能为空
- `task.player_login_message`
  - 当 `task.player_logging=true` 时不能为空
- `task.player_logout_message`
  - 当 `task.player_logging=true` 时不能为空

对应测试已覆盖：

- 最小配置可通过校验
- 多项错误可聚合返回
- `PalDefender` 关键参数校验
- RCON 配置约束
- 玩家上下线广播消息与 REST 用户名约束

## 2.2 错误响应模型统一

新增共享错误响应包：

- `internal/httpx/error.go`

统一后的错误结构如下：

```json
{
  "error": "message",
  "error_code": "invalid_configuration",
  "details": [],
  "errors": 3
}
```

已完成的收口点：

- `api/response.go`
  - 统一通过 `httpx.WriteError(...)` 输出错误
  - 对配置校验错误返回 `invalid_configuration`
  - 对存档链路错误自动映射到 save 相关错误码
- `internal/auth/jwt.go`
  - 鉴权失败统一使用共享错误结构
  - 已区分：
    - `auth_token_missing`
    - `auth_token_invalid`
    - `auth_token_claims_invalid`
- `api/router.go`
  - `ErrorResponse` 类型改为复用共享定义

## 2.3 定时任务状态治理

新增：

- `internal/task/status.go`

统一暴露四类后台任务状态：

- `player_sync`
- `save_sync`
- `backup`
- `cache_cleanup`

每个任务状态包含：

- 是否启用
- 是否正在执行
- 执行间隔秒数
- 最近开始/结束/成功时间
- 最近耗时
- 成功次数 / 失败次数
- 最近错误信息 / 错误码

`internal/task/task.go` 已完成以下增强：

- 任务开始、成功、失败统一记录运行状态
- 日志输出统一包含：
  - `task=...`
  - `duration_ms=...`
  - `code=...`
- 新增 `CacheCleanupTask()`，不再直接把缓存清理逻辑散落在调度器里
- 玩家同步、存档同步、备份任务的错误日志更加结构化

## 2.4 诊断能力增强

已增强 `main.go`：

- 新增配置快照和任务快照结构
- 启动时输出关键配置摘要：
  - 配置来源
  - Web/TLS/公开 URL 状态
  - REST/RCON 配置摘要
  - Save 模式、同步/备份周期
  - `PalDefender` 与白名单踢出状态
- 诊断路由增强：
  - `/diag`
  - `/diag.txt`
  - `/diag.json`

其中：

- `/diag`
  - 作为可视化排障页
  - 会自动请求 `/api/server` 与 `/diag.json`
- `/diag.txt`
  - 适合文本化巡检、日志收集和远程排障
- `/diag.json`
  - 适合脚本、前端或外部监控系统消费

## 2.5 新增任务状态接口

已新增匿名访问接口：

- `GET /api/server/task-status`

用途：

- 快速查看后台任务当前运行状态
- 辅助前端或脚本侧展示同步/备份健康度
- 为后续监控页和系统状态面板提供数据来源

## 3. 主要收益

完成本阶段后，仓库在以下方面明显改善：

- 配置错误更早暴露，降低“启动后才发现问题”的成本
- 前后端可以依赖稳定错误码，而不是猜测字符串内容
- 后台任务运行状态可以直接观察，不必反复临时插日志
- `/diag` 从“探活页”升级为“排障入口”
- 后续做测试补强与模块收敛时，有了更清晰的运行面与接口面

## 4. 验收结果

本阶段对应目标已完成：

- [x] 配置错误可在启动早期被识别
- [x] API / 鉴权错误结构统一
- [x] 定时任务状态可被诊断与查询
- [x] `/diag`、`/diag.txt`、`/diag.json` 具备排障价值

## 5. 与后续阶段的关系

`P2` 完成后，后续建议进入 `P3`：测试保护网与同步链路加固，重点包括：

- 为高风险纯逻辑补单元测试
- 为 `save.path` 多来源适配补边界测试
- 为 `PalDefender` 预设与错误码映射补回归保护
- 加固同步、备份与来源适配链路

## 6. 相关文件

本阶段主要涉及：

- `internal/config/validate.go`
- `internal/config/validate_test.go`
- `internal/httpx/error.go`
- `internal/task/status.go`
- `internal/task/task.go`
- `internal/auth/jwt.go`
- `api/response.go`
- `api/router.go`
- `api/server.go`
- `main.go`
