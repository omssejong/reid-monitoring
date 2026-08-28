package util

import (
	"fmt"
)

// memInfo 물리 메모리 정보 (바이트 단위).
// 수집(readMemInfo)은 플랫폼별 파일에서 구현한다.
//   - memory_linux.go   : /proc/meminfo
//   - memory_windows.go : GlobalMemoryStatusEx
type memInfo struct {
	totalBytes     uint64
	availableBytes uint64
	freeBytes      uint64
}

func GetTotalMemorySize() (float64, error) {
	info, err := readMemInfo()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	return bytesToGB(info.totalBytes), nil
}

func GetUsedMemorySize() (float64, error) {
	info, err := readMemInfo()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	used := info.totalBytes - info.availableBytes
	return bytesToGB(used), nil
}

func GetFreeMemorySize() (float64, error) {
	info, err := readMemInfo()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	return bytesToGB(info.freeBytes), nil
}

func GetMemoryUsage() (float64, error) {
	info, err := readMemInfo()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	if info.totalBytes == 0 {
		return 0, nil
	}
	used := info.totalBytes - info.availableBytes
	return float64(used) / float64(info.totalBytes) * 100, nil
}

func bytesToGB(v uint64) float64 {
	return float64(v) / 1024 / 1024 / 1024
}
