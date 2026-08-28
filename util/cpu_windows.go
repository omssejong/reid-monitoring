//go:build windows

package util

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// LOGICAL_PROCESSOR_RELATIONSHIP (winnt.h)
const (
	relationProcessorCore    = 0
	relationProcessorPackage = 3
)

// allProcessorGroups GetActiveProcessorCount 에 넘기는 ALL_PROCESSOR_GROUPS 값.
const allProcessorGroups = 0xFFFF

var (
	modKernel32CPU = windows.NewLazySystemDLL("kernel32.dll")

	procGetSystemTimes                   = modKernel32CPU.NewProc("GetSystemTimes")
	procGetLogicalProcessorInformationEx = modKernel32CPU.NewProc("GetLogicalProcessorInformationEx")
	procGetActiveProcessorCount          = modKernel32CPU.NewProc("GetActiveProcessorCount")
)

// GetCPUModelNameAndPhysicalThreadCount CPU 모델명과 물리 코어 수를 반환한다.
// 리눅스판이 /proc/cpuinfo 의 (physical id, core id) 조합 개수를 세는 것과 동일하게
// 윈도우에서는 RelationProcessorCore 항목 수를 물리 코어 수로 사용한다.
func GetCPUModelNameAndPhysicalThreadCount() (string, int, error) {
	modelName, err := cpuModelName()
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU info: %v", err))
		return "", 0, err
	}

	physicalThreads, err := countProcessorRelations(relationProcessorCore)
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU info: %v", err))
		return "", 0, err
	}

	if physicalThreads == 0 {
		physicalThreads, _ = GetCPUCores()
	}
	if physicalThreads == 0 {
		physicalThreads = 1
	}

	return modelName, physicalThreads, nil
}

// GetCPUCores 논리 프로세서(하이퍼스레드 포함) 개수를 반환한다.
func GetCPUCores() (int, error) {
	// GetActiveProcessorCount(ALL_PROCESSOR_GROUPS) 는 64코어 초과 시스템의
	// 프로세서 그룹까지 합산해서 반환한다.
	if r, _, _ := procGetActiveProcessorCount.Call(uintptr(allProcessorGroups)); r > 0 {
		return int(r), nil
	}

	logicalCount := runtime.NumCPU()
	if logicalCount == 0 {
		logicalCount = 1
	}
	return logicalCount, nil
}

// readCPUStat GetSystemTimes 로 시스템 전체 CPU 누적 시간을 조회한다.
// kernel 시간에는 idle 시간이 포함되어 있으므로 total = kernel + user 다.
// 단위는 100ns 이며 차분만 사용하므로 리눅스의 USER_HZ 단위와 혼용해도 문제없다.
func readCPUStat() (cpuStat, error) {
	var idle, kernel, user windows.Filetime

	r, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return cpuStat{}, fmt.Errorf("GetSystemTimes failed: %v", err)
	}

	return cpuStat{
		total: filetimeTicks(kernel) + filetimeTicks(user),
		idle:  filetimeTicks(idle),
	}, nil
}

// readPhysicalSocketIDs 물리 소켓(패키지) 식별자 집합을 반환한다.
// 윈도우 API는 소켓 ID를 노출하지 않으므로 패키지 개수만큼 0..N-1 키를 만들어
// 리눅스판(readPhysicalSocketIDs)과 동일한 시그니처를 유지한다.
func readPhysicalSocketIDs() (map[string]struct{}, error) {
	count, err := countProcessorRelations(relationProcessorPackage)
	if err != nil {
		return nil, err
	}

	socketSet := make(map[string]struct{}, count)
	for i := 0; i < count; i++ {
		socketSet[strconv.Itoa(i)] = struct{}{}
	}
	return socketSet, nil
}

// cpuModelName 레지스트리에서 CPU 모델명을 읽는다. (/proc/cpuinfo 의 "model name" 대응)
func cpuModelName() (string, error) {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return "", err
	}
	defer key.Close()

	name, _, err := key.GetStringValue("ProcessorNameString")
	if err != nil {
		return "", err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "unknown"
	}
	return name, nil
}

// countProcessorRelations GetLogicalProcessorInformationEx 결과에서
// 지정한 관계(코어/패키지) 항목 개수를 센다.
// 각 항목은 {Relationship uint32, Size uint32, ...} 로 시작하므로
// Size 만 따라가면 구조체 내부를 해석하지 않고도 안전하게 순회할 수 있다.
func countProcessorRelations(relationship uint32) (int, error) {
	var size uint32

	r, _, err := procGetLogicalProcessorInformationEx.Call(
		uintptr(relationship),
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return 0, fmt.Errorf("GetLogicalProcessorInformationEx failed: %v", err)
	}
	if size == 0 {
		return 0, nil
	}

	buf := make([]byte, size)
	r, _, err = procGetLogicalProcessorInformationEx.Call(
		uintptr(relationship),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return 0, fmt.Errorf("GetLogicalProcessorInformationEx failed: %v", err)
	}

	count := 0
	for offset := uint32(0); offset+8 <= size; {
		entrySize := *(*uint32)(unsafe.Pointer(&buf[offset+4]))
		if entrySize == 0 {
			break
		}
		count++
		offset += entrySize
	}
	return count, nil
}

// filetimeTicks FILETIME 을 100ns 단위 정수로 변환한다.
func filetimeTicks(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}
