#!/bin/sh

# soulink CLI 一键安装脚本（从 GitHub Releases 下载）。
#   curl -fsSL https://www.soulink.cam/install.sh | sh
# 先审计再执行：
#   curl -fsSL https://www.soulink.cam/install.sh -o install.sh
#   less install.sh && sh install.sh
# 下载源顺序：SOULINK_GITHUB_MIRROR（显式指定）> github.com 直连 > 内置镜像
# （gh-proxy.com、ghproxy.net）。国内网络直连超时属正常，脚本会自动回落镜像；
# 无论走哪个源，包都对照 checksums.txt 做 sha256 强校验。
# 环境变量：
#   SOULINK_INSTALL_DIR    安装目录（默认 /usr/local/bin，不可写时用 sudo）
#   SOULINK_VERSION        指定版本（默认自动解析 latest）
#   SOULINK_GITHUB_MIRROR  显式指定镜像域名（如 gh-proxy.com，不带 https:// 前缀）
set -eu

REPO="soulink9/soulink"
BIN="soulink"
BUILTIN_MIRRORS="gh-proxy.com ghproxy.net"

main() {
  INSTALL_DIR="${SOULINK_INSTALL_DIR:-/usr/local/bin}"
  VERSION="${SOULINK_VERSION:-}"

  command -v curl >/dev/null 2>&1 || err "需要 curl"

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin | linux) ;;
    *) err "暂不支持 $os；Windows 请用 PowerShell：irm https://www.soulink.cam/install.ps1 | iex" ;;
  esac

  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64) arch="amd64" ;;
    arm64 | aarch64) arch="arm64" ;;
    *) err "不支持的架构：$arch" ;;
  esac

  [ -n "$VERSION" ] || VERSION="$(resolve_version)"
  pkg="${BIN}_${VERSION}_${os}_${arch}.tar.gz"

  # 候选下载源：显式镜像 > 直连 > 内置镜像
  set --
  mirror="$(normalize_mirror "${SOULINK_GITHUB_MIRROR:-}")"
  [ -z "$mirror" ] || set -- "$@" "https://$mirror/github.com/$REPO/releases/download/$VERSION"
  set -- "$@" "https://github.com/$REPO/releases/download/$VERSION"
  for m in $BUILTIN_MIRRORS; do
    set -- "$@" "https://$m/github.com/$REPO/releases/download/$VERSION"
  done

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' INT TERM EXIT

  printf '下载 soulink %s (%s/%s)...\n' "$VERSION" "$os" "$arch"
  base=""
  for candidate do
    if curl -fL --progress-bar --connect-timeout 5 --speed-limit 1024 --speed-time 30 \
      -o "$tmp/$pkg" "$candidate/$pkg"; then
      base="$candidate"
      break
    fi
    printf '提示：%s 不可达，尝试下一个源\n' "$candidate" >&2
  done
  [ -n "$base" ] || err "所有下载源均失败；可设 SOULINK_GITHUB_MIRROR=<镜像域名> 后重试"
  case "$base" in
    https://github.com/*) ;;
    *) printf '提示：经由第三方镜像 "%s" 下载（完整性由 sha256 校验）\n' "$base" >&2 ;;
  esac

  curl -fsSL --connect-timeout 5 --max-time 15 -o "$tmp/checksums.txt" "$base/checksums.txt" ||
    err "checksums.txt 下载失败（$base）"

  want="$(awk -v a="$pkg" '$2 == a {print tolower($1)}' "$tmp/checksums.txt")"
  [ -n "$want" ] || err "checksums.txt 中找不到 $pkg"
  got="$(sha256 "$tmp/$pkg" | awk '{print tolower($1)}')"
  [ "$got" = "$want" ] || err "校验和不匹配（got $got, want $want）"

  tar -xzf "$tmp/$pkg" -C "$tmp" || err "解压失败"
  install_binary "$tmp/$BIN-$os-$arch" "$INSTALL_DIR" "$BIN"

  printf '已安装 soulink %s -> %s\n' "$VERSION" "$INSTALL_DIR/$BIN"
  path_notice "$BIN" "$INSTALL_DIR"
  printf '\n直接使用（匿名、免注册）：\n  soulink 3000\n\n固定域名：到 https://hub.soulink.cam 注册后执行\n  soulink config set token <token>\n'
}

# 解析最新版本号：先试 GitHub API（国内通常可达），再试内置镜像的
# /releases/latest 重定向；都失败则提示显式指定 SOULINK_VERSION。
resolve_version() {
  json="$(curl -fsSL --connect-timeout 5 --max-time 20 \
    "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null)" || json=""
  if [ -n "$json" ]; then
    v="$(printf '%s' "$json" |
      sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
    [ -z "$v" ] || { printf '%s' "$v"; return; }
  fi
  for m in $BUILTIN_MIRRORS; do
    v="$(curl -fsSI --connect-timeout 5 --max-time 15 \
      "https://$m/https://github.com/$REPO/releases/latest" 2>/dev/null |
      tr -d '\r' | sed -n 's|^[Ll]ocation:.*/tag/\([^[:space:]]*\).*|\1|p' | head -n1)"
    [ -z "$v" ] || { printf '%s' "$v"; return; }
  done
  err "无法解析最新版本；可设 SOULINK_VERSION=vX.Y.Z 后重试"
}

normalize_mirror() {
  m="$(printf '%s' "$1" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
  m="${m#https://}"
  m="${m#http://}"
  m="${m%/}"
  case "$m" in *[[:space:]]*) err "SOULINK_GITHUB_MIRROR 含有空格：'$m'" ;; esac
  printf '%s' "$m"
}

# 安装二进制（0755）；目录不可写时用 sudo 提权。
install_binary() {
  src="$1"
  dir="$2"
  bin="$3"
  if mkdir -p "$dir" 2>/dev/null && [ -w "$dir" ]; then
    install -m 0755 "$src" "$dir/$bin"
  elif command -v sudo >/dev/null 2>&1; then
    printf '提示：%s 不可写，使用 sudo 安装\n' "$dir"
    sudo mkdir -p "$dir"
    sudo install -m 0755 "$src" "$dir/$bin"
  else
    err "$dir 不可写且无 sudo；请设 SOULINK_INSTALL_DIR 为可写目录"
  fi
}

path_notice() {
  bin="$1"
  dir="$2"
  case ":$PATH:" in
    *":$dir:"*) ;;
    *) printf '提示：%s 不在 PATH 中，请添加或直接运行 %s/%s\n' "$dir" "$dir" "$bin" ;;
  esac
}

# 优先 shasum（macOS），其次 sha256sum（Linux）。
sha256() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1"
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1"
  else
    err "需要 shasum 或 sha256sum 进行校验"
  fi
}

err() { printf '错误：%s\n' "$1" >&2; exit 1; }

main "$@"
