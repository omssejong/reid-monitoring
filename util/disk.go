package util

import (
	"fmt"
	"syscall"
)

func GetDiskInfo() (map[string]interface{}, error) {
	diskUsageDict := make(map[string]interface{})

	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		log.Error(fmt.Errorf("Error fetching disk usage: %v", err))
		return nil, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	diskUsageDict["total"] = total / 1024 / 1024 / 1024
	diskUsageDict["free"] = free / 1024 / 1024 / 1024
	diskUsageDict["used"] = used / 1024 / 1024 / 1024

	return diskUsageDict, nil
}

func GetDiskUsage() (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		log.Error(fmt.Errorf("Error fetching disk usage: %v", err))
		return 0, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	if total == 0 {
		return 0, nil
	}

	return float64(used) / float64(total) * 100, nil
}
