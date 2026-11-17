// Package messagelimit preserves old call sites after retiring the unverified chat quota.
package messagelimit

import (
	"context"
	"github.com/ixayda/ix-Go/internal/app/server/model"
)

type Decision struct {
	Blocked bool
	Reason  string
	Count   int64
	Limit   int64
}

// CheckBeforeSend no longer gates chat on real-name status or historical counts.
// Keep this compatibility entry point so an older caller cannot re-enable the quota.
func CheckBeforeSend(context.Context, *model.User) (Decision, error) {
	return Decision{}, nil
}

// RecordSuccessfulSend no longer reads or writes the retired quota counters.
func RecordSuccessfulSend(context.Context, *model.User) error { return nil }

// InvalidateRealNameVerified is retained for existing real-name management callers.
// Chat no longer maintains a real-name quota cache; verification itself is unchanged.
func InvalidateRealNameVerified(context.Context, uint) {}
