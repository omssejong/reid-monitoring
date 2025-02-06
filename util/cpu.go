package util

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/cpu"
	"time"
)

/**
 * GetCPUModelName
 * CPU 모델명을 가져오는 함수
 *
 * @param: void
 * @return: string, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetCPUModelNameAndPhysicalThreadCount() (string, int, error) {
	cpuCnt, err := cpu.Counts(false)
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU count: %v", err))
		return "", 0, err
	}
	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU Info: %v", err))
		return "", 0, err
	}

	// 선별관제 전용 모니터링
	//return fmt.Sprintf("%s, %d Core", cpuInfo[0].ModelName, cpuCnt), cpuCnt, nil
	// 고속분석 전용 모니터링
	return fmt.Sprintf("%s", cpuInfo[0].ModelName), cpuCnt, nil
}

/**
 * GetCPUUsage
 * CPU 사용량을 가져오는 함수
 *
 * @param: void
 * @return: float64, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetCPUUsage() (float64, error) {
	percentages, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU usage: %v", err))
		return 0, err
	}
	return percentages[0], nil
}

/**
 * GetCPUCores
 * CPU 코어 수를 가져오는 함수
 *
 * @param: void
 * @return: int, error
 *
 * @autor: Han Seong San
 * @since: 2024.06.19
 * @version: 1.0.0
 */
func GetCPUCores() (int, error) {

	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Error(fmt.Errorf("fetching CPU Info: %v", err))
		return 0, err
	}
	return len(cpuInfo), nil
}

/**
 * GetCpuSocket
 * CPU 소켓 수를 가져오는 함수
 *
 * @param: int
 * @return: int
 *
 * @autor: park kyeong bin
 * @since: 2025.02.06
 * @version: 1.0.1
 */
func GetCPUSocket(thread int) (int, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return 0, err
	}

	return len(cpuInfo) / thread, nil
}
