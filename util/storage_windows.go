//go:build windows

package util

import "path/filepath"

// defaultStorageRootDir 파일 저장소 기본 경로 (config 미설정 시 사용).
// 리눅스의 /opt/oms/omeye/omeye-hss/filestorage 에 대응하는 시스템 드라이브 경로.
// 실제 운영에서는 config의 storageRootDir 로 지정하는 것을 권장한다.
var defaultStorageRootDir = filepath.Join(systemDriveRoot(), "oms", "omeye", "omeye-hss", "filestorage")
