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

// ensureNVML NVML을 1회만 초기화하고 이후 재사용한다.
// 매 tick Initialize/Shutdown 반복을 제거한다. 초기화 실패 시(드라이버 미준비 등)
// nvmlReady를 false로 두어 다음 호출에서 재시도한다.
// 장수 프로세스이므로 Shutdown은 호출하지 않는다(프로세스 종료 시 자동 정리).
func ensureNVML() error {
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

/**
 * GetGPUInfo
 * GPU 정보를 가져오는 함수
 *
 * @param: void
 * @return: map[string]interface{}, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetGPUInfo() (map[string]interface{}, error) {

	gpuInfoDict := make(map[string]interface{})

	// NVML 초기화 (1회만, 이후 재사용)
	if err := ensureNVML(); err != nil {
		log.Error(fmt.Errorf("initializing NVML: %v", err))
		return nil, err
	}

	// GPU 디바이스 수 가져오기
	count, err := gonvml.DeviceCount()
	if err != nil {
		log.Error(fmt.Errorf("getting device count: %v", err))
		return nil, err
	}

	device, err := gonvml.DeviceHandleByIndex(0)
	if err != nil {
		log.Error(fmt.Errorf("getting device handle by index: %v", err))
		return nil, err
	}

	name, err := device.Name()
	if err != nil {
		log.Error(fmt.Errorf("getting device name: %v", err))
		return nil, err
	}

	uid, err := device.UUID()
	if err != nil {
		log.Error(fmt.Errorf("getting device UUID: %v", err))
		return nil, err
	}

	gpuInfoDict["device_name"] = name
	gpuInfoDict["device_count"] = count
	gpuInfoDict["device_uuid"] = uid

	return gpuInfoDict, nil

}

func ReidGetGPUInfo() ([]string, error) {

	// NVML 초기화 (1회만, 이후 재사용)
	if err := ensureNVML(); err != nil {
		log.Error(fmt.Errorf("initializing NVML: %v", err))
		return nil, err
	}

	// GPU 디바이스 수 가져오기
	count, err := gonvml.DeviceCount()
	if err != nil {
		log.Error(fmt.Errorf("getting device count: %v", err))
		return nil, err
	}
	gpuInfoList := make([]string, count)
	device, err := gonvml.DeviceHandleByIndex(0)
	if err != nil {
		log.Error(fmt.Errorf("getting device handle by index: %v", err))
		return nil, err
	}

	name, err := device.Name()
	if err != nil {
		log.Error(fmt.Errorf("getting device name: %v", err))
		return nil, err
	}

	var i uint

	for i = 0; i < count; i++ {
		gpuInfoList[i] = fmt.Sprintf("GPU %d: %s ", i, name)
	}

	return gpuInfoList, nil

}

/**
 * GetGPUUsage
 * GPU 사용량을 가져오는 함수
 *
 * @param: void
 * @return: map[string]interface{}, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetGPUUsage() (map[string]interface{}, error) {

	gpuUtilDict := make(map[string]interface{})

	// NVML 초기화 (1회만, 이후 재사용 — 매 tick init/shutdown 제거)
	if err := ensureNVML(); err != nil {
		log.Error(fmt.Errorf("initializing NVML: %v", err))
		return nil, err
	}

	// GPU 디바이스 수 가져오기
	count, err := gonvml.DeviceCount()
	if err != nil {
		log.Error(fmt.Errorf("getting device count: %v", err))
		return nil, err
	}

	for i := uint(0); i < count; i++ {
		gpuUtilItemDict := make(map[string]interface{})

		device, err := gonvml.DeviceHandleByIndex(i)
		if err != nil {
			log.Error(fmt.Errorf("getting device handle by index: %v", err))
			continue
		}

		temperature, err := device.Temperature()
		if err != nil {
			log.Error(fmt.Errorf("getting device temperature: %v", err))
		}

		memoryTotal, memoryUsed, err := device.MemoryInfo()
		if err != nil {
			log.Error(fmt.Errorf("getting memory info: %v", err))
		}

		gpuUtilization, memoryUtilization, err := device.UtilizationRates()
		if err != nil {
			log.Error(fmt.Errorf("getting utilization rates: %v", err))
		}

		gpuUtilItemDict["temperature"] = temperature                            // °C
		gpuUtilItemDict["memory_total_mb"] = float64(memoryTotal / 1024 / 1024) // MB
		gpuUtilItemDict["memory_used_mb"] = float64(memoryUsed / 1024 / 1024)   // MB
		gpuUtilItemDict["gpu_utilization"] = gpuUtilization                     // %
		gpuUtilItemDict["memory_utilization"] = memoryUtilization               // %

		key := fmt.Sprintf("%d", i)
		gpuUtilDict[key] = gpuUtilItemDict

	}

	return gpuUtilDict, nil

}
