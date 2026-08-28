#!/usr/bin/env bash
#
# build.sh
# 빌드 시각을 바이너리에 주입해서 실행 파일을 만든다.
# ldflags를 빼먹으면 빌드 시각 대신 마지막 커밋 시각이 표시되므로 이 스크립트로 빌드한다.
#
#   ./build.sh                      → oms-monitoring        (linux/amd64)
#   ./build.sh my-binary            → 지정한 이름으로 생성   (linux/amd64)
#   ./build.sh "" windows           → oms-monitoring.exe    (windows/amd64)
#   ./build.sh my-binary.exe windows → 지정한 이름으로 생성  (windows/amd64)
#
# GOARCH 는 환경변수로 덮어쓸 수 있다 (기본 amd64).

set -eu

TARGET_OS="${2:-linux}"
TARGET_ARCH="${GOARCH:-amd64}"

OUTPUT="${1:-}"
if [ -z "${OUTPUT}" ]; then
    if [ "${TARGET_OS}" = "windows" ]; then
        OUTPUT="oms-monitoring.exe"
    else
        OUTPUT="oms-monitoring"
    fi
fi

BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
PKG="github.com/omssejong/reid-monitoring/util"

GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" go build \
    -ldflags "-X ${PKG}.buildTime=${BUILD_TIME}" \
    -o "${OUTPUT}"

echo "[OK] ${OUTPUT} (${TARGET_OS}/${TARGET_ARCH}, buildTime=${BUILD_TIME})"
