package util

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/disk"
)

/**
 * GetDiskInfo
 * 디스크 정보를 가져옴
 *
 * @param void
 * @return map[string]interface{}
 * @return error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetDiskInfo() (map[string]interface{}, error) {
	diskUsageDict := make(map[string]interface{})

	// 전체 디스크 사용량 정보 가져오기
	usage, err := disk.Usage("/")
	if err != nil {
		log.Error(fmt.Errorf("Error fetching disk usage: %v", err))
		return nil, err
	}

	diskUsageDict["total"] = usage.Total / 1024 / 1024 / 1024
	diskUsageDict["free"] = usage.Free / 1024 / 1024 / 1024
	diskUsageDict["used"] = usage.Used / 1024 / 1024 / 1024

	return diskUsageDict, nil
}

/**
 * GetDiskUsage
 * 디스크 사용량을 가져옴
 *
 * @param void
 * @return float64
 * @return error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetDiskUsage() (float64, error) {

	// 전체 디스크 사용량 정보 가져오기
	usage, err := disk.Usage("/")
	if err != nil {
		log.Error(fmt.Errorf("Error fetching disk usage: %v", err))
		return 0, err
	}

	return usage.UsedPercent, nil
}
