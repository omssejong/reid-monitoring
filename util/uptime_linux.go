//go:build linux

package util

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// systemUptime /proc/uptime 을 직접 읽어 부팅 후 경과 시간을 반환한다. (cat 프로세스 스폰 제거)
func systemUptime() (time.Duration, error) {
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	serverRunTime := strings.Split(string(raw), " ")[0]
	convServerRunTime, err := strconv.ParseFloat(serverRunTime, 32)
	if err != nil {
		return 0, err
	}

	return time.Duration(convServerRunTime * float64(time.Second)), nil
}
