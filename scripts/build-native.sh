#!/usr/bin/env bash
# 构建原生 fpk 产物:双架构静态二进制 + 前端 dist,落入 fnos/(打包目录)。
#
# 产物(均被 gitignore,fnpack build 前必须先跑本脚本):
#   fnos/app/bin/pigeonbox-linux-amd64
#   fnos/app/bin/pigeonbox-linux-arm64
#   fnos/app/www/            (前端构建产物)
#
# 前端来源优先级(与 openwrt/scripts/build-frontend.sh 同策略):
#   1. FRONTEND_DIST 环境变量指定的现成 dist 目录
#   2. 工作区已检出的 frontend 仓(../frontend,与 fnos 同级,hub make setup 布局)
#   3. 临时克隆 pigeonbox/frontend <FRONTEND_REF,默认 main>
# 只跑 vite build fnos flavor(类型检查由 frontend 壳仓 CI 独立把守);wire 类型依赖
# @pigeonbox/contracts 的 Release tgz 资产(匿名可下)。
#
# 版本注入:VERSION/COMMIT 环境变量优先,缺省取 git describe/rev-parse。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PACK_DIR="$ROOT/fnos"
BIN_DIR="$PACK_DIR/app/bin"
WWW_DIR="$PACK_DIR/app/www"
FRONTEND_REF=${FRONTEND_REF:-main}

VERSION=${VERSION:-"$(git -C "$ROOT" describe --tags --always 2>/dev/null || echo dev)"}
COMMIT=${COMMIT:-"$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"}
BUILD_TIME=${BUILD_TIME:-"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
LDFLAGS="-w -s -X 'github.com/pigeonbox/kit/version.Version=${VERSION}' -X 'github.com/pigeonbox/kit/version.BuildCommit=${COMMIT}' -X 'github.com/pigeonbox/kit/version.BuildTime=${BUILD_TIME}'"

echo "==> 构建原生二进制 (CGO_ENABLED=0 静态链接,纯 Go sqlite)"
mkdir -p "$BIN_DIR"
for arch in amd64 arm64; do
  echo "    GOOS=linux GOARCH=${arch}"
  # GOWORK=off:保证构建严格按 go.mod 钉版(否则 hub go.work 联编本地 core main,
  # 产物含未发布代码且不可复现——真机修复到位性依赖钉版)。
  CGO_ENABLED=0 GOWORK=off GOOS=linux GOARCH=${arch} \
    go build -C "$ROOT" -ldflags="${LDFLAGS}" \
    -o "$BIN_DIR/pigeonbox-linux-${arch}" ./cmd/fnos-adapter
done

echo "==> 构建前端 dist → $WWW_DIR"
rm -rf "$WWW_DIR"
SRC="${FRONTEND_DIST:-}"
CLEANUP_SRC=""
if [ -z "$SRC" ]; then
  if [ -f "$ROOT/../frontend/package.json" ]; then
    SRC="$(cd "$ROOT/../frontend" && pwd)"
    echo "    使用工作区 frontend: $SRC"
  else
    SRC="$(mktemp -d)/frontend"
    CLEANUP_SRC="$SRC"
    echo "    克隆 frontend@$FRONTEND_REF"
    git clone -q --depth 1 -b "$FRONTEND_REF" \
      "https://github.com/pigeonbox/frontend.git" "$SRC"
  fi
fi
trap '[ -n "${CLEANUP_SRC:-}" ] && rm -rf "$(dirname "$CLEANUP_SRC")"' EXIT

cd "$SRC"
[ -d node_modules ] || npm ci --no-audit --no-fund
# fnos flavor:壳入口注入宿主适配器(2026-10-09 拆分双仓后 neutral 产物无适配器,
# fpk 必须用 build:fnos flavor;vite.fnos.config.ts 内置 publicDir=core public)
npx vite build --config vite.fnos.config.ts --outDir "$WWW_DIR" --emptyOutDir

echo "==> 完成"
ls -lh "$BIN_DIR"
du -sh "$WWW_DIR"

# 打包配置:与 configs/config.yaml 单源同步(fnpack 打包目录内为只读副本)
mkdir -p "$PACK_DIR/app/configs"
cp "$ROOT/configs/config.yaml" "$PACK_DIR/app/configs/config.yaml"
echo "==> 配置副本就绪: $PACK_DIR/app/configs/config.yaml"
