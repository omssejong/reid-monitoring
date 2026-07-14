#!/usr/bin/env bash
#
# smoke_test.sh
# omeye3.monitoring.service 가 정상적으로 올라와 있는지 확인한다.
#   - 정상(active/running): exit 0
#   - 비정상(그 외 상태):    exit 1

set -u

SERVICE="omeye3.monitoring.service"

# systemctl 존재 여부 확인
if ! command -v systemctl >/dev/null 2>&1; then
    echo "[FAIL] systemctl 을 찾을 수 없습니다."
    exit 1
fi

# 서비스가 active 상태인지 확인
if systemctl is-active --quiet "${SERVICE}"; then
    echo "[OK] ${SERVICE} 가 정상적으로 동작 중입니다."
    exit 0
fi

# 실패 시 현재 상태를 함께 출력하여 디버깅을 돕는다.
STATE="$(systemctl is-active "${SERVICE}" 2>/dev/null || true)"
echo "[FAIL] ${SERVICE} 가 정상 상태가 아닙니다. (현재 상태: ${STATE:-unknown})"
exit 1
