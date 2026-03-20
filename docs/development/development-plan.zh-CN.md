# palworld-server-tool 开发计划（重构版）

> 本计划基于当前仓库结构、已有文档、现有构建链路和实际开发阻塞点重新梳理，目标是把项目推进到“可稳定开发、可持续维护、可安全扩展功能”的状态。

> 说明：若需要查看已经合并工程计划与功能计划的统一版本，请优先阅读 `docs/development/unified-development-roadmap.zh-CN.md`。
> 若需要查看阶段 `P1` 已完成的构建与配置治理结果，请阅读 `docs/development/phase-1-build-and-config.zh-CN.md`。
> 若需要查看当前已完成的统一路线图 `P4` 运维闭环交付，请阅读 `docs/development/phase-4-core-operations.zh-CN.md`。
> 若需要查看已完成的 `P5` 玩家管理与离线数据闭环交付，请阅读 `docs/development/phase-5-player-management-and-offline-data.zh-CN.md`。
> 若需要查看已完成的 `P6` PalDefender 运营能力闭环交付，请阅读 `docs/development/phase-6-paldefender-operations.zh-CN.md`。

## 1. 计划目标

本轮开发计划聚焦 4 个核心目标：

1. **先稳定开发基线**：让新开发者可以快速跑起项目，明确源码边界、配置方式、依赖关系和联调路径。
2. **再提升可维护性**：降低全局配置、超大组件、构建脚本和外部依赖带来的修改风险。
3. **再强化关键链路**：重点加固玩家同步、存档解析、备份、RCON、PalDefender 等核心能力。
4. **最后收束发布流程**：统一版本来源、构建步骤、验证清单和最小 CI。

## 2. 当前现状判断

### 2.1 已有基础

- 主后端已具备完整服务骨架：`main.go`、`api/`、`service/`、`internal/` 分层清晰。
- 主前端和配置页都已经存在并可独立开发：
  - `web/`：Vue 3 + Vite
  - `pal-conf/`：React + Vite
- 项目已有较强的业务覆盖：REST 管理、RCON、自定义命令、备份、玩家数据、公会数据、PalDefender 批量操作。
- 当前仓库已补充开发准备基线文档：`docs/development/week1-dev-kit.zh-CN.md`。

### 2.2 主要问题

- **开发基线不够收口**：完整构建链路依赖较多，且 `module/` 目录在当前快照中缺失。
- **构建与发布存在硬编码**：`Dockerfile` 内固定下载旧版本资源，`Makefile` 仍有历史构建耦合。
- **后端全局依赖偏多**：`viper` 与 `database.GetDB()` 在多处直接调用，可测试性一般。
- **前端维护成本偏高**：`web/src/views/PcHome/PcHome.vue` 体量过大，状态和请求逻辑耦合较重。
- **测试覆盖不足**：当前仓库几乎没有针对业务代码的有效测试保护网。
- **外部依赖较多**：REST、RCON、PalDefender、`sav_cli`、Docker/K8s 存档来源都可能成为联调阻塞点。

## 3. 开发原则

### 3.1 原则

- 优先修复根因，不做表面补丁。
- 优先让代码“更容易改”，再去堆新功能。
- 保持改动小步快跑，每个阶段都有明确交付物。
- 尽量减少对构建产物目录的直接编辑。
- 先补最小测试保护，再做大范围重构。

### 3.2 源码边界

优先修改：

- `api/`
- `service/`
- `internal/`
- `web/src/`
- `pal-conf/src/`
- `example/`
- `script/`
- `docs/development/`

避免直接手改：

- `assets/`
- `index.html`
- `pal-conf.html`
- `map/`

## 4. 里程碑拆分

整个计划按 6 个阶段推进，推荐顺序为 `P0 -> P1 -> P2 -> P3 -> P4 -> P5`。

---

## P0：开发基线固化

### 目标

把项目变成“可启动、可阅读、可联调”的状态。

### 主要任务

