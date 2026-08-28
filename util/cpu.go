package util

import (
	"fmt"
	"time"
)

// cpuStat CPU 누적 시간 스냅샷.
// 단위는 플랫폼마다 다르지만(리눅스 USER_HZ, 윈도우 100ns) 두 스냅샷의 차분만 쓰므로 무관하다.
//
// 스냅샷 수집(readCPUStat)과 CPU 정보 조회는 플랫폼별 파일에서 구현한다.
//   - cpu_linux.go   : /proc/cpuinfo, /proc/stat
//   - cpu_windows.go : GetSystemTimes, GetLogicalProcessorInformationEx, 레지스트리
type cpuStat struct {
	total uint64
	idle  uint64
}

// calcCPUUsage 두 CPU 시간 스냅샷 차이로 사용률(%) 계산 (sleep 없음)
func calcCPUUsage(first, second cpuStat) float64 {
	totalDelta := second.total - first.total
	idleDelta := second.idle - first.idle
	if totalDelta == 0 {
		return 0
	}

	used := float64(totalDelta-idleDelta) / float64(totalDelta) * 100
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used
}

func GetCPUUsage() (float64, error) {
	first, err := readCPUStat()
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU usage: %v", err))
		return 0, err
	}

	time.Sleep(1 * time.Second)

	second, err := readCPUStat()
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU usage: %v", err))
		return 0, err
	}

	return calcCPUUsage(first, second), nil
}

func GetCPUSocket(thread int) (int, error) {
	if thread <= 0 {
		thread = 1
	}

	socketSet, err := readPhysicalSocketIDs()
	if err != nil {
		return 0, err
	}
	if len(socketSet) > 0 {
		return len(socketSet), nil
	}

	cores, err := GetCPUCores()
	if err != nil {
		return 0, err
	}
	sockets := cores / thread
	if sockets <= 0 {
		sockets = 1
	}
	return sockets, nil
}
