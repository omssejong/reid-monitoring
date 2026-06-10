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
func DeleteFilesAnHours(path string, retention time.Duration) error {
	if retention <= 0 {
		retention = time.Hour
	}

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

			if time.Since(f.ModTime()) > retention {
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
 * 로그 디렉터리(logPath, {rootPath}/ai/logs)를 통째로 훑어
 * 수정 시각(ModTime)이 날짜 범위 안에 드는 파일을 확장자 구분 없이 반환한다.
 *
 * @param start string  (yyyyMMdd)
 * @param end   string  (yyyyMMdd)
 *
 * @return []string
 * @return error
 *
 * @auther: Han Seong San
 * @version: 2.0.0
 * @since: 2024.06.21
 */
func FilterLogFilesByDate(start, end string) ([]string, error) {

	parseStartTime, err := time.Parse("20060102", start)
	if err != nil {
		return nil, err
	}
	parseEndTime, err := time.Parse("20060102", end)
	if err != nil {
		return nil, err
	}
	// 종료일 당일 끝(23:59:59)까지 포함
	parseEndTime = parseEndTime.Add(86399 * time.Second)

	var files []string

	// 통째로 훑을 로그 디렉터리 목록
	roots := []string{
		logPath,
		filepath.Join(rootPath, "ai", "logs"),
	}

	for _, root := range roots {
		// 존재하지 않는 디렉터리는 건너뛴다
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				log.Info("skip not-exist dir: " + root)
				continue
			}
			return nil, err
		}

		// 디렉터리 하위 전체 파일 수집 (확장자 무관)
		found, err := searchFiles(root, "")
		if err != nil {
			return nil, err
		}

		// 수정 시각이 날짜 범위 안인 파일만 선택
		for _, target := range found {
			info, err := os.Stat(target)
			if err != nil {
				continue
			}
			modTime := info.ModTime()
			if modTime.Before(parseStartTime) || modTime.After(parseEndTime) {
				continue
			}
			files = append(files, target)
		}
	}

	log.Info(fmt.Sprintf("%d files filtered", len(files)))

	return files, nil

}

// searchFiles targetPath 하위를 재귀 탐색하여 파일 경로를 반환한다.
// extName이 빈 문자열이면 모든 파일, 아니면 해당 확장자 파일만 수집한다.
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
		if extName == "" || filepath.Ext(path) == extName {
			files = append(files, path)
		}

		return err
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
