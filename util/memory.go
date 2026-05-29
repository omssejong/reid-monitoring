package util

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/mem"
)

/**
 * GetTotalMemorySize
 * 전체 메모리 크기를 가져오는 함수
 *
 * @param: void
 * @return: float64, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetTotalMemorySize() (float64, error) {
	// 메모리 사용량을 가져옴
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	return float64(vmStat.Total / 1024 / 1024 / 1024), nil
}

/**
 * GetUsedMemorySize
 * 사용중인 메모리 크기를 가져오는 함수
 *
 * @param: void
 * @return: float64, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetUsedMemorySize() (float64, error) {
	// 메모리 사용량을 가져옴
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	return float64(vmStat.Used / 1024 / 1024 / 1024), nil
}

/**
 * GetFreeMemorySize
 * 사용 가능한 메모리 크기를 가져오는 함수
 *
 * @param: void
 * @return: float64, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetFreeMemorySize() (float64, error) {
	// 메모리 사용량을 가져옴
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	return float64(vmStat.Free / 1024 / 1024 / 1024), nil
}

/**
 * GetMemoryUsage
 * 메모리 사용량을 가져오는 함수
 *
 * @param: void
 * @return: float64, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetMemoryUsage() (float64, error) {
	// 메모리 사용량을 가져옴
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Error(fmt.Errorf("fetching memory usage: %v", err))
		return 0, err
	}
	return vmStat.UsedPercent, nil
}
