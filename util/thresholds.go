package util

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// DiskThreshold 디스크 사용률 임계값 설정 (warning 단일 레벨)
type DiskThreshold struct {
	Warning float64 `yaml:"warning" json:"warning"`
}

// thresholdsFile thresholds.yml 구조
type thresholdsFile struct {
	Disk DiskThreshold `yaml:"disk"`
}

// Breach 임계값 초과 정보
type Breach struct {
	Metric       string  `json:"metric"`
	CurrentValue float64 `json:"currentValue"`
	Threshold    float64 `json:"threshold"`
}

// ThresholdStore 런타임 임계값 저장소 (메모리 + 파일 영속화)
type ThresholdStore struct {
	mu   sync.RWMutex
	disk DiskThreshold
	path string
}

// defaultDiskThreshold 기본 임계값
var defaultDiskThreshold = DiskThreshold{
	Warning: 80.0,
}

// 전역 인스턴스 (main.go에서 초기화)
var thresholdStore *ThresholdStore

// InitThresholdStore 임계값 저장소 초기화. 파일이 없으면 기본값으로 생성한다.
func InitThresholdStore(path string) error {
	store := &ThresholdStore{path: path}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("thresholds 디렉토리 생성 실패: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 파일 없으면 기본값으로 생성
			store.disk = defaultDiskThreshold
			if err := store.save(); err != nil {
				return fmt.Errorf("기본 thresholds 파일 생성 실패: %w", err)
			}
			log.Info(fmt.Sprintf("thresholds 파일을 기본값으로 생성: %s", path))
			thresholdStore = store
			return nil
		}
		return fmt.Errorf("thresholds 파일 읽기 실패: %w", err)
	}

	var fileCfg thresholdsFile
	if err := yaml.Unmarshal(raw, &fileCfg); err != nil {
		return fmt.Errorf("thresholds 파일 파싱 실패: %w", err)
	}

	// 비어있는 값은 기본값으로 대체
	if fileCfg.Disk.Warning <= 0 {
		fileCfg.Disk.Warning = defaultDiskThreshold.Warning
	}

	store.disk = fileCfg.Disk
	thresholdStore = store
	log.Info(fmt.Sprintf("thresholds 로드 완료: disk warning=%.1f", store.disk.Warning))
	return nil
}

// GetThresholdStore 전역 저장소 반환
func GetThresholdStore() *ThresholdStore {
	return thresholdStore
}

// GetDisk 현재 디스크 임계값 반환
func (s *ThresholdStore) GetDisk() DiskThreshold {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disk
}

// UpdateDisk 디스크 임계값 변경 (메모리 갱신 + 원자적 파일 저장)
func (s *ThresholdStore) UpdateDisk(v DiskThreshold) error {
	if v.Warning <= 0 || v.Warning > 100 {
		return fmt.Errorf("warning 값은 0~100 사이여야 합니다: %.2f", v.Warning)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.disk = v
	return s.save()
}

// CheckDisk 현재 디스크 사용률이 warning 이상이면 Breach 반환. 아니면 nil.
func (s *ThresholdStore) CheckDisk(currentUsage float64) *Breach {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if currentUsage >= s.disk.Warning {
		return &Breach{
			Metric:       "disk",
			CurrentValue: currentUsage,
			Threshold:    s.disk.Warning,
		}
	}
	return nil
}

// save 현재 임계값을 파일에 원자적으로 저장 (temp write + rename)
// 호출자는 mu.Lock()을 이미 소유해야 함
func (s *ThresholdStore) save() error {
	fileCfg := thresholdsFile{Disk: s.disk}
	data, err := yaml.Marshal(fileCfg)
	if err != nil {
		return fmt.Errorf("thresholds 직렬화 실패: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("thresholds 임시파일 쓰기 실패: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("thresholds 파일 교체 실패: %w", err)
	}
	return nil
}
