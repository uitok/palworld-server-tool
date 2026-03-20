#!/bin/bash
set -u

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WARNINGS=0
ERRORS=0

ok() {
  printf '[OK] %s\n' "$1"
}

warn() {
  printf '[WARN] %s\n' "$1"
  WARNINGS=$((WARNINGS + 1))
}

err() {
  printf '[ERR] %s\n' "$1"
  ERRORS=$((ERRORS + 1))
}

check_cmd() {
  local name="$1"
  if command -v "$name" >/dev/null 2>&1; then
    ok "命令存在：$name ($(command -v "$name"))"
  else
    err "缺少命令：$name"
  fi
}

check_file() {
  local path="$1"
  if [ -e "$ROOT_DIR/$path" ]; then
    ok "存在：$path"
  else
    err "缺失：$path"
  fi
}

check_dir() {
  local path="$1"
  if [ -d "$ROOT_DIR/$path" ]; then
    ok "目录存在：$path"
  else
    err "目录缺失：$path"
  fi
}

printf '== palworld-server-tool 开发环境检查 ==\n'
printf '仓库路径：%s\n\n' "$ROOT_DIR"

check_cmd git
check_cmd go
check_cmd node
check_cmd pnpm
check_cmd python3

printf '\n== 关键入口检查 ==\n'
check_file main.go
check_file cmd/pst-agent/main.go
check_file web/package.json
check_file pal-conf/package.json
check_file example/config.yaml
check_file example/config.dev.yaml
check_file example/config.agent.yaml
check_file example/config.docker.yaml
check_file example/config.k8s.yaml
check_file example/README.zh-CN.md
check_file script/dev-start.sh
check_file script/download-release-asset.sh

printf '\n== 子模块状态 ==\n'
if git -C "$ROOT_DIR" submodule status -- pal-conf >/dev/null 2>&1; then
  SUBMODULE_STATUS="$(git -C "$ROOT_DIR" submodule status -- pal-conf)"
  ok "pal-conf 子模块已登记：$SUBMODULE_STATUS"
else
  warn "无法读取 pal-conf 子模块状态"
fi

printf '\n== 构建产物边界 ==\n'
check_dir assets
check_file index.html
check_file pal-conf.html
check_dir map

printf '\n== 存档解析依赖 ==\n'
if [ -n "${SAVE__DECODE_PATH:-}" ]; then
  if [ -e "${SAVE__DECODE_PATH}" ]; then
    ok "检测到 SAVE__DECODE_PATH：${SAVE__DECODE_PATH}"
  else
    warn "SAVE__DECODE_PATH 已设置，但文件不存在：${SAVE__DECODE_PATH}"
  fi
elif [ -e "$ROOT_DIR/sav_cli" ] || [ -e "$ROOT_DIR/sav_cli.exe" ]; then
  ok "检测到仓库根目录 sav_cli 可执行文件"
else
  warn "未检测到 sav_cli；如果要联调存档解析，请准备 sav_cli 或设置 SAVE__DECODE_PATH"
fi

printf '\n== 构建链路状态 ==\n'
if [ -d "$ROOT_DIR/module" ]; then
  ok "存在 module/ 目录（当前构建链路已不再强依赖它）"
else
  ok "当前仓库快照缺少 module/ 目录；P1 已切换为 release 预编译 sav_cli 下载链路"
fi

printf '\n== 推荐下一步 ==\n'
printf '1. 查看启动矩阵：bash ./script/dev-start.sh matrix\n'
printf '2. 主服务：bash ./script/dev-start.sh backend\n'
printf '3. 主前端：bash ./script/dev-start.sh web\n'
printf '4. 配置页：bash ./script/dev-start.sh pal-conf\n'
printf '5. Agent：PST_AGENT_DIR=/path/to/Pal/Saved bash ./script/dev-start.sh agent\n'
printf '6. 查看配置样例：cat example/README.zh-CN.md\n'

printf '\n检查完成：%d 个错误，%d 个警告\n' "$ERRORS" "$WARNINGS"
if [ "$ERRORS" -gt 0 ]; then
  exit 1
fi
