package util

import (
	"fmt"
)

// diskStats 는 플랫폼별 파일로 구현한다.
//   - disk_linux.go   : syscall.Statfs
//   - disk_windows.go : GetDiskFreeSpaceExW
//
// 반환값(바이트 단위)은 df 의미를 따른다.
//
//	total : 파티션 전체 용량
//	avail : 사용 가능 용량 (예약 블록 제외)
//	used  : 실제 사용 중인 용량 (예약 블록 포함, total - free)
//
// diskMeasurePath 는 사용률/용량 측정 기준 경로이며 플랫폼별 파일에서 정의한다.

func GetDiskInfo() (map[string]interface{}, error) {
	diskUsageDict := make(map[string]interface{})

	total, avail, used, err := diskStats(diskMeasurePath)
	if err != nil {
		log.Error(fmt.Errorf("Error fetching disk usage: %v", err))
		return nil, err
	}

	diskUsageDict["total"] = total / 1024 / 1024 / 1024
	diskUsageDict["free"] = avail / 1024 / 1024 / 1024
	diskUsageDict["used"] = used / 1024 / 1024 / 1024

	return diskUsageDict, nil
}

func GetDiskUsage() (float64, error) {
	_, avail, used, err := diskStats(diskMeasurePath)
	if err != nil {
		log.Error(fmt.Errorf("Error fetching disk usage: %v", err))
		return 0, err
	}

	denom := used + avail
	if denom == 0 {
		return 0, nil
	}

	return float64(used) / float64(denom) * 100, nil
}

// diskTotalFree 지정한 경로가 속한 파티션의 total/free 바이트 반환 (df 의미).
// total = 전체 용량, free = 사용 가능 용량
func diskTotalFree(path string) (total uint64, free uint64, err error) {
	total, free, _, err = diskStats(path)
	if err != nil {
		return 0, 0, err
	}
	return total, free, nil
}
