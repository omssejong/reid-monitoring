package util

import (
	"fmt"
	"github.com/mindprince/gonvml"
)

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

	// NVML 초기화
	err := gonvml.Initialize()
	if err != nil {
		log.Error(fmt.Errorf("initializing NVML: %v", err))
		return nil, err
	}
	defer gonvml.Shutdown()

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

	// NVML 초기화
	err := gonvml.Initialize()
	if err != nil {
		log.Error(fmt.Errorf("initializing NVML: %v", err))
		return nil, err
	}
	defer gonvml.Shutdown()

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

	// NVML 초기화
	err := gonvml.Initialize()
	if err != nil {
		log.Error(fmt.Errorf("initializing NVML: %v", err))
		return nil, err
	}
	defer gonvml.Shutdown()

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
