package util

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// buildSnapshot cpuUsage(이미 계산된 값)와 직전 network 사용량으로 스냅샷 1개 생성
func buildSnapshot(cpuUsage float64, before NetworkUsage) (map[string]any, NetworkUsage, error) {
	usage := NetworkUsage{}
	temp := make(map[string]any)
	temp["monitorVersion"] = "2.0"
	upTime, err := getServerAliveTime()
	if err != nil {
		return temp, usage, err
	}
	temp["upTime"] = upTime
	serviceStatus, err := GetServiceStatus()
	if err != nil {
		return temp, usage, err
	}
	temp["serviceStatus"] = serviceStatus
	temp["cpu"] = fmt.Sprintf("%0.2f%%", cpuUsage)
	gpusUsage, err := GetGPUUsage()
	if err != nil {
		return temp, usage, err
	}
	gpuResponse := make([]map[string]uint, len(gpusUsage))
	i := 0
	for k, v := range gpusUsage {
		gTemp := make(map[string]uint)
		id, err := strconv.ParseUint(k, 10, 64)
		if err != nil {
			return temp, usage, err
		}
		gTemp["id"] = uint(id)
		gTemp["use"] = v.(map[string]any)["gpu_utilization"].(uint)
		gpuResponse[i] = gTemp
		i++
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
	cmd := exec.Command("cat", "/proc/uptime")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	raw, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	serverRunTime := strings.Split(string(raw), " ")[0]
	convServerRunTime, err := strconv.ParseFloat(serverRunTime, 32)
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	du := time.Duration(convServerRunTime * float64(time.Second))
	days := int(du.Hours()) / 24
	hours := int(du.Hours()) % 24
	minutes := int(du.Minutes()) % 60
	return fmt.Sprintf("%d-%d:%d", days, hours, minutes), nil
}

func calNetworkBandwidth(packet uint64) string {
	bandwidthUnit := []string{"bps", "Kbps", "Mbps", "Gbps"}
	idx := 0
	tmp := float64(packet)
	for tmp >= 1024 && idx < 3 {
		tmp /= 1024
		idx++
	}
	return fmt.Sprintf("%0.2f%s", tmp, bandwidthUnit[idx])
}
