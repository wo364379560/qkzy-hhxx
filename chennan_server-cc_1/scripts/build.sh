#!/bin/sh

set -e

Prefix=$1

Services=$(basename "$(pwd)")
Microservice=$(basename "cmd/$2")
ServicesVersion=$(cat .version)

BINARY="bin/${Prefix}-${Microservice}"
SOURCE="cmd/${Microservice}/main.go"
# 交叉编译时带上系统/架构后缀，避免 amd64 / arm64 互相覆盖
if [ -n "${GOOS:-}" ] && [ -n "${GOARCH:-}" ]; then
  BINARY="bin/${Prefix}-${Microservice}-${GOOS}-${GOARCH}"
fi

MODULE=$(go list -m)
ENV=$(uname -s)
UPT=$(date +%Y-%m-%dT%H:%M:%S%z)
VER=$(cat "internal/app/${Microservice}/.version")
TAG=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
  TAG="${TAG}-dev"
fi
BRS=$(git symbolic-ref --short -q HEAD 2>/dev/null || echo unknown)

# 部署到 Linux 服务器请显式传 GOOS/GOARCH，例如：
#   make server-linux
TARGET_OS=${GOOS:-}
TARGET_ARCH=${GOARCH:-}
if [ -n "$TARGET_OS" ] || [ -n "$TARGET_ARCH" ]; then
  echo "Target  ${TARGET_OS:-host}/${TARGET_ARCH:-host}"
fi

if [ -z "${GOPROXY:-}" ]; then
  export GOPROXY=https://goproxy.cn,direct
fi
if [ -z "${GOSUMDB:-}" ]; then
  export GOSUMDB=sum.golang.google.cn
fi

echo "Release ${Prefix}-${Microservice} :${ServicesVersion}.${VER}@${TAG}"

mkdir -p "$(dirname "${BINARY}")"

LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.verStr=${VER}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.brsStr=${BRS}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.tagStr=${TAG}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.uptStr=${UPT}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.envStr=${ENV}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.Prefix=${Prefix}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.Services=${Services}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.MicroService=${Microservice}"
LDFLAGS="$LDFLAGS -X ${MODULE}/pkg/ix.ServicesVersion=${ServicesVersion}"

CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
  go build -trimpath -ldflags "$LDFLAGS" -o "${BINARY}" "${SOURCE}"

file "${BINARY}" 2>/dev/null || true
