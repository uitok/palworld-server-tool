# P3 交付说明：测试保护网与同步链路加固

> 本文档记录统一开发路线图中 `P3` 阶段的实际交付，重点是把“高风险逻辑可回归”和“同步/备份链路更稳定”两件事真正落到代码里。

## 1. 阶段目标

本阶段重点完成三类工作：

1. **补测试保护网**：为 `service`、`internal/source`、`internal/tool`、`api` 中高风险逻辑补最小回归测试。
2. **加固同步链路**：减少本地 / HTTP / Docker / K8s 来源失败时的临时目录残留。
3. **加固备份链路**：在定时清理时主动移除“文件已丢失但记录还在”的脏备份记录。

## 2. 本阶段交付物

### 2.1 测试保护网扩充

当前已覆盖的回归面包括：

- `service/`
  - `service/player_test.go`
  - `service/guild_test.go`
  - `service/backup_test.go`
  - `service/rcon_test.go`
  - `service/rcon_preset_test.go`
- `internal/source/`
  - `internal/source/docker_test.go`
  - `internal/source/pod_test.go`
  - `internal/source/http_test.go`
  - `internal/source/local_test.go`
- `internal/tool/`
  - `internal/tool/save_test.go`
  - `internal/tool/backup_cleanup_test.go`
  - `internal/tool/paldefender_api_test.go`
- `api/`
  - `api/backup_test.go`
  - `api/player_admin_test.go`
  - `api/paldefender_admin_test.go`

重点覆盖内容：

- 玩家、公会、备份、RCON 的基础 CRUD 和预设导入逻辑
- Docker / K8s 地址解析
- HTTP 来源下载与 404 / 解压失败处理
- 本地来源失败时的清理行为
- Save 来源选择、错误码映射和错误详情输出
- `PalDefender` 预设加载、去重、合并和错误码映射
- 备份 API 对缺文件脏记录的清理行为

### 2.2 同步链路加固

已增强四类来源的失败清理行为：

- `internal/source/local.go`
- `internal/source/http.go`
- `internal/source/docker.go`
- `internal/source/pod.go`

本次改动的核心点：

- 当临时目录已经创建，但随后发生复制 / 解压 / 解包失败时，会自动清理本次创建的临时目录
- 降低 `palworldsav-*` 临时目录在异常路径下残留的概率
- 保持成功路径不受影响，返回的 `Level.sav` 临时目录仍由上层使用完后统一清理

### 2.3 备份链路加固

已增强 `internal/tool/save.go` 中的 `CleanOldBackups(...)`：

- 清理旧备份时，不再只处理“超过保留期”的记录
- 对所有备份记录都会先做一次实体校验：
  - 文件不存在 → 直接删除脏记录
  - 路径指向目录 → 直接删除无效记录
- 对仍然存在且超出保留期的备份文件，继续执行“删文件 + 删记录”逻辑

这意味着：

- “文件已经被手工删掉，但数据库里还留着记录”的情况，会在定时清理阶段自动收敛
- “记录路径错误地指向目录”的异常记录，也不会长期残留

## 3. 错误分类清单

当前存档链路的主要错误分类已固定为：

- `save_source_invalid`
  - 地址格式错误
  - 来源内容不是有效 `Level.sav`
- `save_source_not_found`
  - 文件不存在
  - 远端目录中找不到 `Level.sav`
- `save_source_unreachable`
  - HTTP / Docker / K8s 来源不可达
  - 网络超时、连接拒绝、集群配置不可读
- `save_source_copy_failed`
  - 复制 / 解包 / 中间步骤失败
- `save_decode_cli_missing`
  - `sav_cli` 缺失
- `save_decode_prepare_failed`
  - 解码前准备失败
- `save_decode_failed`
  - `sav_cli` 启动 / 等待 / 执行失败

错误详情可继续通过：

- `tool.SaveOperationErrorCode(err)`
- `tool.SaveOperationErrorDetails(err)`

进行稳定消费。

## 4. 最小回归脚本

新增脚本：

- `script/check-p3-regression.sh`

用途：

- 先跑 `service` 层测试
- 再跑 `internal/source` 与 `internal/tool` 链路测试
- 再跑 `api` 层测试
- 最后跑 `go test ./...` 做全量确认

执行方式：

```bash
bash ./script/check-p3-regression.sh
```

## 5. 本阶段收益

完成本阶段后，项目在以下方面更稳定：

- 高风险纯逻辑已具备最小回归保护
- 来源适配失败时更不容易遗留脏临时目录
- 备份记录与实际文件状态更加一致
- 排查同步失败时可以更快分辨“地址问题 / 不可达 / 内容无效 / 解码失败”

## 6. 验收结果

本阶段对应目标已完成：

- [x] 第一批高风险逻辑单测落地
- [x] 同步来源失败时的临时目录残留概率下降
- [x] 备份脏记录可在清理阶段自动收敛
- [x] 已提供最小回归脚本和操作清单

## 7. 相关文件

本阶段主要涉及：

- `internal/source/local.go`
- `internal/source/http.go`
- `internal/source/docker.go`
- `internal/source/pod.go`
- `internal/source/http_test.go`
- `internal/source/local_test.go`
- `internal/tool/save.go`
- `internal/tool/save_error.go`
- `internal/tool/save_test.go`
- `internal/tool/backup_cleanup_test.go`
- `internal/tool/paldefender_api_test.go`
- `service/rcon_preset_test.go`
- `script/check-p3-regression.sh`
