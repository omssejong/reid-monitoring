//go:build windows

package util

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// NVML 상수 (nvml.h)
const (
	nvmlSuccess        = 0
	nvmlTemperatureGPU = 0 // NVML_TEMPERATURE_GPU
	nvmlBufferSize     = 256
)

// nvmlDevice nvmlDevice_t — 포인터 크기의 불투명 핸들.
type nvmlDevice uintptr

// nvmlMemory nvmlMemory_t { total, free, used } (bytes)
type nvmlMemory struct {
	total uint64
	free  uint64
	used  uint64
}

// nvmlUtilization nvmlUtilization_t { gpu, memory } (%)
type nvmlUtilization struct {
	gpu    uint32
	memory uint32
}

var (
	nvmlMu    sync.Mutex
	nvmlReady bool
	nvmlDLL   *windows.LazyDLL

	procNvmlInit                    *windows.LazyProc
	procNvmlDeviceGetCount          *windows.LazyProc
	procNvmlDeviceGetHandleByIndex  *windows.LazyProc
	procNvmlDeviceGetName           *windows.LazyProc
	procNvmlDeviceGetUUID           *windows.LazyProc
	procNvmlDeviceGetTemperature    *windows.LazyProc
	procNvmlDeviceGetMemoryInfo     *windows.LazyProc
	procNvmlDeviceGetUtilizationRat *windows.LazyProc
)

// nvmlEnsure NVML을 1회만 초기화하고 이후 재사용한다.
// 초기화 실패 시(드라이버 미설치/미준비 등) nvmlReady를 false로 두어 다음 호출에서 재시도한다.
// 장수 프로세스이므로 nvmlShutdown은 호출하지 않는다(프로세스 종료 시 자동 정리).
func nvmlEnsure() error {
	nvmlMu.Lock()
	defer nvmlMu.Unlock()
	if nvmlReady {
		return nil
	}

	if err := loadNVML(); err != nil {
		return err
	}
	if r, _, _ := procNvmlInit.Call(); r != nvmlSuccess {
		return nvmlError("nvmlInit_v2", r)
	}

	nvmlReady = true
	return nil
}

// loadNVML nvml.dll을 찾아 로드하고 필요한 심볼을 바인딩한다.
// 최신 드라이버는 System32에 설치하지만, 구버전은 NVSMI 디렉토리에만 두는 경우가 있다.
func loadNVML() error {
	if nvmlDLL != nil {
		return nil
	}

	candidates := []*windows.LazyDLL{windows.NewLazySystemDLL("nvml.dll")}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		candidates = append(candidates,
			windows.NewLazyDLL(filepath.Join(programFiles, "NVIDIA Corporation", "NVSMI", "nvml.dll")))
	}

	var lastErr error
	for _, dll := range candidates {
		if err := dll.Load(); err != nil {
			lastErr = err
			continue
		}

		nvmlDLL = dll
		procNvmlInit = dll.NewProc("nvmlInit_v2")
		procNvmlDeviceGetCount = dll.NewProc("nvmlDeviceGetCount_v2")
		procNvmlDeviceGetHandleByIndex = dll.NewProc("nvmlDeviceGetHandleByIndex_v2")
		procNvmlDeviceGetName = dll.NewProc("nvmlDeviceGetName")
		procNvmlDeviceGetUUID = dll.NewProc("nvmlDeviceGetUUID")
		procNvmlDeviceGetTemperature = dll.NewProc("nvmlDeviceGetTemperature")
		procNvmlDeviceGetMemoryInfo = dll.NewProc("nvmlDeviceGetMemoryInfo")
		procNvmlDeviceGetUtilizationRat = dll.NewProc("nvmlDeviceGetUtilizationRates")
		return nil
	}

	return fmt.Errorf("nvml.dll 을 로드할 수 없습니다 (NVIDIA 드라이버 미설치?): %v", lastErr)
}

func nvmlDeviceCount() (uint, error) {
	var count uint32
	r, _, _ := procNvmlDeviceGetCount.Call(uintptr(unsafe.Pointer(&count)))
	if r != nvmlSuccess {
		return 0, nvmlError("nvmlDeviceGetCount_v2", r)
	}
	return uint(count), nil
}

func nvmlDeviceName(idx uint) (string, error) {
	device, err := nvmlDeviceHandle(idx)
	if err != nil {
		log.Error(fmt.Errorf("getting device handle by index: %v", err))
		return "", err
	}
	return nvmlDeviceString(procNvmlDeviceGetName, "nvmlDeviceGetName", device)
}

func nvmlDeviceUUID(idx uint) (string, error) {
	device, err := nvmlDeviceHandle(idx)
	if err != nil {
		log.Error(fmt.Errorf("getting device handle by index: %v", err))
		return "", err
	}
	return nvmlDeviceString(procNvmlDeviceGetUUID, "nvmlDeviceGetUUID", device)
}

