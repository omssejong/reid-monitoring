#
# build.ps1
# 윈도우 개발 장비에서 빌드 시각을 주입해 실행 파일을 만든다. (build.sh 의 PowerShell 판)
#
#   .\build.ps1                              -> oms-monitoring.exe (windows/amd64)
#   .\build.ps1 -Output my-binary.exe        -> 지정한 이름으로 생성
#   .\build.ps1 -TargetOS linux              -> oms-monitoring     (linux/amd64)
#
param(
    [string]$Output = "",
    [string]$TargetOS = "windows",
    [string]$TargetArch = "amd64"
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Output)) {
    if ($TargetOS -eq "windows") { $Output = "oms-monitoring.exe" } else { $Output = "oms-monitoring" }
}

$buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$pkg = "github.com/omssejong/reid-monitoring/util"

$env:GOOS = $TargetOS
$env:GOARCH = $TargetArch
go build -ldflags "-X $pkg.buildTime=$buildTime" -o $Output
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Output "[OK] $Output ($TargetOS/$TargetArch, buildTime=$buildTime)"
