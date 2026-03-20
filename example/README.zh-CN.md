# 示例配置说明

本目录提供若干开发与部署场景下的配置样例，便于阶段 `P1` 的配置治理与联调收口。

## 文件说明

- `config.yaml`
  - 通用配置模板，适合发行包与手动部署场景
- `config.dev.yaml`
  - 本地开发默认配置模板，适合 `go run . --config ./example/config.dev.yaml`
- `config.agent.yaml`
  - 适合通过 `pst-agent` 暴露 `GET /sync` 的远程存档拉取模式
- `config.docker.yaml`
  - 适合通过 `docker://container:/path` 读取存档的场景
- `config.k8s.yaml`
  - 适合通过 `k8s://namespace/pod/container:/path` 读取存档的场景

## 推荐使用方式

### 本地开发

```bash
PST_CONFIG=./example/config.dev.yaml bash ./script/dev-start.sh backend
```

### Agent 模式

```bash
PST_CONFIG=./example/config.agent.yaml bash ./script/dev-start.sh backend
```

### Docker 存档模式

```bash
PST_CONFIG=./example/config.docker.yaml bash ./script/dev-start.sh backend
```

### K8s 存档模式

```bash
PST_CONFIG=./example/config.k8s.yaml bash ./script/dev-start.sh backend
```

## 配置治理说明

这些样例的目的不是替代你的生产配置，而是：

1. 明确 `save.path` 支持的来源模式
2. 为开发与联调提供可复制的基线模板
3. 避免开发者每次都从空白 `config.yaml` 手工拼装配置
