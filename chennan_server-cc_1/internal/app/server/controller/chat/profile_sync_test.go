package controller_chat

import (
	"errors"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"testing"
)

func TestSavedProfileRequiresConfirmedIMSync(t *testing.T) {
	items := []tencentim.ProfileItemDetail{{Tag: tencentim.TagImage, Value: "https://example.invalid/avatar.jpg"}}
	failure := errors.New("IM transport unavailable")
	err := syncSavedProfile("alice", "already-uploaded-avatar", items, true, func(string, []tencentim.ProfileItemDetail) error { return failure })
	var partial *ProfileSyncError
	if !errors.As(err, &partial) || partial.AvatarURL != "already-uploaded-avatar" || !errors.Is(err, failure) {
		t.Fatal("lost retry-only-sync state", err)
	}
	calls := 0
	if err := syncSavedProfile("alice", partial.AvatarURL, items, true, func(string, []tencentim.ProfileItemDetail) error { calls++; return nil }); err != nil || calls != 1 {
		t.Fatal("sync-only retry failed", err)
	}
	if err := syncSavedProfile("alice", "avatar", items, false, nil); err == nil {
		t.Fatal("unconfigured IM reported success")
	}
}
