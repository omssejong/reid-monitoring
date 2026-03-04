# 아키텍처

## 1. 개요

`reid-monitoring`은 OMS/ReID 운영 환경에서 서버 상태 조회, 서비스 제어, 로그 다운로드, 패치 적용을 수행하는 Go 기반 모니터링 에이전트입니다.

주요 기능:
- HTTP API 제공 (`/monitoring/mgmt/*`)
- CPU/GPU/메모리/디스크/네트워크 상태 수집
- 서비스 재시작, 서버 재부팅/종료 제어
- 로그 파일 필터링/암호화/압축 다운로드
- 패치 파일 검증/복호화/실행

핵심 기술:
- Go 1.26
- Gin
- NVML(gonvml)
- Linux `/proc`, `/sys`, 표준 라이브러리 기반 시스템 정보 수집
- slog
- YAML 설정

## 2. 디렉터리 구조

```text
.
|- main.go
|- go.mod
|- go.sum
|- README.md
|- ARCHITECTURE.md
|- omeye2.sc_monitoring.service
|- conf.d/
|  |- config-dev.yml
|  |- config-local.yml
|  |- config-prod.yml
|  |- config-docker.yml
|  |- docker-compose.yml
|  `- legacy/
|     |- monitoring-dev.ini
|     |- monitoring-local.ini
|     `- monitoring-prod.ini
`- util/
   |- common.go
   |- config.go
   |- web.go
   |- middelware.go
   |- sse.go
   |- service.go
   |- cpu.go
   |- gpu.go
   |- memory.go
   |- disk.go
   |- network.go
   |- file.go
   |- compress.go
   |- patch.go
   |- logger.go
   `- worker.go
```

참고:
- 현재 설정 로딩은 `conf.d/config-*.yml`(YAML 단일 포맷)만 사용합니다.
- `conf.d/legacy/*.ini`는 보관용 레거시 파일입니다.

## 3. 실행 흐름

1. `main.go`에서 `data/` 디렉터리 존재 확인/생성
2. `util` 패키지 import 시 전역 초기화 수행 (`util/common.go`)
3. `NewConfig(os.Args[1])`로 활성 프로파일 YAML 로딩
4. `util.WebApp()`으로 라우터/미들웨어 구성 후 HTTP 서버 시작
5. SIGINT/SIGTERM 수신 시 graceful shutdown

## 4. 설정 구조

설정 파일: `conf.d/config-{dev|local|prod|docker}.yml`

YAML 최상위 키:
- `SETTING`
- `VERSION`
- `CATEGORY`
- `REDIS`

`util/config.go`에서 위 YAML을 직접 파싱하여 `Config` 구조체로 매핑합니다.

## 5. API 엔드포인트

기본 경로: `/monitoring/mgmt`

- `GET /server-info`: 서버/하드웨어/네트워크/버전 정보
- `GET /info`: SSE 실시간 상태 스트림
- `POST /reboot`: 서버 재부팅
- `POST /servicectrl`: 서비스 재시작
- `POST /log/download`: 로그 다운로드(필터링 + 암호화 + 압축)
- `POST /upload/patch`: 패치 업로드 및 적용

## 6. 모듈 구성

- 웹 계층: `util/web.go`, `util/middelware.go`
- 메트릭 계층:
  - `util/cpu.go` (`/proc/cpuinfo`, `/proc/stat`)
  - `util/memory.go` (`/proc/meminfo`)
  - `util/disk.go` (`syscall.Statfs`, `df` 유사 계산)
  - `util/network.go` (`net`, `/proc/net/dev`, `/proc/net/route`, `/sys/class/net/*/speed`)
  - `util/gpu.go` (NVML)
  - `util/sse.go` (1초 주기 수집/스트리밍)
- 운영 계층: `util/service.go`, `util/file.go`, `util/compress.go`, `util/patch.go`
- 공통 계층: `util/config.go`, `util/logger.go`, `util/common.go`

## 7. 의존성 상태

제거 완료:
- `go-redis`
- `logrus`, `lumberjack`
- `gopsutil`
- `ini` 파서(`gopkg.in/ini.v1`)
- 네트워크 수집에서 `ifconfig`, `route -n`, `ethtool` 의존

현재 주요 외부 의존:
- `github.com/gin-gonic/gin`
- `github.com/mindprince/gonvml`
- `gopkg.in/yaml.v3`

## 8. 운영 전제

Linux 환경 전제:
- `/proc/cpuinfo`, `/proc/stat`, `/proc/meminfo`, `/proc/net/dev`, `/proc/net/route`
- `/sys/class/net/<iface>/speed`
- 서비스 제어 명령: `bash`, `systemctl`, `sudo`, `docker`, `virsh`

즉, 시스템 메트릭 수집과 서비스 제어 모두 Linux 중심 동작을 전제로 합니다.