// nvmlDeviceStats 개별 항목 조회가 실패해도 나머지 값은 그대로 채운다(리눅스판과 동일).
// 디바이스 핸들 자체를 얻지 못할 때만 error를 반환한다.
func nvmlDeviceStats(idx uint) (gpuStats, error) {
	device, err := nvmlDeviceHandle(idx)
	if err != nil {
		return gpuStats{}, fmt.Errorf("getting device handle by index: %v", err)
	}

	stats := gpuStats{}

	var temperature uint32
	r, _, _ := procNvmlDeviceGetTemperature.Call(uintptr(device), uintptr(nvmlTemperatureGPU), uintptr(unsafe.Pointer(&temperature)))
	recordGPUMetric("temperature", nvmlErrorIfFailed("nvmlDeviceGetTemperature", r))
	stats.temperature = uint(temperature)

	var memory nvmlMemory
	r, _, _ = procNvmlDeviceGetMemoryInfo.Call(uintptr(device), uintptr(unsafe.Pointer(&memory)))
	recordGPUMetric("memory info", nvmlErrorIfFailed("nvmlDeviceGetMemoryInfo", r))
	stats.memoryTotal = memory.total
	stats.memoryUsed = memory.used

	var utilization nvmlUtilization
	r, _, _ = procNvmlDeviceGetUtilizationRat.Call(uintptr(device), uintptr(unsafe.Pointer(&utilization)))
	recordGPUMetric("utilization rates", nvmlErrorIfFailed("nvmlDeviceGetUtilizationRates", r))
	stats.gpuUtilization = uint(utilization.gpu)
	stats.memoryUtilization = uint(utilization.memory)

	return stats, nil
}

func nvmlDeviceHandle(idx uint) (nvmlDevice, error) {
	var device nvmlDevice
	r, _, _ := procNvmlDeviceGetHandleByIndex.Call(uintptr(uint32(idx)), uintptr(unsafe.Pointer(&device)))
	if r != nvmlSuccess {
		return 0, nvmlError("nvmlDeviceGetHandleByIndex_v2", r)
	}
	return device, nil
}

// nvmlDeviceString (device, char *buf, unsigned int len) 형태의 NVML 호출을 감싼다.
func nvmlDeviceString(proc *windows.LazyProc, name string, device nvmlDevice) (string, error) {
	buf := make([]byte, nvmlBufferSize)
	r, _, _ := proc.Call(uintptr(device), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r != nvmlSuccess {
		return "", nvmlError(name, r)
	}
	if end := bytes.IndexByte(buf, 0); end >= 0 {
		buf = buf[:end]
	}
	return string(buf), nil
}

// nvmlErrorNames NVML 반환 코드 → 이름 (nvml.h nvmlReturn_t).
// nvmlErrorString은 DLL 내부 문자열 포인터를 uintptr로 돌려주는데,
// 이를 다시 포인터로 되돌리는 것은 go vet이 경고하는 패턴이라 코드 표를 직접 둔다.
var nvmlErrorNames = map[uintptr]string{
	1:   "NVML_ERROR_UNINITIALIZED",
	2:   "NVML_ERROR_INVALID_ARGUMENT",
	3:   "NVML_ERROR_NOT_SUPPORTED",
	4:   "NVML_ERROR_NO_PERMISSION",
	5:   "NVML_ERROR_ALREADY_INITIALIZED",
	6:   "NVML_ERROR_NOT_FOUND",
	7:   "NVML_ERROR_INSUFFICIENT_SIZE",
	8:   "NVML_ERROR_INSUFFICIENT_POWER",
	9:   "NVML_ERROR_DRIVER_NOT_LOADED",
	10:  "NVML_ERROR_TIMEOUT",
	11:  "NVML_ERROR_IRQ_ISSUE",
	12:  "NVML_ERROR_LIBRARY_NOT_FOUND",
	13:  "NVML_ERROR_FUNCTION_NOT_FOUND",
	14:  "NVML_ERROR_CORRUPTED_INFOROM",
	15:  "NVML_ERROR_GPU_IS_LOST",
	16:  "NVML_ERROR_RESET_REQUIRED",
	17:  "NVML_ERROR_OPERATING_SYSTEM",
	18:  "NVML_ERROR_LIB_RM_VERSION_MISMATCH",
	19:  "NVML_ERROR_IN_USE",
	20:  "NVML_ERROR_MEMORY",
	21:  "NVML_ERROR_NO_DATA",
	22:  "NVML_ERROR_VGPU_ECC_NOT_SUPPORTED",
	23:  "NVML_ERROR_INSUFFICIENT_RESOURCES",
	999: "NVML_ERROR_UNKNOWN",
}

// nvmlError NVML 반환 코드를 error로 만든다.
func nvmlError(name string, code uintptr) error {
	if reason, ok := nvmlErrorNames[code]; ok {
		return fmt.Errorf("%s failed: %s (nvml code %d)", name, reason, code)
	}
	return fmt.Errorf("%s failed (nvml code %d)", name, code)
}

// nvmlErrorIfFailed 성공(NVML_SUCCESS)이면 nil, 아니면 error를 반환한다.
func nvmlErrorIfFailed(name string, code uintptr) error {
	if code == nvmlSuccess {
		return nil
	}
	return nvmlError(name, code)
}
