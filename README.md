# OMS Monitoring Agent
원모어아이 선별관제 서버 모니터링 Application

## 1. 필수 패키지
- `ethtool` 패키지 설치
```bash
sudo apt install ethtool
```

## 2. 사용 언어
- Go 1.21.6 이상


## 3. 사용 방법

### 3.1. 빌드
```bash
./build.sh
```

빌드 시각을 바이너리에 주입하기 위해 스크립트를 사용한다. 기동 로그의 `Build Info:` 줄에서 확인할 수 있다.

직접 빌드하려면 `ldflags`로 빌드 시각을 넘긴다. 생략하면 빌드 시각 대신 마지막 커밋 시각(`vcs.time`)이 표시된다.
```bash
go build -ldflags "-X github.com/omssejong/reid-monitoring/util.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o oms-monitoring
```

### 3.2. conf.d 설정
모니터링 앱을 실행하기 전에 `conf.d` 디렉토리에 설정 파일을 서버 환경에 맞추어야 한다.
- `conf.d` 디렉토리 파일 리스트
    - `conf.d/config-dev.yml` : 개발용 설정 파일
    - `conf.d/config-docker.yml` : 도커용 설정 파일
    - `conf.d/config-local.yml` : 로컬용 설정 파일
    - `conf.d/config-prod.yml` : 운영용 설정 파일
    - `conf.d/omeye_monitoring.ini` : 모니터링 서비스 대상 설정 파일
    - `conf.d/docker-compose.yml` : 모니터링 서비스 대상 중 도커 컨테이너에 빌드된 어플리케이션에 대한 설정 파일


