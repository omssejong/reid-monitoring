package util

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// defaultStorageDeleteTermDays Redis 설정을 한 번도 읽지 못했을 때 사용하는 기본 보관 일수
const defaultStorageDeleteTermDays = 30

var (
	retentionMu sync.Mutex

	// lastKnownDeleteTermDays 런타임 중 Redis에서 정상적으로 읽어온 마지막 보관 일수.
	// Redis가 끊기거나 storage 필드가 빠진 값이 올라와도 직전 정상값으로 계속 동작하기 위한 캐시.
	// 프로세스 재시작 시에는 초기화된다.
	lastKnownDeleteTermDays int
)

// resolveStorageDeleteTermDays 적용할 보관 일수와 그 출처를 반환한다.
//
//  1. Redis 설정의 storage.deleteTermDays (0 이하면 값이 없는 것으로 본다)
//  2. 런타임 중 마지막으로 정상 조회했던 값
//  3. 기본값 (defaultStorageDeleteTermDays)
func resolveStorageDeleteTermDays() (int, string) {
	if store := GetOmeyeSettingsStore(); store != nil {
		// Get()은 REID_BACK보다 먼저 뜬 경우 nil일 수 있다. 정상 상황이므로 폴백으로 넘어간다.
		if s := store.Get(); s != nil && s.Storage.DeleteTermDays > 0 {
			days := s.Storage.DeleteTermDays
			retentionMu.Lock()
			lastKnownDeleteTermDays = days
			retentionMu.Unlock()
			return days, "redis"
		}
	}

	retentionMu.Lock()
	cached := lastKnownDeleteTermDays
	retentionMu.Unlock()
	if cached > 0 {
		return cached, "last-known"
	}

	return defaultStorageDeleteTermDays, "default"
}

// retentionHour 실행 시각 (KST, 0~23). 미설정이거나 범위를 벗어나면 기본값.
func retentionHour() int {
	h := configs.SC.Setting.StorageRetentionHour
	if h == nil || *h < 0 || *h > 23 {
		return defaultStorageRetentionHour
	}
	return *h
}

// nextRetentionRun 다음 실행 시각. 오늘 hour시가 이미 지났으면 다음 날 같은 시각.
//
// 티커(24시간 간격)가 아니라 매번 다음 시각을 계산하는 이유는, 티커는 기준점이 프로세스
// 기동 시각이라 재시작할 때마다 실행 시각이 옮겨다니고 재시작 횟수만큼 추가 실행되기 때문이다.
// 서버 타임존과 무관하게 항상 KST 기준으로 잡는다.
func nextRetentionRun(now time.Time, hour int) time.Time {
	kstNow := now.In(kstZone)
	next := time.Date(kstNow.Year(), kstNow.Month(), kstNow.Day(), hour, 0, 0, 0, kstZone)
	if !next.After(kstNow) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// retentionCutoff 삭제 기준 시각. 오늘(KST) 자정에서 days 만큼 뺀 시점.
//
// 실행 시각이 아니라 자정 기준인 것은 REID_BACK의 기존 스케줄러가
// LocalDate.now().minusDays(N)로 날짜 단위 비교를 했기 때문이다. 기준을 맞춰 둔다.
func retentionCutoff(now time.Time, days int) time.Time {
	kstNow := now.In(kstZone)
	midnight := time.Date(kstNow.Year(), kstNow.Month(), kstNow.Day(), 0, 0, 0, 0, kstZone)
	return midnight.AddDate(0, 0, -days)
}

// StartStorageRetention 보관 기간이 지난 항목을 매일 정해진 시각에 정리한다.
//
// 대상 디렉토리는 config의 storageRetentionDirs를 따르고, 각 디렉토리의 직계 자식만
// 검사한다(디렉토리면 통째로 삭제). 보관 일수는 REID_BACK이 Redis에 올린 값을 사용한다.
func StartStorageRetention(ctx context.Context) {
	hour := retentionHour()
	root, dirs := retentionConfig()

	log.Info(fmt.Sprintf("storage retention scheduler started: hour=%02d:00(KST) root=%s dirs=%v defaultDays=%d",
		hour, root, dirs, defaultStorageDeleteTermDays))

	for {
		next := nextRetentionRun(time.Now(), hour)
		wait := time.Until(next)

		log.Info(fmt.Sprintf("storage retention next run: %s (in %s)",
			formatKST(next), wait.Truncate(time.Second)))

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Info("storage retention scheduler stopped")
			return
		case <-timer.C:
			runStorageRetention(ctx)
		}
	}
}

// runStorageRetention 정리 Job 1회 기동. 실제 삭제는 StorageJobStore가 비동기로 수행한다.
func runStorageRetention(ctx context.Context) {
	store := GetStorageJobStore()
	if store == nil {
		log.Warn("storage retention skipped: job store not initialized")
		return
	}

	days, source := resolveStorageDeleteTermDays()

	job, err := store.StartRetention(ctx, days)
	if err != nil {
		// 수동 API로 띄운 Job이 돌고 있는 경우. 다음 주기에 다시 시도한다.
		if errors.Is(err, ErrStorageJobBusy) {
			log.Info(fmt.Sprintf("storage retention skipped: another storage job is running (current=%s)",
				store.currentJobID()))
			return
		}
		log.Error(fmt.Errorf("storage retention start failed: %w", err))
		return
	}

	log.Info(fmt.Sprintf("storage retention accepted: id=%s days=%d source=%s cutoff=%s",
		job.ID, days, source, job.Cutoff))
}
