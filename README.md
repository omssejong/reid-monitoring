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
go build -mod=vendor -o oms-monitoring
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


