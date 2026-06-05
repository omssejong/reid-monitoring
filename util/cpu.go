package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type cpuStat struct {
	total uint64
	idle  uint64
}

func GetCPUModelNameAndPhysicalThreadCount() (string, int, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU info: %v", err))
		return "", 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	modelName := ""
	physicalCoreSet := make(map[string]struct{})
	logicalCount := 0
	var physicalID string
	var coreID string

	flushCore := func() {
		if physicalID != "" && coreID != "" {
			physicalCoreSet[physicalID+":"+coreID] = struct{}{}
		}
		physicalID = ""
		coreID = ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			flushCore()
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "model name":
			if modelName == "" {
				modelName = value
			}
		case "processor":
			logicalCount++
		case "physical id":
			physicalID = value
		case "core id":
			coreID = value
		}
	}
	flushCore()

	if err := scanner.Err(); err != nil {
		log.Error(fmt.Errorf("scanning CPU info: %v", err))
		return "", 0, err
	}

	if modelName == "" {
		modelName = "unknown"
	}

	physicalThreads := len(physicalCoreSet)
	if physicalThreads == 0 {
		physicalThreads = logicalCount
	}
	if physicalThreads == 0 {
		physicalThreads = 1
	}

	return modelName, physicalThreads, nil
}

// calcCPUUsage 두 /proc/stat 스냅샷 차이로 사용률(%) 계산 (sleep 없음)
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

func GetCPUCores() (int, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU info: %v", err))
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	logicalCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "processor") {
			logicalCount++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Error(fmt.Errorf("scanning CPU info: %v", err))
		return 0, err
	}

	if logicalCount == 0 {
		logicalCount = 1
	}
	return logicalCount, nil
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

func readCPUStat() (cpuStat, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return cpuStat{}, fmt.Errorf("invalid /proc/stat format")
		}

		var total uint64
		for i := 1; i < len(fields); i++ {
			v, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return cpuStat{}, err
			}
			total += v
		}

		idle, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			return cpuStat{}, err
		}

		return cpuStat{total: total, idle: idle}, nil
	}

	if err := scanner.Err(); err != nil {
		return cpuStat{}, err
	}
	return cpuStat{}, fmt.Errorf("cpu line not found")
}

func readPhysicalSocketIDs() (map[string]struct{}, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	socketSet := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == "physical id" {
			socketSet[strings.TrimSpace(parts[1])] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return socketSet, nil
}
