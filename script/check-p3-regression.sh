#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

printf '== P3 最小回归验证 ==\n'
printf '仓库路径：%s\n\n' "$ROOT_DIR"

cd "$ROOT_DIR"

echo '[1/4] Go service 层测试'
go test ./service

echo '[2/4] Go source/tool 链路测试'
go test ./internal/source ./internal/tool

echo '[3/4] Go API 层测试'
go test ./api

echo '[4/4] 全量回归'
go test ./...

echo
echo 'P3 最小回归验证通过。'
