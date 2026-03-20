# palworld-server-tool 统一开发路线图

> 版本基线：`2026-03-20`  
> 目的：把当前项目的“工程/平台开发计划”和“功能开发计划”合并为一份可执行路线图，作为后续阶段开发、任务拆分和里程碑推进的统一依据。

## 1. 文档定位

本文档是 `palworld-server-tool` 的统一开发总计划，负责收口以下两类内容：

1. **工程与平台建设计划**：构建链路、配置治理、测试、文档、CI/CD、诊断、安全与发布。
2. **功能与业务能力计划**：服务器运维、玩家管理、存档解析、数据运营、`PalDefender` 增强功能。

与其他文档的关系如下：

- `README.md`：面向使用者与部署者
- `docs/development/phase-0-baseline.zh-CN.md`：面向阶段 0 的开发基线、目录边界与启动矩阵
- `docs/development/week1-dev-kit.zh-CN.md`：面向第一周准备工作与联调风险
- `docs/development/development-plan.zh-CN.md`：保留原始的工程治理计划
- **本文档**：作为后续阶段开发的总路线图与任务归并入口

## 2. 总体目标

项目整体目标不是单纯“补几个接口”，而是把仓库演进成一个：

- **可稳定开发**：新成员可快速启动、理解、联调
- **可持续维护**：目录清晰、配置明确、错误可追踪、测试可回归
- **可安全扩展**：新增功能不会持续放大历史耦合
- **可对外发布**：构建、版本、镜像、文档、验收标准统一
- **可运营管理**：支持从基础运维到玩家活动管理的完整流程

## 3. 核心原则

### 3.1 技术原则

- 优先修复根因，不做表面补丁
- 先稳定基线，再扩功能
- 先保证链路可观察，再扩动作能力
- 优先改稳定源码目录，不直接编辑构建产物
- 新增高风险逻辑时优先补最小测试保护

### 3.2 能力分层原则

- **官方 REST 优先**：服务器信息、在线玩家、广播、关服、基础管理优先走官方 REST API
- **RCON 兼容保留**：只保留兼容性与兜底角色，不再作为新增功能首选渠道
- **PalDefender 负责增强运营**：实时发放、礼包、批量运营、审计相关能力走 `PalDefender`
- **存档链路负责离线数据**：离线玩家、公会、背包、帕鲁、地图等能力通过 `save + sav_cli` 获得

### 3.3 开发边界原则

优先修改：

- `api/`
- `service/`
- `internal/`
- `web/src/`
- `pal-conf/src/`
- `cmd/`
- `example/`
- `script/`
- `docs/development/`

避免直接手改：

- `assets/`
- `index.html`
- `pal-conf.html`
- `map/`
- `dist/`

## 4. 项目能力结构

### 4.1 工程流

负责让项目“能被稳定开发与交付”：

- 开发基线
- 配置治理
- 构建统一
- 测试保护网
- 日志与诊断
- 安全与发布

### 4.2 核心运维流

负责服务器日常管理：

- 服务信息
- 指标
- 广播
- 关服
- 同步
- 备份
- 基础诊断

### 4.3 玩家与数据流

负责玩家与离线数据能力：

- 玩家列表与详情
- 白名单
- 踢出/封禁/解封
- 公会数据
- 地图数据
- 背包与帕鲁查询

### 4.4 运营增强流

负责 `PalDefender` 驱动的实时运营能力：

- 单玩家发物品
- 发帕鲁 / 发蛋 / 发模板
- 批量礼包
- 预设管理
- 审计回溯

## 5. 当前现状与问题归纳

### 5.1 已有基础

- 后端主骨架完整，`main.go + api + service + internal` 分层已形成
- 主前端与配置页均可独立开发
- 已接入官方 REST API、RCON、`PalDefender`、存档解析、备份、公会与玩家数据
- 阶段 0 文档与开发配置模板已初步落地

### 5.2 当前主要阻塞