- 统一本地开发入口和依赖说明。
- 固化配置模板、联调模式、环境检查脚本。
- 明确源码目录与构建产物目录边界。
- 梳理主服务、主前端、配置页、`pst-agent` 的启动方式。

### 当前状态

- 已完成基础落地：
  - `docs/development/phase-0-baseline.zh-CN.md`
  - `docs/development/week1-dev-kit.zh-CN.md`
  - `docs/development/README.md`
  - `example/config.dev.yaml`
  - `script/check-dev-env.sh`

### 验收标准

- 新开发者能在半天内完成环境检查并跑起至少一套开发链路。
- 对 `save.path`、`sav_cli`、`pal-conf` 子模块、主前端代理关系有清晰认知。

### 推荐阅读顺序

阶段 0 相关资料建议按以下顺序阅读：

1. `docs/development/README.md`：开发文档索引与阅读导航
2. `docs/development/phase-0-baseline.zh-CN.md`：阶段 0 正式基线、目录边界、启动矩阵、验收清单
3. `docs/development/week1-dev-kit.zh-CN.md`：更细的周内准备项与风险说明
4. `example/config.dev.yaml`：开发联调配置模板
5. `script/check-dev-env.sh`：环境检查与快速自检脚本

---

## P1：后端可维护性治理

### 目标

把后端从“能跑”提升到“易定位、易修改、易测试”。

### 主要任务

1. **配置校验与启动早失败**
   - 在 `internal/config/config.go` 附近补配置合法性校验。
   - 对 `web.password`、`rest.password`、`save.path`、`save.decode_path`、`paldefender` 进行显式校验。

2. **统一错误返回模型**
   - 收口 `api/` 层错误结构。
   - 优先覆盖：
     - `api/server.go`
     - `api/rcon.go`
     - `api/player_admin.go`
     - `api/paldefender_admin.go`
     - `api/sync.go`

3. **增强日志与诊断能力**
   - 启动日志输出关键配置摘要。
   - 定时任务增加执行耗时和失败原因。
   - 保持 `diag` 页可用，同时增强排障价值。

4. **减少全局调用扩散**
   - 逐步收敛 `database.GetDB()` 的直接散落调用。
   - 减少 `api -> internal/tool -> viper` 的隐式耦合。

### 交付物

- 配置校验实现
- 统一 API 错误格式
- 更清晰的启动/任务日志

### 验收标准

- 配置错误能在服务启动早期被清晰报出。
- 核心 API 错误结构统一，便于前端处理。
- 排障不再严重依赖人工插日志。

---

## P2：测试保护网建立

### 目标

先为纯逻辑和关键路径建立最小测试保护，降低后续重构成本。

### 主要任务

1. 为 `service/*.go` 增加基础单测：
   - 玩家 CRUD
   - 公会读取
   - 备份记录
   - RCON 预设与命令存储

2. 为 `internal/source/` 增加地址解析测试：
   - `docker://`
   - `k8s://`
   - 本地/HTTP 来源解析边界

3. 为 `internal/tool/paldefender_api.go` 增加测试：
   - preset 加载
   - preset 去重
   - grant 合并和规范化
   - 错误码映射

4. 为 `internal/tool/save.go` 中来源选择逻辑增加测试。

### 交付物

- 第一批业务单测
- 最小回归验证清单

### 验收标准

- 后续每次改动关键逻辑前，都能快速跑一轮最小验证。
- 配置、来源解析、PalDefender preset 等高风险逻辑有回归保护。

---

## P3：关键业务链路加固

### 目标

加固项目最有价值、最容易出问题的核心链路。

### 主要任务

1. **存档同步链路加固**
   - 复核 `save.path` 的 4 类来源：本地、HTTP、Docker、K8s。
   - 统一错误提示，区分“找不到文件”“外部连接失败”“解码失败”。
   - 明确临时目录清理和失败后的资源释放行为。

