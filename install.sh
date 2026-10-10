#!/bin/sh

# soulink CLI 一键安装脚本（从 GitHub Releases 下载）。
#   curl -fsSL https://www.soulink.cam/install.sh | sh
# 先审计再执行：
#   curl -fsSL https://www.soulink.cam/install.sh -o install.sh
#   less install.sh && sh install.sh
# 环境变量：
#   SOULINK_INSTALL_DIR    安装目录（默认 /usr/local/bin，不可写时用 sudo）
#   SOULINK_VERSION        指定版本（默认 latest，跳过 GitHub API 查询）
#   SOULINK_GITHUB_MIRROR  GitHub 镜像域名，供国内加速（如 ghfast.top，不带 https:// 前缀）
set -eu

main() {
  REPO="soulink9/soulink"
  BIN="soulink"
  INSTALL_DIR="${SOULINK_INSTALL_DIR:-/usr/local/bin}"
  VERSION="${SOULINK_VERSION:-}"

  command -v curl >/dev/null 2>&1 || err "需要 curl"

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin | linux) ;;
    *) err "暂不支持 $os；Windows 请从 https://github.com/$REPO/releases 下载 zip" ;;
  esac

  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64) arch="amd64" ;;
    arm64 | aarch64) arch="arm64" ;;
    *) err "不支持的架构：$arch" ;;
  esac

  if [ -z "$VERSION" ]; then
    release_json="$(curl -fsSL --connect-timeout 5 --max-time 20 \
      "https://api.github.com/repos/$REPO/releases/latest")" ||
      err "查询最新版本失败；可设 SOULINK_VERSION=vX.Y.Z 跳过 GitHub API"
    VERSION="$(printf '%s' "$release_json" |
      sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
    [ -n "$VERSION" ] || err "解析最新版本号失败"
  fi

  pkg="${BIN}_${VERSION}_${os}_${arch}.tar.gz"
  mirror="$(printf '%s' "${SOULINK_GITHUB_MIRROR:-}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
  mirror="${mirror#https://}"
  mirror="${mirror#http://}"
  mirror="${mirror%/}"
  if [ -n "$mirror" ]; then
    printf '提示：经由第三方镜像 "%s" 下载（完整性由 sha256 校验）\n' "$mirror" >&2
    base="https://${mirror}/github.com/$REPO/releases/download/$VERSION"
  else
    base="https://github.com/$REPO/releases/download/$VERSION"
  fi

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' INT TERM EXIT

  printf '下载 soulink %s (%s/%s)...\n' "$VERSION" "$os" "$arch"
  curl -fL --progress-bar --connect-timeout 5 --speed-limit 1024 --speed-time 30 \
    -o "$tmp/$pkg" "$base/$pkg" ||
    err "下载失败：$base/$pkg（国内网络可设 SOULINK_GITHUB_MIRROR=gh-proxy.com 走镜像加速）"
  curl -fsSL --connect-timeout 5 --max-time 15 -o "$tmp/checksums.txt" "$base/checksums.txt" ||
    err "checksums.txt 下载失败"

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
