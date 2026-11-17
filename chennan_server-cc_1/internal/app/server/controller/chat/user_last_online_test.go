package controller_chat

import (
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"testing"
	"time"
)

func TestLastOnlineCountsFromOfflineNotLogin(t *testing.T) {
	login := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	offline := login.Add(5 * time.Minute)
	got := userLastOnlineResult(model.User{IMUserID: "peer", LastOnlineAt: &login, LastOfflineAt: &offline})
	if got.LastOnlineAt != offline.Format(time.RFC3339) || got.LastOfflineAt != offline.Format(time.RFC3339) {
		t.Fatalf("expected offline timestamp for both new and legacy clients: %+v", got)
	}
	nextLogin := offline.Add(time.Hour)
	got = userLastOnlineResult(model.User{IMUserID: "peer", LastOnlineAt: &nextLogin, LastOfflineAt: &offline})
	if got.LastOnlineAt != nextLogin.Format(time.RFC3339) {
		t.Fatal("previous session's offline time replaced newer activity")
	}
	got = userLastOnlineResult(model.User{IMUserID: "peer"})
	if got.LastOnlineAt != "" || got.LastOfflineAt != "" {
		t.Fatal("unknown presence fabricated a timestamp")
	}
}
