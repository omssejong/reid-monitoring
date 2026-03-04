# 아키텍처

## 1. 개요

`reid-monitoring`은 OMS/ReID 배포 환경을 위한 Go 기반 모니터링 및 제어 에이전트입니다.

주요 역할:
- 서버 정보, 실시간 메트릭(SSE), 서비스 제어, 로그 다운로드, 패치 업로드용 HTTP API 제공
- 시스템 리소스(CPU, GPU, 메모리, 디스크, 네트워크) 수집
- 호스트 서비스 및 Docker 기반 서비스 제어
- 다운로드용 로그 암호화/압축 패키징

핵심 스택:
- 언어/런타임: Go 1.21.6
- 웹 프레임워크: Gin
- 메트릭 수집: gopsutil, NVML(gonvml), Linux procfs/명령어(`/proc`, `ifconfig`, `route`, `ethtool`)
- 로깅: logrus + lumberjack
- 선택적 Pub/Sub: Redis 클라이언트 래퍼

## 2. 프로젝트 구조

```text
.
|- main.go
|- go.mod / go.sum
|- README.md
|- omeye2.sc_monitoring.service
|- conf.d/
|  |- config-{dev,local,prod,docker}.yml
|  |- monitoring-{dev,local,prod}.ini
|  `- docker-compose.yml
`- util/
   |- common.go        # 전역 설정/부트스트랩 변수
   |- config.go        # yaml + ini 로딩
   |- web.go           # HTTP 라우트 + 핸들러
   |- middelware.go    # CORS/보안/SSE 미들웨어
   |- sse.go           # 1초 주기 스트리밍 메트릭 루프
   |- service.go       # systemctl/docker/reboot/shutdown 제어
   |- cpu.go           # CPU 메타데이터/사용률
   |- gpu.go           # GPU 메타데이터/사용률(NVML)
   |- memory.go        # 메모리 통계
   |- disk.go          # 디스크 통계
   |- network.go       # 인터페이스 정보/트래픽/대역폭
   |- file.go          # 로그 파일 필터링/검색/정리
   |- compress.go      # 로그 암호화 + zip 압축
   |- patch.go         # 패치 해시검증/복호화/압축해제/실행
   |- redis.go         # Redis 래퍼
   |- logger.go        # 로깅 추상화
   `- worker.go        # 레거시 워커(현재 주석 처리)
```

## 3. 시작 흐름

1. `main.go`가 `data/` 디렉터리가 없으면 생성합니다.
2. `util` 패키지 import 시 `util/common.go` 전역 초기화가 실행됩니다.
   - CLI 인자(`os.Args[1]`)의 활성 프로파일로 `NewConfig(active)` 호출
   - `conf.d/config-<active>.yml`과 참조된 INI 설정 로딩
   - 전역 경로/서비스명/비밀번호/토큰/네트워크 인터페이스 설정
   - 로거 및 Redis 클라이언트 초기화
3. `util.WebApp()`이 Gin 라우터와 미들웨어를 구성합니다.
4. `SC.Setting.ServerPort` 포트에서 HTTP 서버를 시작합니다.
5. SIGINT/SIGTERM 수신 시 5초 타임아웃으로 graceful shutdown을 수행합니다.

## 4. 설정 모델

설정은 2단계로 해석됩니다.
- 1단계: `conf.d/config-<profile>.yml` (`CONFIG.ConfigPath`, `REDIS.*`)
- 2단계: `ConfigPath`가 가리키는 INI (`[SETTING]`, `[VERSION]`, `[CATEGORY]`)

저장소 내 프로파일:
- `dev`, `local`, `prod`, `docker`

주요 런타임 설정값:
- 서버 포트 / 네트워크 인터페이스
- backend/AI/media/image processing 서비스명
- 루트 및 카테고리 경로
- 권한 명령 실행에 사용되는 사용자 비밀번호
- 로그 암호화 키 재료로 쓰이는 토큰

## 5. HTTP API 표면

기본 그룹: `/monitoring/mgmt`

- `GET /server-info`
  - 서버 하드웨어/네트워크/버전 스냅샷 조회
- `GET /info`
  - 1초 주기 실시간 메트릭 SSE 스트림
- `POST /reboot`
  - 호스트 서버 재부팅
- `POST /servicectrl`
  - 서비스 타입 기준 대상 서비스 재시작
- `POST /log/download`
  - 날짜 범위 로그 필터링 후 암호화/압축하여 파일 다운로드 응답
- `POST /upload/patch`
  - 암호화된 패치 파일과 해시를 받아 검증/복호화/압축해제/스크립트 실행

## 6. 내부 컴포넌트

### 6.1 HTTP 계층 (`util/web.go`, `util/middelware.go`)
- Gin release 모드 사용
- 미들웨어: 기본 로깅/리커버리(Gin), 보안 헤더, 개방형 CORS, SSE 헤더
- 핸들러가 메트릭/서비스/로그/패치 모듈을 오케스트레이션

### 6.2 메트릭 계층 (`cpu.go`, `gpu.go`, `memory.go`, `disk.go`, `network.go`, `sse.go`)
- Pull 기반 메트릭 수집
- SSE 루프가 1초마다 다음 정보를 전송:
  - uptime
  - 서비스 상태
  - CPU/GPU/메모리/디스크 사용률
  - 네트워크 대역폭 변화량(uplink/downlink/total)

### 6.3 서비스 제어 계층 (`service.go`)
- `bash -c` 기반 셸 명령으로 sudo/systemctl/docker/virsh/reboot/shutdown 실행
- system service와 docker target 모두 지원

### 6.4 로그 패키징 계층 (`file.go`, `compress.go`)
- 날짜 범위 기준 로그 + 고정 시스템 로그 선택
- 파일별 AES CFB 암호화 후 `data/` 하위 zip 생성
- API 다운로드 응답용 경로 반환

### 6.5 패치 계층 (`patch.go`, `web.go` 일부)
- 업로드 파일 + SHA-256 해시 수신
- 해시 검증
- 페이로드 복호화(AES-CBC, 코드 내 고정 key/iv 사용)
- `temp/patch.zip` 생성, 압축 해제 후 `temp/patch_script.sh` 실행

### 6.6 통합 유틸리티 (`redis.go`, `logger.go`)
- Redis 래퍼는 set/get/delete/publish 제공(현재는 레거시 워커 경로에서 주 사용)
- 커스텀 로거는 logrus 래핑 + lumberjack 기반 로테이션

## 7. 런타임/OS 가정

주 대상은 Linux 런타임입니다.
- `/proc/uptime`, `/proc/net/dev` 사용
- `ifconfig`, `route`, `ethtool`, `bash`, `systemctl`, `sudo`, `docker`, `virsh` 명령 의존
- GPU 메트릭을 위해 NVML 필요

`omeye2.sc_monitoring.service`에 systemd 서비스 유닛 예시가 포함되어 있습니다.

## 8. 결합도 메모

- `util/common.go`가 패키지 전역 초기화 + `os.Args[1]`에 의존하여 부팅 결합도가 높습니다.
- 핸들러가 전역 공유 상태(`configs`, 경로, 비밀번호, logger, redis client)에 강하게 의존합니다.
- 대부분 함수 중심 호출 구조로 계층이 얕고 추상화 깊이는 낮습니다.
