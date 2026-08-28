//go:build linux

package util

import (
	"fmt"
	"sync"

	"github.com/mindprince/gonvml"
)

var (
	nvmlMu    sync.Mutex
	nvmlReady bool
)

// nvmlEnsure NVML을 1회만 초기화하고 이후 재사용한다.
// 매 tick Initialize/Shutdown 반복을 제거한다. 초기화 실패 시(드라이버 미준비 등)
// nvmlReady를 false로 두어 다음 호출에서 재시도한다.
// 장수 프로세스이므로 Shutdown은 호출하지 않는다(프로세스 종료 시 자동 정리).
func nvmlEnsure() error {
	nvmlMu.Lock()
	defer nvmlMu.Unlock()
	if nvmlReady {
		return nil
	}
	if err := gonvml.Initialize(); err != nil {
		return err
	}
	nvmlReady = true
	return nil
}

func nvmlDeviceCount() (uint, error) {
	return gonvml.DeviceCount()
}

func nvmlDeviceName(idx uint) (string, error) {
	device, err := gonvml.DeviceHandleByIndex(idx)
	if err != nil {
		log.Error(fmt.Errorf("getting device handle by index: %v", err))
		return "", err
	}
	return device.Name()
}

func nvmlDeviceUUID(idx uint) (string, error) {
	device, err := gonvml.DeviceHandleByIndex(idx)
	if err != nil {
		log.Error(fmt.Errorf("getting device handle by index: %v", err))
		return "", err
	}
	return device.UUID()
}

// nvmlDeviceStats 개별 항목 조회가 실패해도 나머지 값은 그대로 채운다(기존 동작 유지).
// 디바이스 핸들 자체를 얻지 못할 때만 error를 반환한다.
func nvmlDeviceStats(idx uint) (gpuStats, error) {
	device, err := gonvml.DeviceHandleByIndex(idx)
	if err != nil {
		return gpuStats{}, fmt.Errorf("getting device handle by index: %v", err)
	}

	stats := gpuStats{}

	temperature, err := device.Temperature()
	recordGPUMetric("temperature", err)
	stats.temperature = temperature

	memoryTotal, memoryUsed, err := device.MemoryInfo()
	recordGPUMetric("memory info", err)
	stats.memoryTotal = memoryTotal
	stats.memoryUsed = memoryUsed

	gpuUtilization, memoryUtilization, err := device.UtilizationRates()
	recordGPUMetric("utilization rates", err)
	stats.gpuUtilization = gpuUtilization
	stats.memoryUtilization = memoryUtilization

	return stats, nil
}
