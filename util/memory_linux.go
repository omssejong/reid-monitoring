//go:build linux

package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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