2. **备份链路加固**
   - 提升备份创建、清理、下载的日志质量。
   - 明确备份记录与文件实体的一致性处理。

3. **玩家实时操作链路加固**
   - 收敛在线校验逻辑。
   - 明确 PlayerUID / UserID / SteamID 的解析优先级。
   - 改善玩家发物品、发帕鲁、发模板的失败提示。

4. **PalDefender 闭环治理**
   - 规范 preset 使用方式。
   - 增强批量发放结果可视化与审计完整性。
   - 明确在线限制与错误码对照关系。

### 交付物

- 更稳定的存档同步链路
- 更可信的备份链路
- 更清晰的玩家实时操作反馈
- PalDefender 的完整审计闭环

### 验收标准

- 常见联调失败能在日志与 API 返回中被清楚识别。
- 同步、备份、批量操作都可追踪、可解释、可复查。

---

## P4：前端可维护性重构

### 目标

降低主前端修改成本，为后续功能开发创造空间。

### 主要任务

1. **拆分超大页面组件**
   - 优先拆 `web/src/views/PcHome/PcHome.vue`
   - 建议拆成：
     - 页面容器
     - 服务器状态区
     - 玩家列表区
     - 玩家详情区
     - 管理动作区
     - PalDefender 批量操作区

2. **统一请求层**
   - 收敛 `web/src/service/api.js` 和 `web/src/service/service.js`
   - 统一处理：
     - 登录失效
     - 统一错误提示
     - 请求参数序列化
     - 空数据与异常数据适配

3. **确认状态持久化机制**
   - 当前 store 使用 `persist: true`
   - 需要明确是否真正启用了 Pinia 持久化插件
   - 若未启用，明确修复方案

4. **减少 PC / Mobile 重复逻辑**
   - 抽离公共业务逻辑
   - 降低双端同步维护成本

### 交付物

- 拆分后的前端结构
- 更统一的请求层
- 明确的状态持久化方案

### 验收标准

- 后续新增一个管理能力时，不再需要把大量逻辑塞进单一大文件。
- 前端错误处理方式更稳定，接口变更影响面更小。

---

## P5：构建、发布与版本治理

### 目标

让项目具备更可重复的构建与发布能力。

### 主要任务

1. **统一版本来源**
   - 收敛 `Makefile`、`Dockerfile`、发布脚本的版本变量。
   - 移除硬编码旧版本资源的方式。

2. **校正构建脚本问题**
   - 复核 `Makefile` 中的历史路径依赖。
   - 明确当前快照缺少 `module/` 时的替代方案。
   - 复核 `pip install requests tdqm` 这类可疑拼写问题。

3. **梳理资源产物搬运流程**
   - 明确 `web`、`pal-conf`、地图资源、`sav_cli` 最终如何进入运行时镜像与 Go embed。

4. **补最小 CI**
   - Go 构建
   - 前端构建
   - 第一批业务测试

### 交付物

- 更清晰的发布流程
- 更可靠的构建脚本
- 最小 CI 流水线

### 验收标准

- 从干净环境出发，能按文档重复构建。
- 发布前具备固定检查清单。

---

## 5. 推荐执行顺序

### 第一阶段：立即开始

1. `P1` 配置校验与错误返回统一
2. `P2` 建立第一批测试
3. `P3` 优先加固存档同步与备份链路

### 第二阶段：稳定后推进

4. `P4` 主前端拆分与请求层整理
5. `P3` 补完玩家实时操作与 PalDefender 闭环

### 第三阶段：准备对外发布前完成

6. `P5` 构建、版本与发布治理

## 6. 建议的近期任务清单

以下任务建议作为下一轮实际开发的起点：

### 任务 A：配置校验

- 新增配置校验入口
- 在服务启动前完成校验
- 首批覆盖 `web`、`rest`、`save`、`paldefender`

### 任务 B：统一错误结构

- 先在 `api/` 层统一为相同返回格式
- 给前端预留稳定字段，例如：`error`、`error_code`、`details`

