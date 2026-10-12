#!/usr/bin/env bash
# SPDX-License-Identifier: MIT
#
# publish-npm.sh — 发布六平台二进制包与 Node launcher 主包到 npm。
#
# 顺序不可颠倒：平台包必须先于根包。根包若先发，安装期 optionalDependencies
# 解析不到对应版本会被 npm 静默跳过，用户拿到的是「装上了但跑不起来」。
#
# 用法：
#   scripts/publish-npm.sh                      # 本地：版本取 git describe
#   VERSION_TAG=v0.1.5 scripts/publish-npm.sh   # 指定版本（CI 传 github.ref_name）
#
# 环境变量：
#   VERSION_TAG       覆盖版本来源（vX.Y.Z 或 X.Y.Z）
#   NPM_REGISTRY      发布 registry，默认官方（本机 registry 可能指向镜像，发布须显式指定）
#   SKIP_BUILD=1      跳过 make dist，复用现有 dist/
#   PUBLISH_DRY_RUN=1 只跑 npm publish --dry-run，不真正发布
#   NPM_PROVENANCE=1  追加 --provenance（仅 GitHub Actions OIDC 环境可用）

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

REGISTRY="${NPM_REGISTRY:-https://registry.npmjs.org/}"
SKIP_BUILD="${SKIP_BUILD:-0}"
DRY_RUN="${PUBLISH_DRY_RUN:-0}"

# 平台映射：npm 目录 : dist 产物名 : 包内二进制文件名
# 左侧是 Node 命名（对应 package.json 的 os/cpu），dist 产物是 Go 命名。
PLATFORMS=(
  "darwin-arm64:soulink-darwin-arm64:soulink"
  "darwin-x64:soulink-darwin-amd64:soulink"
  "linux-arm64:soulink-linux-arm64:soulink"
  "linux-x64:soulink-linux-amd64:soulink"
  "win32-arm64:soulink-windows-arm64.exe:soulink.exe"
  "win32-x64:soulink-windows-amd64.exe:soulink.exe"
)