- `Makefile` 与当前仓库快照不一致，仍依赖缺失的 `module/` 目录
- `Dockerfile` 存在资源版本硬编码，版本治理未统一
- 根目录同时包含源码与构建产物，容易误改
- `web/src/views/PcHome/PcHome.vue` 等前端文件体量较大，可维护性偏弱
- 关键外部依赖较多：REST、RCON、`PalDefender`、`sav_cli`、Docker/K8s 来源都可能成为联调阻塞点
- 测试与发布闭环尚不完整

## 6. 统一路线图

整个路线图建议按 `P0 -> P1 -> P2 -> P3 -> P4 -> P5 -> P6 -> P7` 推进。

### 阶段进度清单

- [x] `P0` 开发基线固化
- [x] `P1` 工程底座与配置治理
- [x] `P2` 后端可维护性与诊断能力
- [x] `P3` 测试保护网与同步链路加固
- [x] `P4` 核心运维功能闭环
- [x] `P5` 玩家管理与离线数据功能闭环
- [x] `P6` PalDefender 运营功能闭环
- [ ] `P7` 平台成熟度与发布治理

---

## P0：开发基线固化

### 目标

把项目整理到“可启动、可阅读、可联调、可继续开发”的状态。

### 范围

- 统一运行拓扑认知
- 固化源码与构建产物边界
- 统一主服务、主前端、配置页、`pst-agent` 的启动方式
- 提供环境检查脚本、开发配置模板和阶段 0 文档

### 当前状态

已落地基线资料：

- `docs/development/phase-0-baseline.zh-CN.md`
- `docs/development/week1-dev-kit.zh-CN.md`
- `example/config.dev.yaml`
- `script/check-dev-env.sh`
- `script/dev-start.sh`

### 验收标准

- 新开发者能在半天内跑起至少一套联调模式
- 能清楚说明目录边界、`save.path` 模式、`pal-conf` 子模块定位和前端代理关系

---

## P1：工程底座与配置治理

### 目标

解决当前最明显的工程风险，让本地开发、文档、构建、配置三者使用一致事实来源。

### 主要任务

1. **构建链路统一**
   - 修复 `Makefile` 对缺失 `module/` 的依赖
   - 统一 `sav_cli` 获取方式
   - 对齐本地构建、Docker 构建与发行构建

2. **配置治理**
   - 对齐 `example/config.yaml`、`example/config.dev.yaml` 与实际代码配置模型
   - 补齐环境变量映射说明
   - 明确 `save.path` 的本地 / HTTP Agent / Docker / K8s 四种模式示例

3. **文档收口**
   - 把基线文档、索引文档、开发总计划挂到统一入口
   - 明确开发启动顺序与最小验证步骤
   - 为本地 / Agent / Docker / K8s 提供可直接复用的配置样例

### 交付物

- 可执行的统一构建说明
- 对齐后的配置模板
- 开发文档索引与启动指南

### 验收标准

- 本地与 Docker 的构建逻辑不再明显漂移
- 开发者不再需要靠猜测来配置 `sav_cli`、`save.path` 与联调模式

---

## P2：后端可维护性与诊断能力（已完成）

### 目标

把后端从“能跑”提升到“易定位、易修改、易测试”。

### 主要任务

1. **启动早失败**
   - 强化配置校验
   - 对 `web.password`、`rest.password`、`save.path`、`save.decode_path`、`paldefender` 做显式约束

2. **错误模型统一**
   - 收口 `api/` 层错误结构与错误码
   - 统一处理 REST、存档、`PalDefender`、白名单等错误场景

3. **诊断与日志增强**
   - 启动日志输出关键配置摘要
   - 定时任务输出耗时、失败原因、来源模式
   - 把 `/diag` 的价值从“探活”提升到“排障入口”

4. **全局依赖治理**
   - 逐步减少散落的全局配置与数据库调用
   - 为后续测试与模块化重构做准备

### 交付物

