package util

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

/**
 * DeleteFilesAnHours
 * 1시간 전 생성된 파일 삭제 함수
 *
 * @param path string
 *
 * @return error
 *
 * @auther: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.21
 */
func DeleteFilesAnHours(path string) error {

	// data 폴더 내 파일 목록을 가져온다
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// 1시간 전 생성된 파일을 삭제한다
	for _, file := range files {

		log.Info("target file: " + file.Name())
		if filepath.Ext(file.Name()) == ".zip" {
			f, err := file.Info()
			if err != nil {
				return fmt.Errorf("get file info : %s", err.Error())
			}

			if time.Since(f.ModTime()) > time.Hour {
				log.Info("delete file: " + file.Name())
				if err := os.Remove(filepath.Join(path, file.Name())); err != nil {
					return err
				}
			}
		}

	}

	return nil

}

/**
 * FilterLogFilesByDate
 * 날짜 범위 내의 로그 파일 필터링 함수
 *
 * @param start string
 * @param end string
 *
 * @return []string
 * @return error
 *
 * @auther: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.21
 */
func FilterLogFilesByDate(start, end string) ([]string, error) {

	//today := time.Now().Format("20060102")
	parseStartTime, err := time.Parse("20060102", start)
	if err != nil {
		return nil, err
	}
	parseEndTime, err := time.Parse("20060102", end)
	if err != nil {
		return nil, err
	}
	parseEndTime = parseEndTime.Add(86399 * time.Second)
	var files []string
	files = append(files, "/var/log/syslog")

	targetFiles, err := searchFiles(logPath, ".log")
	if err != nil {
		return nil, err
	}

	fmt.Println(parseStartTime, parseEndTime)
	for _, target := range targetFiles {
		if info, err := os.Stat(target); err == nil {
			lastModifyTime := info.ModTime()
			if lastModifyTime.After(parseEndTime) || lastModifyTime.Before(parseStartTime) {
				continue
			}
			files = append(files, target)
		}
	}

	if _, err := os.Stat(filepath.Join(rootPath, "hsa", "log")); err == nil {
		targetFiles, err = searchFiles(filepath.Join(rootPath, "hsa", "log"), ".log")
		for _, target := range targetFiles {
			if info, err := os.Stat(target); err == nil {
				lastModifyTime := info.ModTime()
				if lastModifyTime.After(parseEndTime) || lastModifyTime.Before(parseStartTime) {
					continue
				}
				files = append(files, target)
			}
		}
	}

	//
	//// 날짜 범위 내의 날짜 목록 가져오기
	//dates, err := getDatesInRange(start, end)
	//if err != nil {
	//	log.Error(fmt.Errorf("failed to get dates in range: %w", err))
	//	return nil, err
	//}
	//
	//log.Info(fmt.Sprintf("%s ~ %s dates in range", dates[0], dates[len(dates)-1]))
	//
	//// 로그 파일 목록 가져오기
	//targetFiles, err := searchFiles(logPath, ".log")
	//if err != nil {
	//	log.Error(fmt.Errorf("failed to search for log files: %w", err))
	//	return nil, err
	//}
	//
	//log.Info(fmt.Sprintf("%d files searched", len(targetFiles)))
	//
	//files = append(files, "/var/log/syslog")
	//
	//// 날짜 범위 내의 파일 목록 가져오기
	//for _, file := range targetFiles {
	//
	//	for _, date := range dates {
	//		if strings.Contains(file, date) {
	//			log.Info(fmt.Sprintf("files in date range: %s", file))
	//			files = append(files, file)
	//		}
	//	}
	//}
	//
	//if today == end || today == start {
	//	aiLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, aiLogName))
	//	if _, err := os.Stat(aiLogPath); err == nil {
	//		files = append(files, aiLogPath)
	//	}
	//
	//	backendLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendLogName))
	//	if _, err := os.Stat(backendLogPath); err == nil {
	//		files = append(files, backendLogPath)
	//	}
	//
	//	backendAuthLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendAuthLogName))
	//	if _, err := os.Stat(backendAuthLogPath); err == nil {
	//		files = append(files, backendAuthLogPath)
	//	}
	//
	//	backendGatewayLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendGatewayLogName))
	//	if _, err := os.Stat(backendGatewayLogPath); err == nil {
	//		files = append(files, backendGatewayLogPath)
	//	}
	//
	//	backendMainLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendMainLogName))
	//	if _, err := os.Stat(backendMainLogPath); err == nil {
	//		files = append(files, backendMainLogPath)
	//	}
	//
	//	backendAiLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendAiLogName))
	//	if _, err := os.Stat(backendAiLogPath); err == nil {
	//		files = append(files, backendAiLogPath)
	//	}
	//
	//	backendSettingsLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendSettingsLogName))
	//	if _, err := os.Stat(backendSettingsLogPath); err == nil {
	//		files = append(files, backendSettingsLogPath)
	//	}
	//
	//	backendUserLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendUserLogName))
	//	if _, err := os.Stat(backendUserLogPath); err == nil {
	//		files = append(files, backendUserLogPath)
	//	}
	//
	//	backendVmsLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, backendVmsLogName))
	//	if _, err := os.Stat(backendVmsLogPath); err == nil {
	//		files = append(files, backendVmsLogPath)
	//	}
	//
	//	frontendLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, frontendLogName))
	//	if _, err := os.Stat(frontendLogPath); err == nil {
	//		files = append(files, frontendLogPath)
	//	}
	//
	//	imageProcessingLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, imageProcessingLogName))
	//	if _, err := os.Stat(imageProcessingLogPath); err == nil {
	//		files = append(files, imageProcessingLogPath)
	//	}
	//
	//	mediaStreamingLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, mediaStreamingLogName))
	//	if _, err := os.Stat(mediaStreamingLogPath); err == nil {
	//		files = append(files, mediaStreamingLogPath)
	//	}
	//
	//	monitoringLogPath := fmt.Sprintf("%s.log", filepath.Join(logPath, monitoringLogName))
	//	if _, err := os.Stat(monitoringLogPath); err == nil {
	//		files = append(files, monitoringLogPath)
	//	}
	//
	//}

	log.Info(fmt.Sprintf("%d files filtered", len(files)))

	return files, nil

}

func searchFiles(targetPath, extName string) ([]string, error) {

	var files []string

	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		// 디렉토리는 건너뛰기
		if info.IsDir() {
			return nil
		}

		log.Info("search file path: " + path)
		if filepath.Ext(path) == extName {
			files = append(files, path)
		}

		return err
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func getDatesInRange(start, end string) ([]string, error) {
	const layout = "20060102"
	startDate, err := time.Parse(layout, start)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse(layout, end)
	if err != nil {
		return nil, err
	}

	var dates []string
	for !startDate.After(endDate) {
		dates = append(dates, startDate.Format(layout))
		startDate = startDate.AddDate(0, 0, 1)
	}

	return dates, nil
}
