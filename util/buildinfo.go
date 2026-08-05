package util

import (
	"fmt"
	"runtime/debug"
	"time"
)

// kstZone 한국 표준시. 서머타임이 없으므로 고정 오프셋으로 둔다.
// time.LoadLocation("Asia/Seoul")은 실행 서버의 tzdata에 의존하므로 쓰지 않는다.
var kstZone = time.FixedZone("KST", 9*60*60)

// buildTime 빌드 시각(RFC3339, UTC). 빌드할 때 ldflags로 주입한다.
//
//	go build -ldflags "-X github.com/omssejong/reid-monitoring/util.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o oms-monitoring
//
// 주입하지 않으면 빈 값이며, 이 경우 git 커밋 시각(vcs.time)으로 대체한다.
// 커밋 시각은 빌드 시각이 아니므로 Source로 구분해서 알린다.
var buildTime string

// BuildInfo 바이너리 식별 정보
type BuildInfo struct {
	// BuildTime 빌드 시각 (KST, RFC3339). 주입되지 않았으면 커밋 시각, 그것도 없으면 "unknown".
	// 주입/자동 기록 값은 UTC이므로 KST로 변환해서 담는다.
	BuildTime string `json:"buildTime"`

	// Source BuildTime의 출처: "ldflags"(실제 빌드 시각) | "vcs"(커밋 시각) | "unknown"
	Source string `json:"source"`

	// Revision git 커밋 해시. Modified는 커밋되지 않은 변경이 섞인 빌드인지 여부.
	Revision string `json:"revision"`
	Modified bool   `json:"modified"`
}

// GetBuildInfo 빌드 시각과 git 리비전 반환.
// ldflags로 주입된 빌드 시각을 우선하고, 없으면 Go 툴체인이 자동으로 박아둔 커밋 시각을 쓴다.
func GetBuildInfo() BuildInfo {
	info := BuildInfo{BuildTime: buildTime, Source: "ldflags"}

	var vcsTime string
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.time":
				vcsTime = s.Value
			case "vcs.revision":
				info.Revision = s.Value
			case "vcs.modified":
				info.Modified = s.Value == "true"
			}
		}
	}

	if info.BuildTime == "" {
		info.BuildTime = vcsTime
		info.Source = "vcs"
	}
	if info.BuildTime == "" {
		info.Source = "unknown"
		info.BuildTime = "unknown"
		return info
	}

	info.BuildTime = toKST(info.BuildTime)
	return info
}

// toKST RFC3339 시각 문자열을 KST 기준으로 변환한다.
// 파싱하지 못하면 원본을 그대로 반환한다 (표시가 깨질지언정 값을 잃지 않는다).
func toKST(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return formatKST(t)
}

// formatKST 시각을 KST 기준 RFC3339 문자열로 만든다. 로그 표기를 KST로 통일하는 데 쓴다.
func formatKST(t time.Time) string {
	return t.In(kstZone).Format(time.RFC3339)
}

// String 로그 출력용 한 줄 요약
func (b BuildInfo) String() string {
	revision := b.Revision
	if revision == "" {
		revision = "unknown"
	} else if len(revision) > 12 {
		revision = revision[:12]
	}

	dirty := ""
	if b.Modified {
		dirty = " (dirty)"
	}

	return fmt.Sprintf("buildTime=%s source=%s revision=%s%s", b.BuildTime, b.Source, revision, dirty)
}
