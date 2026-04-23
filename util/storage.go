package util

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

const (
	bytesInGiB = 1024 * 1024 * 1024
)

// 기본값 (config 미설정 시 사용)
var (
	defaultStorageRootDir       = "/opt/oms/omeye/filestorage"
	defaultStorageProtectedDirs = []string{"video", "live", "de_identity", "target"}
	defaultStorageReidResultDir = "reid-result"
)

// ServerStorageInfo 디스크 상태 정보 (단위: 이진 GiB, 소수점 2자리)
type ServerStorageInfo struct {
	Total        float64 `json:"total"`
	Used         float64 `json:"used"`
	Avail        float64 `json:"avail"`
	UsedPercent  float64 `json:"usedPercent"`
	AvailPercent float64 `json:"availPercent"`
}

// candidate 삭제 후보 항목
type candidate struct {
	path     string    // 절대 경로
	mtime    time.Time // 정렬 기준
	size     int64     // 삭제 시 누적 계산 용도
	isReidID bool      // true면 RemoveAll
}

// storageConfig 현재 설정값 (없으면 기본값).
// excluded 는 전체 경로 리스트이며 해당 디렉토리와 하위 전체를 삭제 대상에서 제외한다.
func storageConfig() (root string, protected []string, reidResult string, excluded []string) {
	root = configs.SC.Setting.StorageRootDir
	if root == "" {
		root = defaultStorageRootDir
	}
	root = filepath.Clean(root)

	protected = configs.SC.Setting.StorageProtectedDirs
	if len(protected) == 0 {
		protected = defaultStorageProtectedDirs
	}

	reidResult = configs.SC.Setting.StorageReidResultDir
	if reidResult == "" {
		reidResult = defaultStorageReidResultDir
	}

	// excluded 경로 정규화 (전체 경로, 임의 깊이)
	for _, p := range configs.SC.Setting.StorageExcludedDirs {
		if p == "" {
			continue
		}
		excluded = append(excluded, filepath.Clean(p))
	}
	return
}

// diskMeasurePath 디스크 사용률/용량 측정 기준 경로.
// 파일 삭제 대상(storageRootDir)과는 별개로, 시스템 루트 파티션 기준으로 판정한다.
const diskMeasurePath = "/"

// GetServerStorageInfo 루트 파티션의 디스크 상태 반환 (GiB 단위).
// 측정 대상 파티션은 "/" 이고, 실제 파일 삭제 대상은 storageRootDir 이므로
// 두 경로가 동일 파티션에 있을 때 의도대로 동작한다.
func GetServerStorageInfo() (ServerStorageInfo, error) {
	total, free, err := diskTotalFree(diskMeasurePath)
	if err != nil {
		return ServerStorageInfo{}, err
	}
	used := total - free
	info := ServerStorageInfo{
		Total:        bytesToGiB(total),
		Used:         bytesToGiB(used),
		Avail:        bytesToGiB(free),
		UsedPercent:  percent(used, total),
		AvailPercent: percent(free, total),
	}
	return info, nil
}

// GetRootDiskUsagePercent 루트 파티션 사용률 (0~100).
func GetRootDiskUsagePercent() (float64, error) {
	total, free, err := diskTotalFree(diskMeasurePath)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, fmt.Errorf("disk total is zero")
	}
	return percent(total-free, total), nil
}

// bytesToGiB 바이트를 이진 GiB로 변환 (소수점 2자리 반올림)
func bytesToGiB(b uint64) float64 {
	return roundTo(float64(b)/bytesInGiB, 2)
}

// percent used/total 을 % 로 반환 (소수점 2자리 반올림)
func percent(num, denom uint64) float64 {
	if denom == 0 {
		return 0
	}
	return roundTo(float64(num)/float64(denom)*100, 2)
}

// roundTo n 자리 반올림
func roundTo(v float64, n int) float64 {
	pow := math.Pow(10, float64(n))
	return math.Round(v*pow) / pow
}

// collectCandidates root 아래 삭제 가능한 후보 수집.
// - protectedNames 하위의 디렉토리 자체는 후보에서 제외하되 내부 파일/하위 디렉토리는 개별 수집
// - reidResultName 하위는 {id} 단위로 묶어서 한 항목으로 수집 (하위는 walk 하지 않음)
// - excludedPaths 에 매칭되는 디렉토리는 디렉토리 자체와 하위 전체가 완전 제외됨 (임의 깊이)
func collectCandidates(root string, protectedNames []string, reidResultName string, excludedPaths []string) ([]candidate, error) {
	protected := make(map[string]struct{}, len(protectedNames))
	for _, p := range protectedNames {
		protected[p] = struct{}{}
	}
	excluded := make(map[string]struct{}, len(excludedPaths))
	for _, p := range excludedPaths {
		excluded[p] = struct{}{}
	}
	reidResultPath := filepath.Join(root, reidResultName)

	var result []candidate

	// walk 실패한 경로 개별 기록만 하고 전체는 중단하지 않음
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			log.Warn(fmt.Sprintf("walk error: %s (%v)", path, walkErr))
			return nil
		}
		if path == root {
			return nil
		}

		// 완전 제외 디렉토리: 해당 경로와 하위 전체 스킵
		if info.IsDir() {
			if _, ok := excluded[path]; ok {
				log.Info(fmt.Sprintf("storage excluded dir skipped: %s", path))
				return filepath.SkipDir
			}
		}

		rel, _ := filepath.Rel(root, path)
		topLevel := splitFirst(rel)

		// reid-result/{id} 단위 처리
		if path == reidResultPath {
			// reid-result 디렉토리 자체는 스킵, 하위를 직접 수집
			entries, err := os.ReadDir(path)
			if err != nil {
				log.Warn(fmt.Sprintf("read reid-result: %v", err))
				return filepath.SkipDir
			}
			for _, ent := range entries {
				idPath := filepath.Join(path, ent.Name())
				st, err := os.Lstat(idPath)
				if err != nil {
					log.Warn(fmt.Sprintf("lstat reid %s: %v", idPath, err))
					continue
				}
				size, _ := dirSize(idPath)
				result = append(result, candidate{
					path:     idPath,
					mtime:    st.ModTime(),
					size:     size,
					isReidID: true,
				})
			}
			return filepath.SkipDir
		}

		// 보호 디렉토리 자체는 스킵 (내부는 walk 계속)
		if _, ok := protected[topLevel]; ok && rel == topLevel {
			return nil
		}

		if info.IsDir() {
			return nil // 일반 디렉토리는 walk 계속, 자체는 후보 아님
		}

		// 심볼릭 링크는 스킵
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		result = append(result, candidate{
			path:     path,
			mtime:    info.ModTime(),
			size:     info.Size(),
			isReidID: false,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// splitFirst 상대 경로의 첫 세그먼트 반환
func splitFirst(rel string) string {
	for i := 0; i < len(rel); i++ {
		if rel[i] == os.PathSeparator || rel[i] == '/' {
			return rel[:i]
		}
	}
	return rel
}

// dirSize 디렉토리 총 바이트 크기
func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 접근 불가 항목은 0으로
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// removeCandidate 후보 하나 삭제 (파일이면 Remove, reid-result/{id}면 RemoveAll)
func removeCandidate(c candidate) error {
	if c.isReidID {
		return os.RemoveAll(c.path)
	}
	return os.Remove(c.path)
}
