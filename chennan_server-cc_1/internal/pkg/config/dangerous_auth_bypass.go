package config

import (
	"os"
	"strings"
)

// dangerousAuthBypassEnabled is an explicit process-level kill switch for the
// legacy global IM login password. Universal SMS codes are controlled by the
// admin settings page and database, not this environment variable.
func dangerousAuthBypassEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DANGEROUS_AUTH_BYPASS_ENABLED"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
