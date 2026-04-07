package util

import (
	"context"
	"fmt"
	"os"
	"time"
)

func StartDataCleanup(ctx context.Context, path string) {
	retention := time.Duration(configs.SC.Setting.ZipRetentionHours) * time.Hour
	if retention <= 0 {
		retention = time.Hour
	}

	interval := time.Duration(configs.SC.Setting.ZipCleanupIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	runCleanup := func() {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return
			}
			log.Error(fmt.Errorf("cleanup path stat failed: %w", err))
			return
		}

		if err := DeleteFilesAnHours(path, retention); err != nil {
			log.Error(fmt.Errorf("data cleanup failed: %w", err))
			return
		}

		log.Info(fmt.Sprintf("data cleanup completed: path=%s retention=%s", path, retention))
	}

	runCleanup()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("data cleanup stopped")
			return
		case <-ticker.C:
			runCleanup()
		}
	}
}