- 配置校验实现
- 统一错误返回格式
- 更清晰的启动与任务日志
- 更可靠的诊断输出

### 当前状态

已完成阶段 2 交付：

- `docs/development/phase-2-backend-governance.zh-CN.md`
- `internal/httpx/error.go`
- `internal/task/status.go`
- `api/server.go` 中的任务状态接口
- `main.go` 中的增强诊断页、文本诊断与 JSON 诊断输出

### 验收标准

- 配置错误能在启动早期被明确识别
- 前端不再依赖模糊字符串判断错误类型
- 排障时无需频繁人工插日志

---

## P3：测试保护网与同步链路加固（已完成）

### 目标

为高风险逻辑建立最小回归保护，并优先加固同步、备份、来源适配等关键链路。

### 主要任务

1. **单测补齐**
   - `service/*.go` 的玩家、公会、备份、RCON 逻辑
   - `internal/source/` 的 Docker/K8s 地址解析
   - `internal/tool/paldefender_api.go` 的预设、合并、错误码映射
   - `internal/tool/save.go` 的来源选择与错误包装

2. **同步链路加固**
   - 复核本地、HTTP、Docker、K8s 四类来源
   - 统一区分“源不可达 / 路径错误 / 解码失败 / 权限问题”
   - 降低临时目录残留与脏状态概率

3. **备份链路加固**
   - 完善备份记录与文件一致性
   - 清理缺文件的脏记录
   - 明确保留策略与失败处理

### 交付物

- 第一批回归测试
- 同步/备份错误分类清单
- 最小验证脚本与操作清单

### 当前状态

已完成阶段 3 交付：

- `docs/development/phase-3-testing-and-sync-hardening.zh-CN.md`
- `script/check-p3-regression.sh`
- `internal/source/` 四类来源的失败清理增强
- `internal/tool/save.go` 中的备份脏记录清理增强
- 新增 `service/rcon_preset_test.go`、`internal/source/http_test.go`、`internal/source/local_test.go`、`internal/tool/backup_cleanup_test.go`

### 验收标准

- 高风险逻辑具备可重复回归能力
- 同步失败时能够快速定位失败阶段
- 备份记录与实际文件状态更一致

---

## P4：核心运维功能闭环（已完成）

### 目标

先把“基础运维”做完整，让管理员在不看日志的情况下也能完成日常管理。

### 主要功能

1. **服务总览**
   - 服务器版本
   - 在线人数
   - 指标摘要
   - 最近同步/备份状态
   - 当前启用能力状态（REST / RCON / `PalDefender` / Save Sync）

2. **运维动作**
   - 广播
   - 定时关服
   - 手动同步
   - 手动备份

3. **状态页与诊断页**
   - 任务运行状态
   - 最近失败摘要
   - 上游依赖可达性

### 交付物

- 服务总览接口
- 任务状态接口
- 运维动作统一返回模型
- 面板内可视状态页

### 当前状态

已完成阶段 4 交付：

- `docs/development/phase-4-core-operations.zh-CN.md`
- `api/server_ops.go` 与 `api/server_ops_test.go`
- `api/server.go` / `api/sync.go` / `api/router.go` 的运维动作统一收口
- `internal/task/task.go` 的手动执行入口复用
- `web/src/views/OperationsOverview.vue` 运维总览页
- `web/src/router/index.js` 与 `web/src/service/api.js` 的 `/ops` 路由和请求封装
- `web/src/views/PcHome/PcHome.vue` 与 `web/src/views/MobileHome/MobileHome.vue` 的入口挂载
- `main.go` 中的 `/ops` History 路由入口
- 已通过 `go test ./...` 和 `pnpm --dir web exec vite build --outDir /tmp/pst-web-build` 验证

### 验收标准

- 基础运维动作都能在 UI 内闭环完成
- 任一动作都有明确成功、失败和错误原因反馈

---

## P5：玩家管理与离线数据功能闭环（已完成）

### 目标

把“玩家管理”和“离线存档查询”做成真正可用的后台能力。

