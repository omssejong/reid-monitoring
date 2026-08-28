//go:build linux

package util

import (
	"syscall"
)

// diskMeasurePath 디스크 사용률/용량 측정 기준 경로.
// 리눅스에서는 루트 파티션을 기준으로 한다.
const diskMeasurePath = "/"

// diskStats statfs(2) 로 파티션 용량을 조회한다.
// df 와 동일하게 used 는 bfree, avail 은 bavail 기준으로 계산한다.
func diskStats(path string) (total uint64, avail uint64, used uint64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, err
	}

	bsize := uint64(stat.Bsize)
	total = stat.Blocks * bsize
	avail = stat.Bavail * bsize
	used = (stat.Blocks - stat.Bfree) * bsize
	return total, avail, used, nil
}
