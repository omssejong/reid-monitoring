#!/usr/bin/env bash
#
# build.sh
# 빌드 시각을 바이너리에 주입해서 리눅스용 실행 파일을 만든다.
# ldflags를 빼먹으면 빌드 시각 대신 마지막 커밋 시각이 표시되므로 이 스크립트로 빌드한다.
#
#   ./build.sh              → oms-monitoring 생성
#   ./build.sh my-binary    → 지정한 이름으로 생성

set -eu

OUTPUT="${1:-oms-monitoring}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
PKG="github.com/omssejong/reid-monitoring/util"

GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X ${PKG}.buildTime=${BUILD_TIME}" \
    -o "${OUTPUT}"

echo "[OK] ${OUTPUT} (buildTime=${BUILD_TIME})"
