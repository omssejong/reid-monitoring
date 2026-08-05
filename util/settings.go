package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// omeyeSettingsKey REID_BACK이 서버 설정 전체를 올려두는 Redis 키 (String, JSON, TTL 없음)
	omeyeSettingsKey = "omeye:settings"

	// omeyeSettingsChannel 설정 변경 통지 채널.
	// 메시지 내용은 쓰지 않는다. pub/sub은 전달 순서를 보장하지 않으므로 통지는 신호로만 쓰고
	// 값은 항상 키에서 다시 읽는다. 그래야 통지가 역전돼도 결과가 최신으로 수렴한다.
	omeyeSettingsChannel = "settings-updated"

	// omeyeSettingsRefreshInterval 주기적 재조회 간격.
	// pub/sub은 구독이 끊겨 있는 동안 온 메시지를 버리기 때문에 통지만 믿으면
	// 네트워크가 한 번 끊긴 뒤 계속 낡은 설정으로 돌게 된다. 그 구멍을 메우는 안전망.
	omeyeSettingsRefreshInterval = 5 * time.Minute
)

// ErrOmeyeSettingsNotPublished REID_BACK이 아직 스냅샷을 올리지 않은 상태.
// 장애가 아니라 기동 순서 문제일 수 있으므로 호출측이 구분할 수 있도록 별도 에러로 둔다.
var ErrOmeyeSettingsNotPublished = errors.New("omeye settings not published yet")

// OmeyeSettings REID_BACK이 발행하는 서버 설정 스냅샷.
// 구성은 REID_BACK의 GET /api/v3/settings 응답과 같고 storage/updatedAt이 추가돼 있다.
//
// Jackson NON_NULL 설정이라 null 필드는 키 자체가 빠진 채로 온다.
// (Java 원시 타입인 int/boolean 필드는 null이 될 수 없어 항상 존재한다.)
type OmeyeSettings struct {
	Server       OmeyeServerSetting  `json:"server"`
	AvailService OmeyeAvailSetting   `json:"availService"`
	Frame        OmeyeFrameSetting   `json:"frame"`
	Model        OmeyeModelSetting   `json:"model"`
	Storage      OmeyeStorageSetting `json:"storage"`

	// Map은 지도 UI 전용이라 모니터링에서 쓸 일이 없다. 구조를 따라 만들지 않고 원본만 보관한다.
	Map json.RawMessage `json:"map,omitempty"`

	// UpdatedAt 발행 시각(RFC3339). 값 비교용이 아니라 로그/디버깅용이다.
	UpdatedAt string `json:"updatedAt"`
}

type OmeyeServerSetting struct {
	ThreshHold          int    `json:"threshHold"`
	MaxResultDuration   int    `json:"maxResultDuration"`
	MaxAnalyzeCount     int    `json:"maxAnalyzeCount"`
	MaxLiveAnalyzeCount int    `json:"maxLiveAnalyzeCount"`
	MaxAnalyzeDuration  int    `json:"maxAnalyzeDuration"`
	ServiceLang         string `json:"serviceLang"`
}

type OmeyeAvailSetting struct {
	ReIdAvail                bool `json:"reIdAvail"`
	MonitoringAvail          bool `json:"monitoringAvail"`
	ExportAvail              bool `json:"exportAvail"`
	AreaAnalyzeAvail         bool `json:"areaAnalyzeAvail"`
	IsRtspAnalyze            bool `json:"isRtspAnalyze"`
	IsDefaultAnalyzeDownload bool `json:"isDefaultAnalyzeDownload"`
	MiddleServer             bool `json:"middleServer"`
	UseOmpass                bool `json:"useOmpass"`
	RtspSpeed                int  `json:"rtspSpeed"`
}

type OmeyeFrameSetting struct {
	CarFrame         int `json:"carFrame"`
	PersonFrame      int `json:"personFrame"`
	FaceFrame        int `json:"faceFrame"`
	AttributionFrame int `json:"attributionFrame"`
}

type OmeyeModelSetting struct {
	Person   bool `json:"person"`
	Car      bool `json:"car"`
	Face     bool `json:"face"`
	CarPlate bool `json:"carPlate"`
}

// OmeyeStorageSetting REID_BACK의 설정 파일(omeye.server.delete-term-day) 유래 값.
// 나머지와 달리 API로 바뀌지 않고 변경하려면 REID_BACK 재부팅이 필요하다(정책).
type OmeyeStorageSetting struct {
	// DeleteTermDays 파일 보관 일수. 이보다 오래된 항목이 정리 대상이다.
	DeleteTermDays int `json:"deleteTermDays"`
}

// OmeyeSettingsStore 설정 스냅샷 보관소. 여러 goroutine에서 동시에 Get해도 안전하다.
type OmeyeSettingsStore struct {
	rdb *redis.Client

	mu      sync.RWMutex
	current *OmeyeSettings
}

// 전역 인스턴스 (main.go에서 초기화)
var omeyeSettingsStore *OmeyeSettingsStore

