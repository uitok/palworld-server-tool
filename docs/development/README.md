# 开发文档索引

> 本目录用于沉淀 `palworld-server-tool` 的开发基线、阶段计划、周准备包和后续设计资料。建议从本页开始阅读。

## 阅读导航

### 1. 统一开发路线图

- `unified-development-roadmap.zh-CN.md`
  - 用途：查看工程计划与功能计划合并后的统一路线图、阶段目标、版本里程碑与优先级
  - 适合：需要先看全局执行顺序与主线安排时阅读

### 2. 开发总计划

- `development-plan.zh-CN.md`
  - 用途：查看原始工程治理计划与阶段拆分
  - 适合：需要回看工程侧问题与治理细节时阅读

### 3. 阶段 0 基线

- `phase-0-baseline.zh-CN.md`
  - 用途：查看阶段 0 的正式交付物，包含运行拓扑、源码边界、启动矩阵和最小验收清单
  - 适合：准备开始开发、联调或给新成员做 onboarding 时阅读

### 4. 阶段 1 交付

- `phase-1-build-and-config.zh-CN.md`
  - 用途：查看阶段 1 已完成的构建链路统一、配置样例补齐和文档收口结果
  - 适合：需要理解当前仓库为什么不再依赖 `module/`、如何选择配置样例、如何构建时阅读

### 5. 阶段 2 交付

- `phase-2-backend-governance.zh-CN.md`
  - 用途：查看阶段 2 已完成的配置早失败、错误模型统一、任务状态治理与诊断增强结果
  - 适合：需要排障、理解新的错误结构与任务状态能力时阅读

### 6. 阶段 3 交付

- `phase-3-testing-and-sync-hardening.zh-CN.md`
  - 用途：查看阶段 3 已完成的测试保护网、同步链路清理和备份脏记录治理结果
  - 适合：需要理解当前回归覆盖面与同步/备份稳定性提升点时阅读

### 7. 阶段 4 交付

- `phase-4-core-operations.zh-CN.md`
  - 用途：查看阶段 4 已完成的服务总览、统一运维动作返回模型、`/ops` 运维页和入口整合结果
  - 适合：需要理解当前运维闭环能力、接口契约和前端入口时阅读

### 8. 阶段 5 进展

- `phase-5-player-management-and-offline-data.zh-CN.md`
  - 用途：查看阶段 5 已完成的玩家聚合、玩家管理、公会/地图检索与离线搜索工作台
  - 适合：需要理解 P5 的完整交付范围和接口/页面入口时阅读

### 9. 阶段 6 交付

- `phase-6-paldefender-operations.zh-CN.md`
  - 用途：查看阶段 6 已完成的 PalDefender 独立工作台、批量审计导出与失败重试结果
  - 适合：需要理解 P6 的运营能力闭环、页面入口与接口契约时阅读

### 10. Week 1 准备包

- `week1-dev-kit.zh-CN.md`
  - 用途：查看更细的开发准备项、风险清单和第一周建议动作
  - 适合：需要按天落地开发准备工作时阅读

## 推荐阅读顺序

1. `unified-development-roadmap.zh-CN.md`
2. `phase-0-baseline.zh-CN.md`
3. `phase-1-build-and-config.zh-CN.md`
4. `phase-2-backend-governance.zh-CN.md`
5. `phase-3-testing-and-sync-hardening.zh-CN.md`
6. `phase-4-core-operations.zh-CN.md`
7. `phase-5-player-management-and-offline-data.zh-CN.md`
8. `phase-6-paldefender-operations.zh-CN.md`
9. `week1-dev-kit.zh-CN.md`
10. `development-plan.zh-CN.md`
11. `example/config.dev.yaml`
12. `example/README.zh-CN.md`
13. `script/check-dev-env.sh`
14. `script/dev-start.sh`
15. `script/check-p3-regression.sh`

## 目录职责

- 阶段计划：放长期路线图、里程碑和阶段目标
- 基线文档：放开发入口、目录边界、联调矩阵、验收清单
- 周准备包：放短周期准备任务、风险和落地清单
- 后续设计：放接口契约、重构方案、模块设计说明

## 配置样例

- `example/README.zh-CN.md`
  - 用途：查看本地开发、Agent、Docker、K8s 四种配置样例
  - 适合：需要快速选择 `save.path` 运行模式时阅读

## 当前阶段

### 阶段进度清单

- [x] `P0` 开发基线固化
- [x] `P1` 工程底座与配置治理
- [x] `P2` 后端可维护性与诊断能力
- [x] `P3` 测试保护网与同步链路加固
- [x] `P4` 核心运维功能闭环
- [x] `P5` 玩家管理与离线数据功能闭环
- [x] `P6` PalDefender 运营功能闭环

### 已完成资料

- `unified-development-roadmap.zh-CN.md`
- `phase-0-baseline.zh-CN.md`
- `phase-1-build-and-config.zh-CN.md`
- `phase-2-backend-governance.zh-CN.md`
- `phase-3-testing-and-sync-hardening.zh-CN.md`
- `phase-4-core-operations.zh-CN.md`
- `phase-5-player-management-and-offline-data.zh-CN.md`
- `phase-6-paldefender-operations.zh-CN.md`
- `week1-dev-kit.zh-CN.md`
- `example/config.dev.yaml`
- `example/README.zh-CN.md`
- `script/check-dev-env.sh`
- `script/dev-start.sh`
- `script/download-release-asset.sh`
- `script/check-p3-regression.sh`

### P5 当前已落地

- [x] 玩家聚合总览接口与详情接口
- [x] 物品 / 帕鲁全局搜索接口
- [x] 玩家工作台内的白名单 / 踢出 / 封禁 / 解封 / 批量动作
- [x] 公会检索与地图页签
- [x] `/players` 玩家工作台

### P6 当前已落地

- [x] `PalDefender` 审计日志筛选与导出接口
- [x] 失败批次重试接口与后端测试
- [x] 批量发放面板增强：筛选 / 导出 / 重试 / 结果摘要
- [x] `/paldefender` 独立工作台
- [x] `/ops` 与 `/players` 已补回 PalDefender 入口
