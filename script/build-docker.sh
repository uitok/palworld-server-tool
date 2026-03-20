#!/bin/bash
set -euo pipefail

VERSION="$(git describe --tags 2>/dev/null || printf 'dev')"
ASSETS_VERSION="$(git describe --tags --abbrev=0 2>/dev/null || printf 'v0.9.9')"

echo "Building pst Docker image..."
docker build \
  -f Dockerfile \
  --build-arg proxy=true \
  --build-arg version="$VERSION" \
  --build-arg assets_version="$ASSETS_VERSION" \
  -t palworld-server-tool .

echo "Building pst-agent Docker image..."
docker build -f Dockerfile.agent --build-arg proxy=true -t palworld-server-tool-agent .
