#!/usr/bin/env bash
set -euo pipefail

repo="chxcodepro/qilin-manager-tui"
arch="$(uname -m)"

case "$arch" in
  x86_64|amd64)
    asset_arch="amd64"
    ;;
  aarch64|arm64)
    asset_arch="arm64"
    ;;
  *)
    echo "暂不支持的架构: $arch" >&2
    exit 1
    ;;
esac

asset="qilin-manager-tui_linux_${asset_arch}.tar.gz"
url="https://github.com/${repo}/releases/latest/download/${asset}"
install_dir="${QILIN_MANAGER_TUI_HOME:-$HOME/.local/bin}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

mkdir -p "$install_dir"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp_dir/app.tar.gz"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_dir/app.tar.gz" "$url"
else
  echo "系统里没有 curl 或 wget，没法下载程序" >&2
  exit 1
fi

tar -xzf "$tmp_dir/app.tar.gz" -C "$tmp_dir"
install -m 0755 "$tmp_dir/qilin-manager-tui" "$install_dir/qilin-manager-tui"
exec "$install_dir/qilin-manager-tui"