// InitOmeyeSettingsStore 설정 보관소 초기화 + 구독/주기 재조회 고루틴 시작.
// Redis 접속 정보가 없으면 보관소를 만들지 않는다(호출측은 nil을 정상 상태로 다뤄야 한다).
func InitOmeyeSettingsStore(ctx context.Context) {
	host := configs.Redis.RedisHost
	if host == "" {
		log.Warn("omeye settings store disabled: RedisHost is empty")
		return
	}

	port := configs.Redis.RedisPort
	if port == 0 {
		port = 6379
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(host, strconv.Itoa(port)),
		Username: configs.Redis.Username,
		Password: configs.Redis.Password,
	})

	store := &OmeyeSettingsStore{rdb: rdb}
	omeyeSettingsStore = store

	// 최초 읽기가 실패해도 계속 진행한다. REID_BACK이 아직 안 떴을 수 있고,
	// 그 경우 첫 통지나 주기적 재조회에서 채워진다.
	if err := store.Reload(ctx); err != nil {
		log.Warn(fmt.Sprintf("initial omeye settings load failed, will retry on notification: %v", err))
	}
	go store.watch(ctx)

	log.Info(fmt.Sprintf("omeye settings store initialized: addr=%s key=%s", rdb.Options().Addr, omeyeSettingsKey))
}

// GetOmeyeSettingsStore 전역 보관소 반환. 초기화 전이거나 Redis 설정이 없으면 nil이다.
func GetOmeyeSettingsStore() *OmeyeSettingsStore {
	return omeyeSettingsStore
}

// Get 마지막으로 읽어온 설정 반환.
//
// 한 번도 읽지 못했으면 nil이다. 모니터링이 REID_BACK보다 먼저 뜨는 경우가 있으므로
// 호출측 nil 체크가 필수다. 기동 실패로 다루면 안 된다.
func (s *OmeyeSettingsStore) Get() *OmeyeSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Reload Redis에서 스냅샷을 다시 읽어 보관 값을 교체한다.
// 키가 아직 없으면 ErrOmeyeSettingsNotPublished를 반환하고 기존 값은 유지한다.
func (s *OmeyeSettingsStore) Reload(ctx context.Context) error {
	raw, err := s.rdb.Get(ctx, omeyeSettingsKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrOmeyeSettingsNotPublished
	}
	if err != nil {
		return err
	}

	var loaded OmeyeSettings
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return err
	}

	// 파싱에 성공한 경우에만 교체한다.
	// 중간에 깨진 값을 받았을 때 기존 정상 설정을 날리지 않기 위함.
	// 보관 값은 통째로 교체만 하고 제자리에서 수정하지 않으므로 prev는 락 밖에서 읽어도 안전하다.
	s.mu.Lock()
	prev := s.current
	s.current = &loaded
	s.mu.Unlock()

	// 5분 주기 재조회가 같은 줄로 로그를 채우지 않도록 최초 로드와 실제 변경만 남긴다.
	switch {
	case prev == nil:
		log.Info(fmt.Sprintf("omeye settings loaded: updatedAt=%s deleteTermDays=%d",
			toKST(loaded.UpdatedAt), loaded.Storage.DeleteTermDays))
	case prev.UpdatedAt != loaded.UpdatedAt || prev.Storage.DeleteTermDays != loaded.Storage.DeleteTermDays:
		log.Info(fmt.Sprintf("omeye settings updated: updatedAt=%s -> %s deleteTermDays=%d -> %d",
			toKST(prev.UpdatedAt), toKST(loaded.UpdatedAt),
			prev.Storage.DeleteTermDays, loaded.Storage.DeleteTermDays))
	}
	return nil
}

// watch 변경 통지를 받을 때마다 다시 읽고, 주기적으로도 재조회한다.
// ctx가 취소되면 구독을 정리하고 종료한다.
func (s *OmeyeSettingsStore) watch(ctx context.Context) {
	pubsub := s.rdb.Subscribe(ctx, omeyeSettingsChannel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Warn(fmt.Sprintf("omeye settings pubsub close failed: %v", err))
		}
	}()

	// Channel()은 연결이 끊기면 내부적으로 재구독한다.
	ch := pubsub.Channel()

	ticker := time.NewTicker(omeyeSettingsRefreshInterval)
	defer ticker.Stop()

	log.Info(fmt.Sprintf("omeye settings watcher started: key=%s channel=%s", omeyeSettingsKey, omeyeSettingsChannel))

	for {
		select {
		case <-ctx.Done():
			log.Info("omeye settings watcher stopped")
			return

		case _, ok := <-ch:
			if !ok {
				// 구독이 완전히 끊긴 경우. 여기서 종료하면 주기 재조회까지 같이 멈춰
				// 낡은 보관 일수로 삭제를 계속하게 되므로, 티커만 남기고 계속 돈다.
				log.Warn("omeye settings pubsub channel closed, falling back to periodic refresh only")
				ch = nil
				continue
			}
			// 통지 내용은 쓰지 않는다. 신호일 뿐이고 값은 키에서 읽는다.
			if err := s.Reload(ctx); err != nil {
				log.Warn(fmt.Sprintf("omeye settings reload failed: %v", err))
			}

		case <-ticker.C:
			// 구독이 끊겨 있던 동안의 변경을 메우는 안전망.
			if err := s.Reload(ctx); err != nil && !errors.Is(err, ErrOmeyeSettingsNotPublished) {
				log.Warn(fmt.Sprintf("periodic omeye settings refresh failed: %v", err))
			}
		}
	}
}
