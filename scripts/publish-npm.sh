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
#   SKIP_BUILD=1      跳过 make dist，复用现有 dist/（CI 里产物经 artifact 传下来）
#   PUBLISH_DRY_RUN=1 只跑 npm publish --dry-run，不真正发布
#   NPM_PROVENANCE=1  追加 --provenance（仅 GitHub Actions OIDC 环境可用）
#   NODE_AUTH_TOKEN   CI 由 setup-node 注入；纯 OIDC（无 token）时预检自动跳过

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

# 查询 registry 上是否已存在 $NPM_VERSION：存在返回 0，不存在返回 1。
# 用 --json 拿机器可读的 error.code 判断，不去匹配「is not in this registry」这类
# 人类文案——文案随 npm 版本与 registry 实现而变，而误判的代价不可逆：把「不存在」
# 误判成「已存在」会跳过平台包，随后根包照发，那个版本号就废了（npm 不允许重发
# 同一版本）。仅 E404 视为「不存在」，其余错误一律终止。
npm_version_exists() {
  local pkg="$1" out code got
  # 只看 stdout：npm 的错误 JSON 写 stdout，人类可读的日志写 stderr
  if out="$(npm view "$pkg@$NPM_VERSION" version --registry "$REGISTRY" --json 2>/dev/null)"; then
    got="$(printf '%s' "$out" | jq -r '.' 2>/dev/null || true)"
    if [ "$got" = "$NPM_VERSION" ]; then
      return 0
    fi
    if [ -z "$got" ]; then
      # 部分 registry 对「版本不存在」返回退出码 0 + 空输出，按不存在处理。万一
      # 版本其实已被占用，随后的 npm publish 会报冲突——仍是安全的那一侧。
      return 1
    fi
    # 拿到非预期结果：不猜测，更不当作「不存在」而去发布，直接报错
    die "查询 $pkg 返回了非预期结果：$(printf '%s' "$out" | head -c 200)"
  fi
  code="$(printf '%s' "$out" | jq -r '.error.code // empty' 2>/dev/null || true)"
  if [ "$code" = "E404" ]; then
    return 1
  fi
  die "查询 $pkg 失败：${code:-无法从 npm 输出解析错误码}（原始输出：$(printf '%s' "$out" | head -c 200)）"
}

# ── 前置检查 ──────────────────────────────────────────────────────────
# 必须在版本解析之前：下一段用 git describe 取版本，缺 git 时它会静默失败
# （|| true），最后报成误导性的「取不到版本」，掩盖真正的原因。
for cmd in npm jq git; do
  command -v "$cmd" >/dev/null 2>&1 || die "缺少命令：$cmd"
done
# go 只在需要现场构建时才要求：CI 走 SKIP_BUILD=1，产物由 release job 编好传下来。
if [ "$SKIP_BUILD" != "1" ]; then
  command -v go >/dev/null 2>&1 ||
    die "缺少命令：go（构建需要；SKIP_BUILD=1 可复用现有 dist/）"
fi

# ── 版本 ──────────────────────────────────────────────────────────────
VERSION_TAG="${VERSION_TAG:-$(git describe --tags --abbrev=0 2>/dev/null || true)}"
[ -n "$VERSION_TAG" ] || die "取不到版本；请显式指定 VERSION_TAG=vX.Y.Z"
NPM_VERSION="${VERSION_TAG#v}"

# 发布过程要临时改写 package.json（结束还原），脏工作树会让还原语义不确定
[ -z "$(git status --porcelain)" ] || die "工作树有未提交改动，请先提交或 stash"

# 认证预检：仅在本地或显式配置了 token 时执行。npm CLI 检测到 OIDC 环境会优先用
# trusted publishing（先于 NODE_AUTH_TOKEN），而 npm whoami 不支持 OIDC——CI 里
# 无 token（纯 OIDC）时预检会误报未登录并中断发布，故跳过；该场景的认证交给
# npm publish 自己保证。
if [ "$DRY_RUN" != "1" ] &&
  { [ -z "${GITHUB_ACTIONS:-}" ] || [ -n "${NODE_AUTH_TOKEN:-}" ]; }; then
  npm whoami --registry "$REGISTRY" >/dev/null 2>&1 ||
    die "npm 未登录或 token 无效（本地请先 npm login --registry $REGISTRY）"
