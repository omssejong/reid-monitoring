//go:build windows

package util

import (
	"time"

	"golang.org/x/sys/windows"
)

var (
	modKernel32Uptime  = windows.NewLazySystemDLL("kernel32.dll")
	procGetTickCount64 = modKernel32Uptime.NewProc("GetTickCount64")
)

// systemUptime GetTickCount64 로 부팅 후 경과 시간을 반환한다. (/proc/uptime 대응)
// 49.7일에 래핑되는 GetTickCount 와 달리 64비트라 사실상 오버플로가 없다.
func systemUptime() (time.Duration, error) {
	ms, _, _ := procGetTickCount64.Call()
	return time.Duration(ms) * time.Millisecond, nil
}
