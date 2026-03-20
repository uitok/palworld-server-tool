#!/bin/sh
set -eu

usage() {
  cat <<'USAGE'
Usage:
  sh ./script/download-release-asset.sh <asset-name> [output-path]

Environment variables:
  PST_RELEASE_OWNER     GitHub owner, default: zaigie
  PST_RELEASE_REPO      GitHub repository, default: palworld-server-tool
  PST_RELEASE_VERSION   Preferred release tag. If the asset is missing on that tag,
                        the script falls back to releases/latest/download.
USAGE
}

asset_name="${1:-}"
output_path="${2:-}"

if [ -z "$asset_name" ]; then
  usage
  exit 1
fi

if [ -z "$output_path" ]; then
  output_path="$asset_name"
fi

release_owner="${PST_RELEASE_OWNER:-zaigie}"
release_repo="${PST_RELEASE_REPO:-palworld-server-tool}"
release_version="${PST_RELEASE_VERSION:-}"

if [ -z "$release_version" ] && command -v git >/dev/null 2>&1; then
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    release_version="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  fi
fi

fallback_version="v0.9.9"
output_dir="$(dirname "$output_path")"
mkdir -p "$output_dir"

tmp_path="${output_path}.tmp"
cleanup() {
  rm -f "$tmp_path"
}
trap cleanup EXIT INT TERM

try_download() {
  url="$1"
  printf 'Downloading %s\n' "$url"
  if curl -fL --retry 3 --retry-delay 1 --output "$tmp_path" "$url"; then
    mv "$tmp_path" "$output_path"
    printf 'Saved to %s\n' "$output_path"
    return 0
  fi
  return 1
}

if [ -n "$release_version" ]; then
  if try_download "https://github.com/${release_owner}/${release_repo}/releases/download/${release_version}/${asset_name}"; then
    exit 0
  fi
  printf 'Preferred release %s does not provide %s, falling back...\n' "$release_version" "$asset_name" >&2
fi

if try_download "https://github.com/${release_owner}/${release_repo}/releases/latest/download/${asset_name}"; then
  exit 0
fi

if [ -z "$release_version" ] || [ "$release_version" != "$fallback_version" ]; then
  if try_download "https://github.com/${release_owner}/${release_repo}/releases/download/${fallback_version}/${asset_name}"; then
    exit 0
  fi
fi

printf 'Failed to download asset %s from preferred release, latest release, and fallback %s\n' "$asset_name" "$fallback_version" >&2
exit 1