### 主要功能

1. **玩家列表与详情**
   - 合并在线 REST 数据与离线存档数据
   - 统一展示 `PlayerUID / UserID / SteamID`
   - 展示等级、位置、建筑数、最后在线、背包、帕鲁、公会归属

2. **玩家操作**
   - 白名单管理
   - 踢出
   - 封禁
   - 解封
   - 批量玩家操作

3. **离线数据运营**
   - 公会检索
   - 地图检索
   - 背包物品搜索
   - 帕鲁筛选与分析

### 交付物

- 玩家聚合模型
- 玩家详情与筛选能力
- 公会/地图/背包/帕鲁查询能力

### 当前状态

已完成阶段 5 交付：

- `docs/development/phase-5-player-management-and-offline-data.zh-CN.md`
- `service/player_overview.go` 与 `service/player_overview_test.go`
- `api/player_overview.go` 与 `api/player_overview_test.go`
- `api/player_batch.go` 与 `api/player_batch_test.go`
- `GET /api/player/overview` 与 `GET /api/player/:player_uid/overview`
- `GET /api/player/search/items` 与 `GET /api/player/search/pals`
- `POST /api/player/batch`
- `web/src/views/PlayersOverview.vue` 与 `/players` 路由
- `web/src/views/OperationsOverview.vue`、`web/src/views/PcHome/PcHome.vue`、`web/src/views/MobileHome/MobileHome.vue` 的入口整合
- `main.go` 中的 `/players` History 路由入口

当前阶段勾选：

- [x] 玩家聚合总览列表接口
- [x] 玩家聚合详情接口
- [x] 全局物品搜索
- [x] 全局帕鲁搜索
- [x] 玩家工作台内的白名单 / 踢出 / 封禁 / 解封
- [x] 批量玩家动作
- [x] 公会检索与地图查看
- [x] `/players` 玩家工作台

### 验收标准

- 管理员可以从玩家列表直接完成主要管理操作
- 能快速回答“某玩家有什么、在哪、属于谁、最近何时在线”

---

## P6：PalDefender 运营功能闭环

### 目标

把 `PalDefender` 从“可调用”提升到“可运营、可追踪、可审计”的增强能力。

### 主要功能

1. **单玩家实时发放**
   - 发物品
   - 发帕鲁
   - 发蛋
   - 发模板
   - 在线校验与前置校验

2. **批量礼包与预设**
   - 礼包预设管理
   - 批量目标选择
   - 批次执行结果聚合
   - 失败重试与失败原因统计

3. **审计能力**
   - 记录操作人、批次号、目标玩家、预设、结果、错误码
   - 支持查询、导出、回溯

### 交付物

- 单玩家发放闭环
- 批量礼包能力
- `PalDefender` 审计日志查询能力

### 当前状态

- 已完成单玩家实时发放组件与独立 `PalDefender` 工作台整合
- 已完成批量发放结果聚合、失败码统计、失败批次重试
- 已完成审计日志筛选、导出与回溯能力
- 已补齐 `/paldefender` 页面以及 `/ops`、`/players` 导航入口

### 验收标准

- 任意一次发放都能回答“谁在什么时候给了谁什么，成功还是失败”
- 错误能清楚区分“玩家离线 / 未配置 / 鉴权失败 / 服务不可达 / 参数非法”

---

## P7：前端重构、安全与发布闭环

### 目标

让项目具备长期维护和稳定发布能力。

### 主要任务

1. **前端治理**
   - 拆分超大组件
   - 统一状态流与请求层行为
   - 统一错误提示、能力状态、权限控制入口

2. **安全与权限**
   - 基于现有 JWT 扩展角色与权限边界
   - 将高风险动作纳入统一审计

3. **发布与 CI/CD**
   - 拆分后端测试、前端构建、镜像构建流程
   - 统一版本号注入与构建产物布局
   - 输出最小发布清单与回滚策略

