package util

import (
	"fmt"
	"sync"
	"time"
)

// NVML 접근은 플랫폼별 파일에서 구현한다.
//   - gpu_linux.go   : gonvml (libnvidia-ml.so 를 dlopen)
//   - gpu_windows.go : nvml.dll 직접 바인딩
//
// 구현 대상
//   nvmlEnsure() error                          NVML 1회 초기화 (실패 시 다음 호출에서 재시도)
//   nvmlDeviceCount() (uint, error)             GPU 개수
//   nvmlDeviceName(idx uint) (string, error)    GPU 이름
//   nvmlDeviceUUID(idx uint) (string, error)    GPU UUID
//   nvmlDeviceStats(idx uint) (gpuStats, error) 온도/메모리/사용률

// gpuRetryInterval NVML 초기화 실패 후 다음 재시도까지의 대기 시간.
// 1초 주기 수집 루프가 매 tick LoadLibrary/Initialize를 반복하며 로그를 도배하는 것을 막는다.
// 나중에 드라이버가 설치되면 이 간격 뒤에 자동으로 다시 잡힌다.
const gpuRetryInterval = 5 * time.Minute

var (
	gpuMu        sync.Mutex
	gpuNextRetry time.Time
	gpuWarnOnce  sync.Once
)

// ensureGPU NVML을 초기화하되, 실패하면 gpuRetryInterval 동안 재시도를 건너뛴다.
// GPU가 없는 장비에서도 나머지 지표 수집이 멈추지 않도록 호출자는 이 에러를 무시하고 진행한다.
func ensureGPU() error {
	gpuMu.Lock()
	defer gpuMu.Unlock()

	if !gpuNextRetry.IsZero() && time.Now().Before(gpuNextRetry) {
		return fmt.Errorf("NVML 사용 불가 (다음 재시도까지 대기 중)")
	}

	if err := nvmlEnsure(); err != nil {
		gpuNextRetry = time.Now().Add(gpuRetryInterval)
		gpuWarnOnce.Do(func() {
			log.Warn(fmt.Sprintf("NVML 초기화 실패 — GPU 지표 없이 동작합니다 (NVIDIA GPU/드라이버 미설치?): %v", err))
		})
		return err
	}

	gpuNextRetry = time.Time{}
	return nil
}

// GPU 개별 지표(온도/메모리/사용률) 조회 실패 로그 억제.
// 수집 루프가 1초 주기라 같은 실패를 매 tick 남기면 로그가 도배된다.
// 노트북(Optimus 등)에서 nvmlDeviceGetUtilizationRates 만 계속 실패하는 사례가 대표적이다.
// 실패가 이어지는 동안에는 첫 1건만 남기고, 해당 지표가 다시 성공하면 상태를 초기화해
// 나중에 재발했을 때 다시 1건을 남긴다.
var (
	gpuMetricMu     sync.Mutex
	gpuMetricFailed = make(map[string]bool)
)

// recordGPUMetric 지표별 조회 결과를 기록한다. err이 nil이면 실패 상태를 해제한다.
func recordGPUMetric(metric string, err error) {
	gpuMetricMu.Lock()
	if err == nil {
		delete(gpuMetricFailed, metric)
		gpuMetricMu.Unlock()
		return
	}
	alreadyLogged := gpuMetricFailed[metric]
	gpuMetricFailed[metric] = true
	gpuMetricMu.Unlock()

	if alreadyLogged {
		return
	}
	log.Warn(fmt.Sprintf("GPU %s 조회 실패 — 해당 값은 0으로 보고합니다 (같은 실패 반복 로그는 생략): %v", metric, err))
}

// gpuStats GPU 1개의 사용량 스냅샷.
type gpuStats struct {
	temperature       uint   // °C
	memoryTotal       uint64 // bytes
	memoryUsed        uint64 // bytes
	gpuUtilization    uint   // %
	memoryUtilization uint   // %
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
	if err := ensureGPU(); err != nil {
		return nil, err
	}

	// GPU 디바이스 수 가져오기
	count, err := nvmlDeviceCount()
	if err != nil {
		log.Error(fmt.Errorf("getting device count: %v", err))
		return nil, err
	}

	if count == 0 {
		gpuInfoDict["device_name"] = ""
		gpuInfoDict["device_count"] = count
		gpuInfoDict["device_uuid"] = ""
		return gpuInfoDict, nil
	}

	name, err := nvmlDeviceName(0)
	if err != nil {
		log.Error(fmt.Errorf("getting device name: %v", err))
		return nil, err
	}

	uid, err := nvmlDeviceUUID(0)
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
	if err := ensureGPU(); err != nil {
		return nil, err
	}

	// GPU 디바이스 수 가져오기
	count, err := nvmlDeviceCount()
	if err != nil {
		log.Error(fmt.Errorf("getting device count: %v", err))
		return nil, err
	}

	gpuInfoList := make([]string, count)
	if count == 0 {
		return gpuInfoList, nil
	}

	name, err := nvmlDeviceName(0)
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
	if err := ensureGPU(); err != nil {
		return nil, err
	}

	// GPU 디바이스 수 가져오기
	count, err := nvmlDeviceCount()
	if err != nil {
		log.Error(fmt.Errorf("getting device count: %v", err))
		return nil, err
	}

	for i := uint(0); i < count; i++ {
		stats, err := nvmlDeviceStats(i)
		if err != nil {
			log.Error(fmt.Errorf("getting device stats: %v", err))
			continue
		}

		gpuUtilItemDict := make(map[string]interface{})
		gpuUtilItemDict["temperature"] = stats.temperature                            // °C
		gpuUtilItemDict["memory_total_mb"] = float64(stats.memoryTotal / 1024 / 1024) // MB
		gpuUtilItemDict["memory_used_mb"] = float64(stats.memoryUsed / 1024 / 1024)   // MB
		gpuUtilItemDict["gpu_utilization"] = stats.gpuUtilization                     // %
		gpuUtilItemDict["memory_utilization"] = stats.memoryUtilization               // %

		key := fmt.Sprintf("%d", i)
		gpuUtilDict[key] = gpuUtilItemDict

	}

	return gpuUtilDict, nil

}
