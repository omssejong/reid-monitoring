package util

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	storageJobStatusPending   = "pending"
	storageJobStatusRunning   = "running"
	storageJobStatusCompleted = "completed"
	storageJobStatusFailed    = "failed"

	storageJobTTL             = time.Hour
	storageJobCleanupInterval = 5 * time.Minute

	storageDeleteLogEvery = 100 // 몇 건마다 진행 로그를 찍을지
)

// ErrStorageJobBusy 이미 실행 중인 Job이 있을 때 반환
var ErrStorageJobBusy = errors.New("another storage job is running")

// StorageJob 비동기 스토리지 삭제 작업 상태
type StorageJob struct {
	ID     string `json:"jobId"`
	Mode   string `json:"mode"`   // "percent" | "date" | "retention"
	Status string `json:"status"` // pending | running | completed | failed

	Progress float64 `json:"progress"` // 0~100

	// percent 모드
	TargetPercent  float64 `json:"targetPercent,omitempty"`
	StartPercent   float64 `json:"startPercent,omitempty"`
	CurrentPercent float64 `json:"currentPercent,omitempty"`
	TargetReached  *bool   `json:"targetReached,omitempty"` // true=목표 달성, false=후보 소진했는데 미달성

	// date 모드
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`

	// retention 모드
	RetentionDays int    `json:"retentionDays,omitempty"`
	Cutoff        string `json:"cutoff,omitempty"` // 이 시각보다 오래된(mtime) 항목이 삭제 대상

	// date / retention 공통
	TotalCandidates int `json:"totalCandidates,omitempty"`
	Processed       int `json:"processed,omitempty"`

	// 공통
	DeletedCount int                `json:"deletedCount"`
	DeletedBytes int64              `json:"deletedBytes"`
	StartedAt    time.Time          `json:"startedAt"`
	FinishedAt   *time.Time         `json:"finishedAt"`
	ErrorMsg     string             `json:"error,omitempty"`
	StorageInfo  *ServerStorageInfo `json:"storageInfo,omitempty"`

	mu sync.RWMutex
}

// Snapshot 동시성 안전한 상태 복사본
func (j *StorageJob) Snapshot() StorageJob {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var finishedAt *time.Time
	if j.FinishedAt != nil {
		t := *j.FinishedAt
		finishedAt = &t
	}
	var info *ServerStorageInfo
	if j.StorageInfo != nil {
		cp := *j.StorageInfo
		info = &cp
	}
	var targetReached *bool
	if j.TargetReached != nil {
		v := *j.TargetReached
		targetReached = &v
	}
	return StorageJob{
		ID:              j.ID,
		Mode:            j.Mode,
		Status:          j.Status,
		Progress:        j.Progress,
		TargetPercent:   j.TargetPercent,
		StartPercent:    j.StartPercent,
		CurrentPercent:  j.CurrentPercent,
		TargetReached:   targetReached,
		From:            j.From,
		To:              j.To,
		RetentionDays:   j.RetentionDays,
		Cutoff:          j.Cutoff,
		TotalCandidates: j.TotalCandidates,
		Processed:       j.Processed,
		DeletedCount:    j.DeletedCount,
		DeletedBytes:    j.DeletedBytes,
		StartedAt:       j.StartedAt,
		FinishedAt:      finishedAt,
		ErrorMsg:        j.ErrorMsg,
		StorageInfo:     info,
	}
}

// StorageJobStore 스토리지 Job 저장소 (동시 1개 실행 보장)
type StorageJobStore struct {
	mu      sync.Mutex             // current 보호
	current *StorageJob            // 실행 중 Job (nil이면 idle)
	jobsMu  sync.RWMutex           // jobs 보호
	jobs    map[string]*StorageJob // ID → Job (TTL 기반 보관)
	ttl     time.Duration
}

var storageJobStore *StorageJobStore

// InitStorageJobStore 저장소 초기화 + TTL 정리 고루틴 시작
func InitStorageJobStore(ctx context.Context) {
	storageJobStore = &StorageJobStore{
		jobs: make(map[string]*StorageJob),
		ttl:  storageJobTTL,
	}
	go storageJobStore.cleanupLoop(ctx)
	log.Info("storage job store initialized")
}

// GetStorageJobStore 전역 저장소 반환
func GetStorageJobStore() *StorageJobStore {
	return storageJobStore
}

// GetJob ID로 조회
func (s *StorageJobStore) GetJob(id string) (*StorageJob, bool) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

// currentJobID 현재 실행 중 Job ID (없으면 "")
func (s *StorageJobStore) currentJobID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ""
	}
	return s.current.ID
}

// StartPercent percent 모드 Job 시작. busy면 ErrStorageJobBusy.
// 현재 사용률이 target 이하면 nil job 반환하고 현재 상태만 같이 돌려준다.
func (s *StorageJobStore) StartPercent(ctx context.Context, target float64) (*StorageJob, *ServerStorageInfo, error) {
	info, err := GetServerStorageInfo()
	if err != nil {
		return nil, nil, err
	}
	if info.UsedPercent <= target {
		// 이미 달성 → 즉시 현재 상태 반환
		return nil, &info, nil
	}

	s.mu.Lock()
	if s.current != nil {
		s.mu.Unlock()
		return nil, nil, ErrStorageJobBusy
	}
	job := s.newJob("percent")
	job.TargetPercent = target
	job.StartPercent = info.UsedPercent
	job.CurrentPercent = info.UsedPercent
	s.current = job
	s.mu.Unlock()

	s.register(job)
	go s.runPercent(ctx, job)
	return job, nil, nil
}

// StartDate date 모드 Job 시작. busy면 ErrStorageJobBusy.
func (s *StorageJobStore) StartDate(ctx context.Context, from, to time.Time, fromStr, toStr string) (*StorageJob, error) {
	s.mu.Lock()
	if s.current != nil {
		s.mu.Unlock()
		return nil, ErrStorageJobBusy
	}
	job := s.newJob("date")
	job.From = fromStr
	job.To = toStr
	s.current = job
	s.mu.Unlock()

	s.register(job)
	go s.runDate(ctx, job, from, to)
	return job, nil
}

// StartRetention 보관 기간(days) 초과 항목 삭제 Job 시작. busy면 ErrStorageJobBusy.
// 하루 주기 자동 정리에서 호출한다.
func (s *StorageJobStore) StartRetention(ctx context.Context, days int) (*StorageJob, error) {
	if days <= 0 {
		return nil, fmt.Errorf("invalid retention days: %d", days)
	}
	cutoff := retentionCutoff(time.Now(), days)

	s.mu.Lock()
	if s.current != nil {
		s.mu.Unlock()
		return nil, ErrStorageJobBusy
	}
	job := s.newJob("retention")
	job.RetentionDays = days
	job.Cutoff = formatKST(cutoff)
	s.current = job
	s.mu.Unlock()

	s.register(job)
	go s.runRetention(ctx, job, cutoff)
	return job, nil
}

// newJob 새 Job 구조체 초기화
func (s *StorageJobStore) newJob(mode string) *StorageJob {
	return &StorageJob{
		ID:        generateStorageJobID(),
		Mode:      mode,
		Status:    storageJobStatusRunning,
		StartedAt: time.Now(),
	}
}

func (s *StorageJobStore) register(job *StorageJob) {
	s.jobsMu.Lock()
	s.jobs[job.ID] = job
	s.jobsMu.Unlock()
}

// runPercent percent 모드 실행
func (s *StorageJobStore) runPercent(ctx context.Context, job *StorageJob) {
	defer s.finishJob(job)

	root, protected, reidResult, excluded := storageConfig()
	candidates, err := collectCandidates(root, protected, reidResult, excluded)
	if err != nil {
		s.setError(job, fmt.Sprintf("collect candidates: %v", err))
		return
	}

	// mtime 오름차순 (오래된 것부터)
	sort.Slice(candidates, func(i, k int) bool {
		return candidates[i].mtime.Before(candidates[k].mtime)
	})

	log.Info(fmt.Sprintf("storage job percent start: id=%s target=%.2f start=%.2f candidates=%d",
		job.ID, job.TargetPercent, job.StartPercent, len(candidates)))

	reached := false
	loopCount := 0
	for _, c := range candidates {
		if ctx.Err() != nil {
			s.setError(job, "cancelled")
			return
		}

		if err := removeCandidate(c); err != nil {
			log.Error(fmt.Errorf("remove %s: %w", c.path, err))
			continue
		}

		loopCount++
		job.mu.Lock()
		job.DeletedCount++
		job.DeletedBytes += c.size
		job.mu.Unlock()

		// 현재 사용률 재조회 및 진행률 갱신
		usage, err := GetRootDiskUsagePercent()
		if err != nil {
			log.Error(fmt.Errorf("disk usage: %w", err))
			continue
		}

		job.mu.Lock()
		job.CurrentPercent = roundTo(usage, 2)
		job.Progress = calcPercentProgress(job.StartPercent, job.CurrentPercent, job.TargetPercent)
		job.mu.Unlock()

		if loopCount%storageDeleteLogEvery == 0 {
			log.Info(fmt.Sprintf("storage job percent progress: id=%s deleted=%d current=%.2f target=%.2f",
				job.ID, loopCount, usage, job.TargetPercent))
		}

		if usage <= job.TargetPercent {
			reached = true
			break
		}
	}

	// 목표 달성 여부 기록
	job.mu.Lock()
	job.TargetReached = &reached
	job.mu.Unlock()

	if !reached {
		log.Warn(fmt.Sprintf("storage job percent target not reached: id=%s current=%.2f target=%.2f deleted=%d",
			job.ID, job.CurrentPercent, job.TargetPercent, job.DeletedCount))
	}
}

// runDate date 모드 실행
func (s *StorageJobStore) runDate(ctx context.Context, job *StorageJob, from, to time.Time) {
	defer s.finishJob(job)

	root, protected, reidResult, excluded := storageConfig()
	candidates, err := collectCandidates(root, protected, reidResult, excluded)
	if err != nil {
		s.setError(job, fmt.Sprintf("collect candidates: %v", err))
		return
	}

	// 날짜 범위 필터링 (mtime 기준)
	filtered := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if (c.mtime.Equal(from) || c.mtime.After(from)) && c.mtime.Before(to) {
			filtered = append(filtered, c)
		}
	}

	// mtime 오름차순 정렬 (오래된 것부터 삭제 — percent 모드와 동일 정책)
	sort.Slice(filtered, func(i, k int) bool {
		return filtered[i].mtime.Before(filtered[k].mtime)
	})

	job.mu.Lock()
	job.TotalCandidates = len(filtered)
	job.mu.Unlock()

	log.Info(fmt.Sprintf("storage job date start: id=%s from=%s to=%s candidates=%d",
		job.ID, job.From, job.To, len(filtered)))

	s.deleteFiltered(ctx, job, filtered, false)
}

// runRetention retention 모드 실행.
// 지정된 디렉토리의 직계 자식 중 cutoff 이전(mtime 기준) 항목을 삭제한다.
// mtime을 쓰는 것은 Linux에서 생성 시각을 얻을 수 없기 때문이며, 기존 스케줄러도 동일하다.
func (s *StorageJobStore) runRetention(ctx context.Context, job *StorageJob, cutoff time.Time) {
	defer s.finishJob(job)

	root, dirs := retentionConfig()
	targets := collectRetentionCandidates(root, dirs, cutoff)

	// mtime 오름차순 정렬 (오래된 것부터 삭제 — percent/date 모드와 동일 정책)
	sort.Slice(targets, func(i, k int) bool {
		return targets[i].mtime.Before(targets[k].mtime)
	})

	job.mu.Lock()
	job.TotalCandidates = len(targets)
	job.mu.Unlock()

	log.Info(fmt.Sprintf("storage job retention start: id=%s days=%d cutoff=%s dirs=%v targets=%d",
		job.ID, job.RetentionDays, job.Cutoff, dirs, len(targets)))

	// 대상이 디렉토리 단위라 건수가 적다. 자동 삭제이므로 항목별로 남겨 사후 추적이 가능하게 한다.
	s.deleteFiltered(ctx, job, targets, true)

	// 한 건이라도 지웠으면 요약을 남긴다. 지운 게 없는 날까지 남기면 신호가 묻힌다.
	snap := job.Snapshot()
	if snap.DeletedCount > 0 {
		log.Info(fmt.Sprintf("storage retention deleted: id=%s days=%d cutoff=%s deleted=%d/%d bytes=%.2fGiB mtime=%s~%s",
			snap.ID, snap.RetentionDays, snap.Cutoff, snap.DeletedCount, len(targets),
			bytesToGiB(uint64(snap.DeletedBytes)),
			formatKST(targets[0].mtime), formatKST(targets[len(targets)-1].mtime)))
	}
}

// deleteFiltered 이미 확정된 후보 목록을 순서대로 삭제하며 진행률을 갱신한다.
// date / retention 모드가 공유한다.
//
// logEach가 true면 항목마다 로그를 남긴다. date 모드는 파일 단위라 수만 건이 될 수 있어
// 진행 로그만 남기고, retention 모드는 디렉토리 단위라 항목별로 남긴다.
func (s *StorageJobStore) deleteFiltered(ctx context.Context, job *StorageJob, filtered []candidate, logEach bool) {
	loopCount := 0
	for _, c := range filtered {
		if ctx.Err() != nil {
			s.setError(job, "cancelled")
			return
		}

		if logEach {
			log.Info(fmt.Sprintf("deleting old entry: %s (mtime=%s)", c.path, formatKST(c.mtime)))
		}

		if err := removeCandidate(c); err != nil {
			log.Error(fmt.Errorf("remove %s: %w", c.path, err))
		} else {
			job.mu.Lock()
			job.DeletedCount++
			job.DeletedBytes += c.size
			job.mu.Unlock()
		}

		loopCount++
		job.mu.Lock()
		job.Processed++
		if job.TotalCandidates > 0 {
			job.Progress = roundTo(float64(job.Processed)/float64(job.TotalCandidates)*100, 2)
		} else {
			job.Progress = 100
		}
		job.mu.Unlock()

		if !logEach && loopCount%storageDeleteLogEvery == 0 {
			log.Info(fmt.Sprintf("storage job %s progress: id=%s processed=%d/%d deleted=%d",
				job.Mode, job.ID, loopCount, len(filtered), job.DeletedCount))
		}
	}
}

// finishJob Job 완료 처리 (상태 갱신, 최종 storageInfo 수집, current 해제)
func (s *StorageJobStore) finishJob(job *StorageJob) {
	info, infoErr := GetServerStorageInfo()
	now := time.Now()

	job.mu.Lock()
	job.FinishedAt = &now
	if job.ErrorMsg != "" {
		job.Status = storageJobStatusFailed
	} else {
		job.Status = storageJobStatusCompleted
	}
	job.Progress = 100
	if infoErr == nil {
		cp := info
		job.StorageInfo = &cp
	}
	job.mu.Unlock()

	log.Info(fmt.Sprintf("storage job finished: id=%s mode=%s status=%s deleted=%d",
		job.ID, job.Mode, job.Status, job.DeletedCount))

	s.mu.Lock()
	if s.current != nil && s.current.ID == job.ID {
		s.current = nil
	}
	s.mu.Unlock()
}

func (s *StorageJobStore) setError(job *StorageJob, msg string) {
	job.mu.Lock()
	if job.ErrorMsg == "" {
		job.ErrorMsg = msg
	}
	job.mu.Unlock()
}

// cleanupLoop TTL 경과한 완료 Job 정리
func (s *StorageJobStore) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(storageJobCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

func (s *StorageJobStore) cleanup() {
	now := time.Now()
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	for id, j := range s.jobs {
		j.mu.RLock()
		finished := j.FinishedAt
		j.mu.RUnlock()
		if finished != nil && now.Sub(*finished) > s.ttl {
			delete(s.jobs, id)
		}
	}
}

// calcPercentProgress percent 모드 진행률 계산 (0~100)
func calcPercentProgress(start, current, target float64) float64 {
	denom := start - target
	if denom <= 0 {
		return 100
	}
	p := (start - current) / denom * 100
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return roundTo(p, 2)
}

// generateStorageJobID "stg-YYYYMMDD-HHMMSS-hex6" 생성
func generateStorageJobID() string {
	now := time.Now()
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("stg-%s-%s", now.Format("20060102-150405"), hex.EncodeToString(buf))
}
