package util

//
//import (
//	"encoding/json"
//	"fmt"
//	"path"
//	"time"
//)
//
///**
// * Monitoring Worker
// * 모니터링 워커 함수
// *
// * @Author: Han Seong San
// * @Version: 1.0.0
// * @Since: 2024.06.20
// */
//func MonitoringWorker() {
//
//	ticker := time.NewTicker(120 * time.Minute)
//	defer ticker.Stop()
//
//	for {
//		select {
//		case <-ticker.C:
//			log.Info("Delete Zip File Start")
//			if err := DeleteFilesAnHours(path.Join(monitoringPath, "data")); err != nil {
//				log.Error(fmt.Errorf("failed to delete zip files: %w", err))
//				return
//			}
//		default:
//			monitorinUsageDict := make(map[string]interface{})
//			networkUsageDict := make(map[string]interface{})
//			hardwareUsageDict := make(map[string]interface{})
//
//			usageStart, _ := GetNetworkUsage(networkName)
//
//			time.Sleep(1 * time.Second)
//
//			// 하드웨어 모니터링
//			cpuUsage, _ := GetCPUUsage()                // CPU 사용량
//			gpuUsage, _ := GetGPUUsage()                // GPU 사용량
//			memoryUsage, _ := GetMemoryUsage()          // 메모리 사용량
//			usageEnd, _ := GetNetworkUsage(networkName) // 네트워크 사용량
//
//			networkUsageDict["bytes_sent"] = usageEnd.bytesSent - usageStart.bytesSent
//			networkUsageDict["bytes_recv"] = usageEnd.bytesRecv - usageStart.bytesRecv
//
//			hardwareUsageDict["cpu"] = map[string]interface{}{
//				"utilization": cpuUsage,
//			}
//			hardwareUsageDict["memory"] = map[string]interface{}{
//				"utilization": memoryUsage,
//			}
//			hardwareUsageDict["gpu"] = gpuUsage
//			hardwareUsageDict["network"] = networkUsageDict
//			monitorinUsageDict["hardware"] = hardwareUsageDict
//
//			// 서비스 모니터링
//			serviceUsageDict, _ := GetServiceStatus()
//			// log.Info(fmt.Sprintf("Service Status: %v", serviceUsageDict))
//			monitorinUsageDict["service"] = serviceUsageDict
//
//			// JSON으로 변환
//			jsonBytes, err := json.Marshal(monitorinUsageDict)
//			if err != nil {
//				log.Error(fmt.Errorf("json marshal error: %w", err))
//				return
//			}
//
//			// 채널 이름 생성 후 Redis에 Publish
//			if err := redisClient.Publish(monitoringChannelName, jsonBytes); err != nil {
//				log.Error(fmt.Errorf("redis publish error: %w", err))
//				return
//			}
//
//		}
//	}
//}
