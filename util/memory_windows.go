//go:build windows

package util

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// memoryStatusEx MEMORYSTATUSEX (sysinfoapi.h)
type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

var (
	modKernel32Mem           = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modKernel32Mem.NewProc("GlobalMemoryStatusEx")
)

// readMemInfo GlobalMemoryStatusEx 로 물리 메모리 상태를 조회한다.
// 윈도우에는 리눅스의 MemFree/MemAvailable 구분이 없으므로
// available/free 모두 ullAvailPhys(즉시 할당 가능한 물리 메모리)를 사용한다.
func readMemInfo() (memInfo, error) {
	var status memoryStatusEx
	status.length = uint32(unsafe.Sizeof(status))

	r, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if r == 0 {
		return memInfo{}, fmt.Errorf("GlobalMemoryStatusEx failed: %v", err)
	}
	if status.totalPhys == 0 {
		return memInfo{}, fmt.Errorf("total physical memory not found")
	}

	return memInfo{
		totalBytes:     status.totalPhys,
		availableBytes: status.availPhys,
		freeBytes:      status.availPhys,
	}, nil
}
