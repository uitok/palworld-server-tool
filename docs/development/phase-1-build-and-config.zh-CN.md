# P1：工程底座与配置治理交付

> 适用时间：`2026-03-20` 当前仓库快照。本文用于记录阶段 `P1` 已完成的工程与配置治理工作，作为阶段 2 之前的交付快照与回顾依据。

## 1. 阶段目标

阶段 `P1` 的目标是解决“本地开发、文档、构建、配置三者事实来源不一致”的问题，让开发者不再依赖历史知识或缺失目录来完成构建与联调。

本阶段聚焦三件事：

1. 统一构建链路与 release 资源下载方式
2. 补齐可复用的配置样例与运行模式说明
3. 把这些变化接回开发文档与开发入口

## 2. 已完成交付

### 2.1 构建链路统一

已新增统一的 release 资源下载脚本：

- `script/download-release-asset.sh`

它负责统一下载：

- `sav_cli_linux_x86_64`
- `sav_cli_linux_aarch64`
- `sav_cli_windows_x86_64.exe`
- `map.zip`

默认下载来源：

- `https://github.com/zaigie/palworld-server-tool/releases/download/<tag>/<asset>`

版本选择规则：

1. 优先使用 `PST_RELEASE_VERSION`
2. 否则尝试读取本地 git tag
3. 最后回退到 `v0.9.9`

### 2.2 Makefile 治理

已完成以下调整：

- `Makefile` 的 `build` 不再依赖缺失的 `module/` 目录
- `Makefile` 的 `build-pub` 不再依赖 `module/dist/*`
- `build` 与 `build-pub` 改为下载 release 中的预编译 `sav_cli`
- `build-pub` 的 Windows 构建目标改为 `amd64`，与 `windows_x86_64` 命名保持一致
- 本地 `build` 默认注入 `main.version=$(git describe --tags)`

### 2.3 Docker 构建治理

已完成以下调整：

- `Dockerfile` 中 `sav_cli` 下载逻辑改为复用 `script/download-release-asset.sh`
- `Dockerfile` 中地图资源 `map.zip` 下载逻辑也改为复用同一脚本
- `Dockerfile` 支持 `assets_version` 构建参数
- `script/build-docker.sh` 会同时传递：
  - `version`
  - `assets_version`

这意味着：

- 本地 Docker 构建
- CI 中基于 tag 的 Docker 构建
- Makefile 的二进制构建

现在都共享同一套 release 资源地址规则。

### 2.4 配置样例补齐

已新增以下样例：

- `example/config.agent.yaml`
- `example/config.docker.yaml`
- `example/config.k8s.yaml`
- `example/README.zh-CN.md`

现有样例分工如下：

- `config.yaml`：通用模板
- `config.dev.yaml`：本地开发默认模板
- `config.agent.yaml`：通过 `pst-agent` 拉取存档
- `config.docker.yaml`：通过 `docker://` 读取存档
- `config.k8s.yaml`：通过 `k8s://` 读取存档

### 2.5 开发入口与文档串联

已完成以下串联：

- 根 README 新增开发者入口：`README.md`
- 开发文档索引补充统一路线图与配置样例说明：`docs/development/README.md`
- 阶段 0 文档补充配置样例入口：`docs/development/phase-0-baseline.zh-CN.md`
- `Week 1` 准备包中的启动命令统一切换到 `script/dev-start.sh`
- 原工程计划顶部增加统一路线图入口：`docs/development/development-plan.zh-CN.md`

## 3. 当前推荐入口

### 3.1 先检查环境

```bash
bash ./script/check-dev-env.sh
```

### 3.2 再查看启动矩阵

```bash
bash ./script/dev-start.sh matrix
```

### 3.3 选择配置模式

```bash
cat example/README.zh-CN.md
```

### 3.4 启动主服务

```bash
PST_CONFIG=./example/config.dev.yaml bash ./script/dev-start.sh backend
```

## 4. 本阶段验收结果

阶段 `P1` 的主要验收项如下：

- [x] 本地构建链路不再依赖缺失的 `module/` 目录
- [x] Docker 与 Makefile 共享统一的 release 资源下载规则
- [x] 已提供本地 / Agent / Docker / K8s 四类配置样例
- [x] 开发入口、文档入口、配置样例入口已串联
- [x] 新脚本与 Makefile 已完成最小语法与解析验证

已完成的验证包括：

- `bash -n script/check-dev-env.sh`
- `bash -n script/dev-start.sh`
- `bash -n script/build-docker.sh`
- `sh -n script/download-release-asset.sh`
- `make help`
- `make -n build`
- `make -n build-pub`

## 5. 已知限制

以下内容仍然存在，但不再属于阶段 `P1` 的未完成项：

1. 当前运行环境缺少 `pnpm`，因此本机直接跑前端开发命令仍会失败。
2. 当前环境未准备 `sav_cli` 本地文件；如果需要联调存档解析，需要通过新脚本下载或显式设置 `SAVE__DECODE_PATH`。
3. `P1` 仅完成工程与配置治理，不包含统一 API 错误模型、任务状态接口、配置早失败等后端治理工作，这些进入下一阶段。

## 6. 下一阶段输入

阶段 `P2` 建议从以下方向继续推进：

1. 配置校验与启动早失败
2. API 错误结构与错误码统一
3. 诊断页与任务日志增强
4. 减少全局配置与数据库耦合

## 7. 一句话总结

阶段 `P1` 已把“缺失目录、硬编码下载、配置模式不清、文档入口分散”的问题收口到可维护状态，为下一阶段的后端治理和核心功能开发打下稳定底座。