4. **恢复与告警**
   - 增加备份恢复预演
   - 增加失败告警和故障排查手册

### 交付物

- 更易维护的前端结构
- 最小权限模型
- 发布流程文档
- 基础 CI/CD 闭环

### 验收标准

- 能稳定发布版本
- 关键高风险动作有权限与审计保护
- 出现问题时能快速判断是配置、上游服务还是代码回归

## 7. 功能优先级归纳

### 7.1 P0 必做功能

- 服务总览
- 任务状态
- 玩家列表 / 玩家详情
- 白名单
- 踢出 / 封禁 / 解封
- 手动同步 / 手动备份

### 7.2 P1 高价值功能

- 单玩家发物品 / 发帕鲁 / 发蛋 / 发模板
- 批量礼包与预设
- 审计日志与失败追踪
- 公会 / 地图 / 背包 / 帕鲁查询
- 配置诊断页

### 7.3 P2 后续增强

- 备份恢复预演
- 角色权限
- 通知告警
- 数据报表
- 活动自动化

## 8. 工作流拆分建议

### 8.1 A：工程流

负责：

- 构建统一
- 配置规范
- 文档基线
- 测试保护网
- CI 与发布基础

### 8.2 B：后端流

负责：

- REST 封装
- 存档与同步链路
- 玩家 / 公会 / 白名单 / 备份服务
- `PalDefender` 后端能力与审计

### 8.3 C：前端流

负责：

- 服务总览页
- 玩家与公会页
- 批量发放页
- 备份与诊断页
- 配置联动体验

### 8.4 D：运维流

负责：

- 部署模式验证
- 诊断与排障
- 发布流程
- 告警与回滚策略

## 9. 版本与里程碑建议

### v0.1 工程稳定版

- P0 + P1 完成
- 文档齐备
- 构建与配置认知收口

### v0.2 运维闭环版

- P2 + P4 完成
- 服务状态、同步、备份、广播、关服、诊断稳定

### v0.3 玩家管理版

- P5 完成第一轮
- 玩家详情、公会、地图、背包、帕鲁查询可用

### v0.4 PalDefender 运营版

- P6 完成第一轮
- 单发、批量发放、预设、审计具备闭环

### v0.5 平台成熟版

- P7 完成第一轮
- 权限、发布、恢复、告警具备最小闭环

## 10. 当前最值得立即推进的 12 项任务

1. 修复 `Makefile` 与 `sav_cli` 构建漂移
2. 对齐配置模板与实际代码配置模型
3. 输出 REST / RCON / `PalDefender` 能力矩阵
4. 增加服务总览接口
5. 增加任务状态接口
6. 统一 API 错误结构与错误码
7. 整理玩家标识映射规则
8. 完善玩家列表与详情页
9. 完善白名单与批量玩家操作
10. 做单玩家物品 / 帕鲁发放闭环
11. 做批量礼包预设与审计能力
12. 做配置诊断页与依赖可达性状态

## 11. 文档与执行建议

后续执行建议按以下顺序使用文档：

1. `docs/development/README.md`
2. `docs/development/unified-development-roadmap.zh-CN.md`
3. `docs/development/phase-0-baseline.zh-CN.md`
4. `docs/development/phase-1-build-and-config.zh-CN.md`
5. `docs/development/phase-2-backend-governance.zh-CN.md`
6. `docs/development/phase-3-testing-and-sync-hardening.zh-CN.md`
7. `docs/development/phase-4-core-operations.zh-CN.md`
8. `docs/development/week1-dev-kit.zh-CN.md`
9. `docs/development/development-plan.zh-CN.md`

如果后续需要继续推进，可以基于本文档继续拆出：

- 按周执行版
- 按模块 TODO 清单版
- GitHub Issue 任务版
- 版本发布路线图版

## 12. 一句话总结

先稳基线，再统一构建与配置；先做核心运维与玩家管理，再做存档数据与 `PalDefender` 运营增强，最后补齐前端治理、安全、恢复和发布闭环。