info() { printf '\033[0;34m[INFO]\033[0m  %s\n' "$*"; }
ok()   { printf '\033[0;32m[OK]\033[0m    %s\n' "$*"; }
warn() { printf '\033[0;33m[WARN]\033[0m  %s\n' "$*" >&2; }
die()  { printf '\033[0;31m[ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

# ── 版本 ──────────────────────────────────────────────────────────────
VERSION_TAG="${VERSION_TAG:-$(git describe --tags --abbrev=0 2>/dev/null || true)}"
[ -n "$VERSION_TAG" ] || die "取不到版本；请显式指定 VERSION_TAG=vX.Y.Z"
NPM_VERSION="${VERSION_TAG#v}"

# ── 前置检查 ──────────────────────────────────────────────────────────
for cmd in go npm jq git; do
  command -v "$cmd" >/dev/null 2>&1 || die "缺少命令：$cmd"
done

# 发布过程要临时改写 package.json（结束还原），脏工作树会让还原语义不确定
[ -z "$(git status --porcelain)" ] || die "工作树有未提交改动，请先提交或 stash"

if [ "$DRY_RUN" != "1" ]; then
  npm whoami --registry "$REGISTRY" >/dev/null 2>&1 ||
    die "npm 未登录或 token 无效（本地请先 npm login --registry $REGISTRY）"
fi

info "版本 ${NPM_VERSION} → ${REGISTRY}"

# ── 构建 ──────────────────────────────────────────────────────────────
if [ "$SKIP_BUILD" = "1" ]; then
  warn "跳过构建（SKIP_BUILD=1），复用现有 dist/"
else
  info "构建六平台产物…"
  make dist >/dev/null || die "make dist 失败"
fi

# ── 组装平台包 ────────────────────────────────────────────────────────
for entry in "${PLATFORMS[@]}"; do
  IFS=':' read -r dir dist_bin bin_name <<< "$entry"
  src="$PROJECT_ROOT/dist/$dist_bin"
  [ -f "$src" ] || die "缺少产物 $src"
  mkdir -p "$PROJECT_ROOT/npm/$dir/bin"
  cp "$src" "$PROJECT_ROOT/npm/$dir/bin/$bin_name"
  chmod 755 "$PROJECT_ROOT/npm/$dir/bin/$bin_name" 2>/dev/null || true
done
ok "六平台二进制已就位"

# ── 注入版本（临时；退出时连同二进制一并还原） ────────────────────────
restore() {
  local entry dir pkg
  for entry in "${PLATFORMS[@]}"; do
    dir="${entry%%:*}"
    pkg="$PROJECT_ROOT/npm/$dir/package.json"
    if [ -f "$pkg.bak" ]; then mv "$pkg.bak" "$pkg"; fi
    rm -rf "$PROJECT_ROOT/npm/$dir/bin"
  done
  if [ -f "$PROJECT_ROOT/package.json.bak" ]; then
    mv "$PROJECT_ROOT/package.json.bak" "$PROJECT_ROOT/package.json"
  fi
}
trap restore EXIT

cp "$PROJECT_ROOT/package.json" "$PROJECT_ROOT/package.json.bak"
jq --arg v "$NPM_VERSION" \
  '.version = $v | .optionalDependencies |= with_entries(.value = $v)' \
  "$PROJECT_ROOT/package.json.bak" > "$PROJECT_ROOT/package.json"

for entry in "${PLATFORMS[@]}"; do
  dir="${entry%%:*}"
  pkg="$PROJECT_ROOT/npm/$dir/package.json"
  cp "$pkg" "$pkg.bak"
  jq --arg v "$NPM_VERSION" '.version = $v' "$pkg.bak" > "$pkg"
done
ok "版本已注入 ${NPM_VERSION}"

# ── 发布平台包 ────────────────────────────────────────────────────────
PUBLISH_ARGS=(--registry "$REGISTRY" --access public)
[ "$DRY_RUN" = "1" ] && PUBLISH_ARGS+=(--dry-run)
[ "${NPM_PROVENANCE:-0}" = "1" ] && PUBLISH_ARGS+=(--provenance)

for entry in "${PLATFORMS[@]}"; do
  dir="${entry%%:*}"
  name="$(jq -r '.name' "$PROJECT_ROOT/npm/$dir/package.json")"
  already="$(npm view "$name@$NPM_VERSION" version --registry "$REGISTRY" 2>/dev/null || true)"
  if [ "$already" = "$NPM_VERSION" ]; then
    warn "  $name@$NPM_VERSION 已存在，跳过"
    continue
  fi
  info "  发布 $name@$NPM_VERSION"
  npm publish "$PROJECT_ROOT/npm/$dir" "${PUBLISH_ARGS[@]}" ||
    die "发布 $name 失败"
  ok "  $name@$NPM_VERSION"
done

# ── 校验平台包全部就位后，才允许发根包 ────────────────────────────────
if [ "$DRY_RUN" != "1" ]; then
  for entry in "${PLATFORMS[@]}"; do
    dir="${entry%%:*}"
    name="$(jq -r '.name' "$PROJECT_ROOT/npm/$dir/package.json")"
    got="$(npm view "$name@$NPM_VERSION" version --registry "$REGISTRY" 2>/dev/null || true)"
    [ "$got" = "$NPM_VERSION" ] || die "$name@$NPM_VERSION 未就位，根包不能先发"
  done
  ok "六平台包均已就位"
fi

# ── 发布根包 ──────────────────────────────────────────────────────────
ROOT_ARGS=(--registry "$REGISTRY")
[ "$DRY_RUN" = "1" ] && ROOT_ARGS+=(--dry-run)
[ "${NPM_PROVENANCE:-0}" = "1" ] && ROOT_ARGS+=(--provenance)

ROOT_NAME="$(jq -r '.name' "$PROJECT_ROOT/package.json")"
info "发布根包 $ROOT_NAME@$NPM_VERSION"
npm publish "$PROJECT_ROOT" "${ROOT_ARGS[@]}" || die "根包发布失败"
ok "根包 $ROOT_NAME@$NPM_VERSION"

echo ""
ok "全部完成：npm install -g $ROOT_NAME"
