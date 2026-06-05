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
}

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
			snap, newNet, err := buildSnapshot(cpu, startNet)
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

	snap, _, err := buildSnapshot(cpu, startNet)
	if err != nil {
		log.Error(err)
		return
	}
	systemCollector.last.Store(snap)
}