### 任务 C：第一批测试

- `service/backup.go`
- `service/rcon.go`
- `internal/source/docker.go`
- `internal/source/pod.go`
- `internal/tool/paldefender_api.go`

### 任务 D：前端状态持久化确认

- 确认 `persist: true` 的依赖是否真的生效
- 若未生效，给出修复方案并补文档

## 7. 风险与依赖

### 7.1 高风险项

- `module/` 缺失导致完整构建链路不闭合
- `sav_cli` 缺失导致存档解析能力无法实测
- 外部 REST / RCON / PalDefender 服务未就绪时，联调容易误判为代码问题
- 根目录保留构建产物，容易误修改

### 7.2 依赖项

- Go、Node、`pnpm`、Python 基础环境
- `pal-conf` 子模块可用
- 至少一种真实或模拟的存档来源
- 至少一种可连接的 Palworld 服务或替代联调环境

## 8. 阶段性验收口径

### 通过标准

如果满足以下条件，可认为本轮开发计划执行有效：

- 新开发者能按照文档快速进入开发
- 配置错误和外部依赖错误能快速区分
- 关键路径具备最小测试保护
- 前端主要页面不再继续恶化为更大的单体组件
- 构建与发布路径比当前更加明确

## 9. 当前建议的下一步

建议马上开始 `P1` 的第一项：

**在 `internal/config/` 附近补配置校验与启动早失败机制。**

原因：

- 这是整个项目最小、收益最高、最能减少本地调试噪音的改动。
- 它会直接提升后续 `P2`、`P3`、`P4` 的开发效率。

## 10. 当前功能开发计划

本章节聚焦“用户可以直接感知到的功能迭代”，与前文的工程治理、测试、构建治理形成配套关系。

### 10.1 功能开发目标

当前功能开发优先围绕以下 5 条主线推进：

1. 服务状态可见化
2. 配置体验优化
3. 备份能力增强
4. 玩家管理闭环
5. PalDefender 运营能力完善

目标不是继续堆叠零散页面，而是先把现有高频功能做完整、做稳定、做可解释。

### 10.2 第一批功能优先级

#### A. 服务状态可见化

目标：优先解决“系统当前是否可用、哪里不可用”的问题。

主要内容：

- 增加服务总览状态：REST、RCON、`sav_cli`、PalDefender、存档来源可达性
- 增加最近任务执行状态：玩家同步、存档同步、自动备份最近一次执行时间与结果
- 明确错误来源：配置错误、连接错误、权限错误、外部服务不可用

建议优先落点：

- `api/server.go`
- `internal/task/task.go`
- `internal/tool/rest_api.go`
- `internal/tool/rcon.go`
- `internal/tool/paldefender_api.go`

#### B. 配置体验优化

目标：降低部署与联调门槛，减少“服务启动了但其实不可用”的情况。

主要内容：

- 为关键配置项补合法性校验
- 区分本地模式、Agent 模式、Docker 模式、K8s 模式
- 在 UI 或接口中补充清晰的配置错误反馈

建议优先落点：

- `internal/config/config.go`
- `main.go`
- `docs/development/week1-dev-kit.zh-CN.md`

#### C. 备份功能增强

目标：把备份做成可信能力，而不是“能点一次”的工具功能。

主要内容：

- 完善备份列表展示与时间信息
- 增加备份文件缺失与数据库记录不一致时的处理
- 优化备份下载、删除、清理策略反馈

建议优先落点：

- `api/backup.go`
- `service/backup.go`
- `internal/task/task.go`
- `internal/tool/save.go`

### 10.3 第二批功能优先级

#### D. 玩家管理闭环

目标：把玩家管理从“能操作”提升到“可筛选、可排错、可批量处理”。

主要内容：

