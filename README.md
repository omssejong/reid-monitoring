# OMS Monitoring Agent
원모어아이 선별관제 서버 모니터링 Application

## 1. 지원 플랫폼 / 필수 패키지
Linux, Windows 를 모두 지원한다. 플랫폼 의존 코드는 빌드 태그로 분리되어 있다(`*_linux.go` / `*_windows.go`).

- Linux
  - `ethtool` 패키지 설치
    ```bash
    sudo apt install ethtool
    ```
  - 서비스 제어에 `systemctl`, `bash` 필요 (root 권한)
- Windows
  - 별도 설치 패키지 없음 (Windows API 직접 사용)
  - GPU 지표는 NVIDIA 드라이버가 설치하는 `nvml.dll` 이 있어야 수집된다
  - 서비스 start/stop/restart, 재부팅/종료는 관리자 권한으로 실행해야 한다

## 2. 사용 언어
- Go 1.21.6 이상


## 3. 사용 방법

### 3.1. 빌드
```bash
./build.sh                        # linux/amd64  → oms-monitoring
./build.sh "" windows             # windows/amd64 → oms-monitoring.exe
./build.sh my-binary              # 이름 지정
```

윈도우 개발 장비에서는 PowerShell 스크립트를 쓸 수 있다.
```powershell
.\build.ps1                       # windows/amd64 → oms-monitoring.exe
.\build.ps1 -TargetOS linux       # linux/amd64   → oms-monitoring
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

#### Windows 설정 시 주의
- `networkName` : 어댑터 표시 이름을 넣는다 (예: `이더넷 2`). `Get-NetAdapter` 로 확인 가능
- `rootPath`, `logPath`, `storageRootDir` : Windows 경로로 지정한다 (예: `C:/omeye`)
- 서비스 이름은 Windows 서비스 이름을 쓴다. `.service` 접미사가 붙어 있으면 자동으로 제거된다
- `middleserver` 재시작(`virsh` 사용)은 Windows 에서 지원하지 않는다