fi

info "版本 ${NPM_VERSION} → ${REGISTRY}"

# ── 构建 ──────────────────────────────────────────────────────────────
if [ "$SKIP_BUILD" = "1" ]; then
  warn "跳过构建（SKIP_BUILD=1），复用现有 dist/"
else
  info "构建六平台产物…"
  # 不吞输出：编译错误直接进日志，否则失败时只剩一句「make dist 失败」
  make dist || die "make dist 失败"
fi

# ── 清理钩子 ──────────────────────────────────────────────────────────
# 必须在组装平台包之前注册：组装阶段会创建 npm/*/bin 并拷入二进制，中途 die
# （如产物缺失）若不清理会留下残留，并被后续 SKIP_BUILD 重跑掩盖成「已就位」。
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

# ── 平台映射一致性校验 ────────────────────────────────────────────────
# 平台清单在三处各自维护：platform.js 的 PLATFORM_PKG、根 package.json 的
# optionalDependencies、本脚本的 PLATFORMS。任一处漏改都会静默失配——运行时解析
# 不到二进制，却很难定位到根因，故先把键集对齐再动手。
PLATFORM_PKGS=()
for entry in "${PLATFORMS[@]}"; do
  dir="${entry%%:*}"
  PLATFORM_PKGS+=("$(jq -r '.name' "$PROJECT_ROOT/npm/$dir/package.json")")
done
EXPECTED="$(printf '%s\n' "${PLATFORM_PKGS[@]}" | jq -R . | jq -s -c 'sort')"
DECLARED="$(jq -c '.optionalDependencies | keys | sort' "$PROJECT_ROOT/package.json")"
[ "$EXPECTED" = "$DECLARED" ] ||
  die "PLATFORMS 与 package.json 的 optionalDependencies 不一致：
    脚本：$EXPECTED
    包内：$DECLARED"

# platform.js 的 PLATFORM_PKG 是运行时解析平台包的唯一依据，也要纳入：只对齐另两处
# 仍可能留下「发布成功但用户装不起来」的静默失配。
PLATFORM_JS="$(node -e 'process.stdout.write(JSON.stringify(Object.values(require("./scripts/platform").PLATFORM_PKG).sort()))' 2>/dev/null || true)"
[ "$PLATFORM_JS" = "$EXPECTED" ] ||
  die "PLATFORMS 与 scripts/platform.js 的 PLATFORM_PKG 不一致：
    脚本：$EXPECTED
    platform.js：${PLATFORM_JS:-（读取失败）}"

# ── 组装平台包 ────────────────────────────────────────────────────────
for entry in "${PLATFORMS[@]}"; do
  IFS=':' read -r dir dist_bin bin_name <<< "$entry"
  src="$PROJECT_ROOT/dist/$dist_bin"
  [ -f "$src" ] || die "缺少产物 $src"
  mkdir -p "$PROJECT_ROOT/npm/$dir/bin"
  cp "$src" "$PROJECT_ROOT/npm/$dir/bin/$bin_name"
  # CI 里产物经 upload/download-artifact 传递，而 v4 不保留文件模式，故此处 chmod
  # 不能省。也不加 || true：权限没设上就该终止——同版本不可重发，把不可执行的
  # 二进制推出去，代价是由用户承担的。
  chmod 755 "$PROJECT_ROOT/npm/$dir/bin/$bin_name"
done
ok "六平台二进制已就位"

# ── 注入版本（临时；退出时还原） ──────────────────────────────────────
# 只改本仓库管辖的六个平台包。若整体改写 optionalDependencies，将来往根包加任何
# 别的可选依赖都会被顺手覆盖成同一版本号。
cp "$PROJECT_ROOT/package.json" "$PROJECT_ROOT/package.json.bak"
PLATFORM_PKGS_JSON="$(printf '%s\n' "${PLATFORM_PKGS[@]}" | jq -R . | jq -s -c .)"
jq --arg v "$NPM_VERSION" --argjson pkgs "$PLATFORM_PKGS_JSON" \
  '.version = $v
   | .optionalDependencies |= with_entries(
       .key as $k | if ($pkgs | index($k)) then .value = $v else . end)' \
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
  if npm_version_exists "$name"; then
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
    npm_version_exists "$name" || die "$name@$NPM_VERSION 未就位，根包不能先发"
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
