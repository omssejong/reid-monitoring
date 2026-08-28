//go:build windows

package util

import (
	"os"

	"golang.org/x/sys/windows"
)

// diskMeasurePath 디스크 사용률/용량 측정 기준 경로.
// 윈도우에서는 OS가 설치된 시스템 드라이브 루트를 기준으로 한다.
var diskMeasurePath = systemDriveRoot()

// systemDriveRoot %SystemDrive% 환경변수로 시스템 드라이브 루트를 구한다.
// 값이 없으면 C:\ 로 대체한다.
func systemDriveRoot() string {
	if drive := os.Getenv("SystemDrive"); drive != "" {
		return drive + `\`
	}
	return `C:\`
}

// diskStats GetDiskFreeSpaceExW 로 파티션 용량을 조회한다.
// avail 은 현재 프로세스 사용자가 쓸 수 있는 용량(디스크 쿼터 반영)이고,
// used 는 total - 전체여유공간 이므로 리눅스 statfs 경로와 같은 의미를 갖는다.
func diskStats(path string) (total uint64, avail uint64, used uint64, err error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, 0, err
	}

	var freeToCaller, totalBytes, totalFree uint64
	if err = windows.GetDiskFreeSpaceEx(pathPtr, &freeToCaller, &totalBytes, &totalFree); err != nil {
		return 0, 0, 0, err
	}

	return totalBytes, freeToCaller, totalBytes - totalFree, nil
}
