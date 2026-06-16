package util

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Collector 시스템 리소스를 단일 goroutine으로 수집해 캐시(last)에 저장하고
// 구독자(subs)에게 브로드캐스트한다. 마지막 구독자가 끊기면 수집을 즉시 중단한다.
type Collector struct {
	mu     sync.Mutex
	subs   map[chan map[string]any]struct{}
	cancel context.CancelFunc
	last   atomic.Value // map[string]any
	svc    atomic.Value // []map[string]interface{} — 느린 루프가 갱신하는 서비스 상태 캐시
}

// serviceStatusInterval 서비스 상태(systemctl/docker/redis) 수집 주기.
// CPU/메모리 등 1초 메트릭과 달리 잘 안 변하고 비싸므로 느슨하게 잡아
// 매초 외부 프로세스 스폰이 1초 fast loop를 막지 않게 한다.
const serviceStatusInterval = 5 * time.Second

var systemCollector = &Collector{subs: make(map[chan map[string]any]struct{})}

// GetCollector 공유 Collector 반환
func GetCollector() *Collector { return systemCollector }

// Subscribe 새 구독 채널(버퍼1)을 등록하고 반환한다.
// subs가 0→1이 되면 수집 goroutine을 기동한다.
// 캐시에 값이 있으면 cold start 시 빈 화면 방지를 위해 즉시 1회 넣어준다.
func (c *Collector) Subscribe() chan map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan map[string]any, 1)
	c.subs[ch] = struct{}{}

	if len(c.subs) == 1 {
		ctx, cancel := context.WithCancel(context.Background())
		c.cancel = cancel
		go c.run(ctx)
	}

	if v, ok := c.last.Load().(map[string]any); ok && v != nil {
		select {
		case ch <- v:
		default:
		}
	}

	return ch
}

// Unsubscribe 구독 채널을 제거하고 close 한다.
// subs가 1→0이 되면 수집 goroutine을 즉시 중단한다.
func (c *Collector) Unsubscribe(ch chan map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.subs[ch]; !ok {
		return
	}
	delete(c.subs, ch)
	close(ch)

	if len(c.subs) == 0 && c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

// Snapshot 캐시된 최신 스냅샷을 반환한다(없으면 nil).
func (c *Collector) Snapshot() map[string]any {
	if v, ok := c.last.Load().(map[string]any); ok {
		return v
	}
	return nil
}

// run 1초 ticker로 스냅샷을 만들어 캐시 갱신 및 브로드캐스트한다.
// ticker 주기(1초)가 곧 CPU 샘플 간격이 되어 time.Sleep이 사라진다.
func (c *Collector) run(ctx context.Context) {
	// 서비스 상태는 느린 루프에서 별도로 수집해 캐시한다. fast loop는 캐시값만 읽어
	// systemctl/docker/redis 호출에 막히지 않으므로 1초 주기가 실제로 지켜진다.
	go c.runServices(ctx)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	prevCPU, _ := readCPUStat()
	startNet, _ := GetNetworkUsage(configs.SC.Setting.NetworkName)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			curCPU, errC := readCPUStat()
			cpu := 0.0
			if errC == nil {
				cpu = calcCPUUsage(prevCPU, curCPU)
				prevCPU = curCPU
			}
			snap, newNet, err := buildSnapshot(cpu, startNet, c.serviceSnapshot())
			if err != nil {
				// 에러 시 broadcast/last 갱신 생략, 직전 정상값 유지 (연결 끊지 않음)
				log.Error(err)
				continue
			}
			startNet = newNet
			c.last.Store(snap)
			c.broadcast(snap)
		}
	}
}

// runServices 서비스 상태를 느린 주기(serviceStatusInterval)로 수집해 캐시한다.
// 기동 직후 1회 즉시 수집해 cold start 시 빈 서비스 목록을 피한다.
func (c *Collector) runServices(ctx context.Context) {
	collect := func() {
		status, err := GetServiceStatus()
		if err != nil {
			log.Error(err)
			return
		}
		c.svc.Store(status)
	}
	collect()

	ticker := time.NewTicker(serviceStatusInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collect()
		}
	}
}

// serviceSnapshot 캐시된 서비스 상태를 반환한다(없으면 빈 슬라이스).
func (c *Collector) serviceSnapshot() []map[string]interface{} {
	if v, ok := c.svc.Load().([]map[string]interface{}); ok {
		return v
	}
	return []map[string]interface{}{}
}

// broadcast 각 구독 채널에 non-blocking 전송한다.
// 버퍼가 차 있으면 오래된 값을 빼고 최신값으로 교체한다(느린 구독자 드롭).
func (c *Collector) broadcast(snap map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for ch := range c.subs {
		select {
		case ch <- snap:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- snap:
			default:
			}
		}
	}
}

// InitCollector 기동 워밍업: 캐시(last)를 1회 선충전한다.
// 기동 시 1회만 sleep을 허용한다(약 1초 소요는 의도된 것).
func InitCollector() {
	prev, _ := readCPUStat()
	time.Sleep(1 * time.Second)
	cur, _ := readCPUStat()
	cpu := calcCPUUsage(prev, cur)

	startNet, _ := GetNetworkUsage(configs.SC.Setting.NetworkName)

	// 서비스 상태도 1회 선수집해 캐시 — fast loop/SSE 첫 응답의 cold start 대비
	if status, err := GetServiceStatus(); err == nil {
		systemCollector.svc.Store(status)
	}

	snap, _, err := buildSnapshot(cpu, startNet, systemCollector.serviceSnapshot())
	if err != nil {
		log.Error(err)
		return
	}
	systemCollector.last.Store(snap)
}
