#!/usr/bin/env bash
# Linux 本地预构建脚本(服务器上直接用 / WSL / macOS 均可)
#
# 用法:
#   bash deploy/build-local.sh            # 增量:只建缺失的 goose
#   bash deploy/build-local.sh --force    # 强制重建 goose
#
# 产物:
#   deploy/bin/gpt2api        linux/amd64 可执行(后端)
#   deploy/bin/goose          linux/amd64 可执行(迁移工具)
#   web/dist/                 前端 Vite 产物
#
# 这套产物 + deploy/Dockerfile 就可以离线构建镜像,无需容器再访问外网。

set -euo pipefail

FORCE=0
for arg in "$@"; do
    case "$arg" in
        -f|--force) FORCE=1 ;;
    esac
done

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "[build-local] repo  = $ROOT"

# ---- step1: 交叉编译 gpt2api ----
echo "[build-local] step1 = cross-build gpt2api (linux/amd64)"
mkdir -p deploy/bin
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags "-s -w" -o deploy/bin/gpt2api ./cmd/server

# ---- step2: 编译 goose ----
GOOSE="$ROOT/deploy/bin/goose"
if [ "$FORCE" = "1" ] || [ ! -x "$GOOSE" ]; then
    echo "[build-local] step2 = cross-build goose (tmp module)"
    TMP="$(mktemp -d)"
    trap 'rm -rf "$TMP"' EXIT
    pushd "$TMP" >/dev/null
    go mod init goose-wrapper >/dev/null 2>&1
    go get github.com/pressly/goose/v3/cmd/goose@v3.20.0 >/dev/null 2>&1
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
        go build -ldflags "-s -w" -o "$GOOSE" github.com/pressly/goose/v3/cmd/goose
    popd >/dev/null
else
    echo "[build-local] step2 = skip goose (exists). use --force to rebuild"
fi

# ---- step3: 前端 ----
echo "[build-local] step3 = npm run build (web)"
pushd web >/dev/null
NEED_NPM_INSTALL=0
if [ ! -d node_modules ]; then
    NEED_NPM_INSTALL=1
elif [ ! -f node_modules/.package-lock.json ] || [ package-lock.json -nt node_modules/.package-lock.json ]; then
    NEED_NPM_INSTALL=1
elif [ ! -d node_modules/dompurify ] || [ ! -d node_modules/@wangeditor/editor ]; then
    NEED_NPM_INSTALL=1
fi

if [ "$NEED_NPM_INSTALL" = "1" ]; then
    echo "[build-local] step3.1 = npm install (web dependencies changed or incomplete)"
    npm install --no-audit --no-fund --loglevel=error
fi
npm run build
popd >/dev/null

echo "[build-local] done. artifacts:"
ls -lh deploy/bin/gpt2api deploy/bin/goose web/dist/index.html