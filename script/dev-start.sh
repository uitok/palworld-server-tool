#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

usage() {
  cat <<'USAGE'
Usage:
  bash ./script/dev-start.sh check
  bash ./script/dev-start.sh matrix
  bash ./script/dev-start.sh backend
  bash ./script/dev-start.sh web
  bash ./script/dev-start.sh pal-conf
  PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent

Commands:
  check      Run development environment checks
  matrix     Print recommended local development startup matrix
  backend    Start the main backend with example/config.dev.yaml by default
  web        Start the Vue web frontend
  pal-conf   Start the React config editor frontend
  agent      Start pst-agent (requires PST_AGENT_DIR or a positional path)

Environment variables:
  PST_CONFIG       Backend config path, default: ./example/config.dev.yaml
  PST_AGENT_DIR    Agent save directory path
  PST_AGENT_PORT   Agent port, default: 8081
USAGE
}

ensure_file() {
  local path="$1"
  local tip="$2"
  if [ ! -e "$path" ]; then
    printf '[ERR] %s\n' "$tip"
    exit 1
  fi
}

print_matrix() {
  cat <<MATRIX
== palworld-server-tool 开发启动矩阵 ==
仓库路径：$ROOT_DIR

1. 环境检查
   bash ./script/check-dev-env.sh

2. 查看推荐模式
   bash ./script/dev-start.sh matrix

3. 启动主服务
   bash ./script/dev-start.sh backend

4. 启动主前端
   bash ./script/dev-start.sh web

5. 启动配置页
   bash ./script/dev-start.sh pal-conf

6. 启动 Agent
   PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent

推荐联调模式：
- 模式 A：仅 REST
- 模式 B：REST + 本地存档解析
- 模式 C：REST + Agent 拉取存档

如果 pal-conf 子模块缺失，请先执行：
  git submodule update --init --recursive
MATRIX
}

command="${1:-help}"
case "$command" in
  help|-h|--help)
    usage
    ;;
  check)
    cd "$ROOT_DIR"
    exec bash ./script/check-dev-env.sh
    ;;
  matrix)
    print_matrix
    ;;
  backend)
    ensure_file "$ROOT_DIR/example/config.dev.yaml" "缺少 example/config.dev.yaml，请先检查仓库基线文件"
    cd "$ROOT_DIR"
    exec go run . --config "${PST_CONFIG:-./example/config.dev.yaml}"
    ;;
  web)
    ensure_file "$ROOT_DIR/web/package.json" "缺少 web/package.json，请确认前端目录完整"
    cd "$ROOT_DIR/web"
    exec pnpm dev
    ;;
  pal-conf)
    ensure_file "$ROOT_DIR/pal-conf/package.json" "缺少 pal-conf/package.json，请先执行 git submodule update --init --recursive"
    cd "$ROOT_DIR/pal-conf"
    exec pnpm dev
    ;;
  agent)
    agent_dir="${PST_AGENT_DIR:-${2:-}}"
    if [ -z "$agent_dir" ]; then
      printf '[ERR] 启动 agent 需要设置 PST_AGENT_DIR，或以第二个参数传入存档目录路径\n'
      printf '示例：PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent\n'
      exit 1
    fi
    cd "$ROOT_DIR"
    exec go run ./cmd/pst-agent --port "${PST_AGENT_PORT:-8081}" -d "$agent_dir"
    ;;
  *)
    usage
    exit 1
    ;;
esac