- 增强玩家列表筛选：在线状态、最后在线时间、昵称、SteamID、等级
- 整合玩家详情：背包、帕鲁、在线信息、公会归属
- 强化实时操作：发物品、调背包、发帕鲁、发蛋、模板发放、清背包、删帕鲁
- 改善操作失败提示和在线校验反馈

建议优先落点：

- `api/player.go`
- `api/player_admin.go`
- `service/player.go`
- `web/src/views/PcHome/PcHome.vue`
- `web/src/components/PlayerItemOperations.vue`
- `web/src/components/PlayerPalOperations.vue`

#### E. 公会管理增强

目标：让公会成为独立管理入口，而不只是玩家的附属信息。

主要内容：

- 完善公会列表与详情展示
- 增加公会成员、据点、管理员的联动跳转
- 为后续筛选与风险分析打基础

建议优先落点：

- `api/guild.go`
- `service/guild.go`
- `web/src/views/PcHome/component/GuildList.vue`
- `web/src/views/MobileHome/component/GuildList.vue`

#### F. RCON 功能增强

目标：让 RCON 成为高频可用的运维工具。

主要内容：

- 优化预设命令管理、备注、导入导出体验
- 提升执行结果反馈与失败原因提示
- 对原始命令执行增加风险提示

建议优先落点：

- `api/rcon.go`
- `service/rcon.go`
- `service/rcon_preset.go`
- `web/src/service/api.js`

### 10.4 第三批重点扩展

#### G. PalDefender 管理面板完善

目标：把 PalDefender 从高级功能提升为可放心使用的运营面板能力。

主要内容：

- 完善 preset 展示、选择、组合与校验
- 强化批量发放结果展示：成功数、失败数、失败原因、批次号
- 完善审计日志：按玩家、类型、时间过滤

建议优先落点：

- `api/paldefender_admin.go`
- `internal/tool/paldefender_api.go`
- `service/paldefender_audit.go`
- `web/src/components/PalDefenderBatchOperations.vue`

#### H. 存档同步能力增强

目标：让存档解析链路稳定可控、故障可定位。

主要内容：

- 加强本地、HTTP、Docker、K8s 多来源支持
- 明确同步失败原因：路径错误、连接失败、压缩失败、解码失败
- 增加手动同步与自动同步状态反馈

建议优先落点：

- `internal/tool/save.go`
- `internal/source/local.go`
- `internal/source/http.go`
- `internal/source/docker.go`
- `internal/source/pod.go`
- `api/sync.go`

#### I. 地图与可视化联动增强

目标：提升地图在管理和诊断中的实际价值。

主要内容：

- 强化地图与玩家、公会、据点信息联动
- 为后续筛选、定位、区域标记能力打基础

建议优先落点：

- `web/src/views/PcHome/component/MapView.vue`
- `web/src/views/PcHome/PcHome.vue`

### 10.5 后续版本方向

#### J. 多服务器支持

适合作为单独立项，不建议与当前治理工作并行混做。

方向包括：

- 多实例统一管理
- 多配置源、多存档源、多状态看板
- 服务切换与隔离

#### K. 权限与操作审计

方向包括：

- 不再仅依赖单一管理员口令
- 增加更细粒度的操作权限控制
- 强化操作日志与可追溯性

#### L. 配置页集成增强

方向包括：

- 强化 `pal-conf` 与主面板之间的衔接
- 优化生成、导入、校验配置的操作链路

### 10.6 推荐执行顺序

第一批先做：

1. 服务状态可见化
2. 配置体验优化
3. 备份功能增强

第二批再做：

4. 玩家管理闭环
5. 公会管理增强
6. RCON 功能增强

第三批重点做：

7. PalDefender 管理面板完善
8. 存档同步能力增强
9. 地图与可视化联动增强

第四批再考虑：

10. 多服务器支持
11. 权限体系
12. 深度审计与配置页集成增强

### 10.7 一句话结论

当前功能开发重点不是继续扩散页面和入口，而是先把 **状态可见化、配置体验、备份、玩家管理、PalDefender** 这 5 条主线做完整。

