package migrationfence

import (
	"errors"
	"os"
	"strings"
)

const (
	markerPathEnv     = "CHENGNAN_SOURCE_MIGRATION_FENCE_FILE"
	defaultMarkerPath = "/app/.chengnan-source-migration-fence.json"
)

// MarkerPath returns the runtime marker used by the source migration fence.
func MarkerPath() string {
	if value := strings.TrimSpace(os.Getenv(markerPathEnv)); value != "" {
		return value
	}
	return defaultMarkerPath
}

// Active fails closed for an unreadable or malformed marker path.
func Active() bool {
	_, err := os.Lstat(MarkerPath())
	if err == nil {
		return true
	}
	return !errors.Is(err, os.ErrNotExist)
}
