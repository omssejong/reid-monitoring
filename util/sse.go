package util

import (
	"fmt"
	"strconv"
)

// buildSnapshot cpuUsage(이미 계산된 값)·직전 network 사용량·캐시된 서비스 상태로 스냅샷 1개 생성.
// serviceStatus는 느린 루프(collector.runServices)가 갱신한 캐시값을 그대로 넣는다.
func buildSnapshot(cpuUsage float64, before NetworkUsage, serviceStatus []map[string]interface{}) (map[string]any, NetworkUsage, error) {
	usage := NetworkUsage{}
	temp := make(map[string]any)
	temp["monitorVersion"] = "2.0"
	upTime, err := getServerAliveTime()
	if err != nil {
		return temp, usage, err
	}
	temp["upTime"] = upTime
	if serviceStatus == nil {
		serviceStatus = []map[string]interface{}{}
	}
	temp["serviceStatus"] = serviceStatus
	temp["cpu"] = fmt.Sprintf("%0.2f%%", cpuUsage)
	// GPU는 선택 지표다. NVIDIA GPU/드라이버가 없는 장비에서도
	// CPU/메모리/디스크/네트워크 지표 수집은 계속되어야 하므로 빈 배열로 처리한다.
	gpuResponse := make([]map[string]uint, 0)
	if gpusUsage, gpuErr := GetGPUUsage(); gpuErr == nil {
		for k, v := range gpusUsage {
			id, err := strconv.ParseUint(k, 10, 64)
			if err != nil {
				return temp, usage, err
			}
			gpuResponse = append(gpuResponse, map[string]uint{
				"id":  uint(id),
				"use": v.(map[string]any)["gpu_utilization"].(uint),
			})
		}
	}
	temp["gpu"] = gpuResponse
	totalMemoryUsage, err := GetTotalMemorySize()
	if err != nil {
		return temp, usage, err
	}
	usageMemorySize, err := GetUsedMemorySize()
	if err != nil {
		return temp, usage, err
	}
	temp["memory"] = map[string]string{
		"total": fmt.Sprintf("%0.2fGB", totalMemoryUsage),
		"used":  fmt.Sprintf("%0.2fGB", usageMemorySize),
	}
	diskUsagePercent, err := GetDiskUsage()
	if err != nil {
		return temp, usage, err
	}
	temp["disk"] = fmt.Sprintf("%0.2f%%", diskUsagePercent)
	if store := GetThresholdStore(); store != nil {
		diskThreshold := store.GetDisk()
		temp["storageThreshold"] = diskThreshold.Warning
	}
	usage, err = GetNetworkUsage(configs.SC.Setting.NetworkName)
	if err != nil {
		return temp, usage, err
	}
	total := (usage.bytesSent + usage.bytesRecv) - (before.bytesSent + before.bytesRecv)
	uplink := usage.bytesSent - before.bytesSent
	downlink := usage.bytesRecv - before.bytesRecv
	temp["networkBandwidth"] = map[string]string{
		"total":    calNetworkBandwidth(total),
		"uplink":   calNetworkBandwidth(uplink),
		"downlink": calNetworkBandwidth(downlink),
	}
	return temp, usage, nil
}

func getServerAliveTime() (string, error) {
	// 부팅 후 경과 시간 조회는 플랫폼별 구현 (uptime_linux.go / uptime_windows.go)
	du, err := systemUptime()
	if err != nil {
		return "", err
	}
	days := int(du.Hours()) / 24
	hours := int(du.Hours()) % 24
	minutes := int(du.Minutes()) % 60
	return fmt.Sprintf("%d-%d:%d", days, hours, minutes), nil
}

func calNetworkBandwidth(packet uint64) string {
	bandwidthUnit := []string{"Bit", "KB", "MB", "GB"}
	idx := 0
	tmp := float64(packet)
	for tmp >= 1024 && idx < 3 {
		tmp /= 1024
		idx++
	}
	return fmt.Sprintf("%0.2f%s", tmp, bandwidthUnit[idx])
}
