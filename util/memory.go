package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

func readMemInfo() (memInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return memInfo{}, err
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}
		values[key] = val * 1024
	}
	if err := scanner.Err(); err != nil {
		return memInfo{}, err
	}

	total := values["MemTotal"]
	if total == 0 {
		return memInfo{}, fmt.Errorf("MemTotal not found")
	}

	available := values["MemAvailable"]
	if available == 0 {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}

	return memInfo{
		totalBytes:     total,
		availableBytes: available,
		freeBytes:      values["MemFree"],
	}, nil
}

func bytesToGB(v uint64) float64 {
	return float64(v) / 1024 / 1024 / 1024
}
