package controller_chat

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
)

// Real FOR UPDATE/commit behavior, using only the opt-in disposable loopback DB.
func TestRefreshConcurrencyAndRevocationMySQL(t *testing.T) {
	db := serviceOwnerDB(t)
	configureRefreshTest(t)
	if err := db.AutoMigrate(&model.UserRefreshSession{}); err != nil {
		t.Fatal(err)
	}
	session := model.UserRefreshSession{
		UserID: 201, RefreshTokenHash: security.HashRefreshToken("isolated-initial-token"),
		DeviceID: "isolated-device", LastUsedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	req := RefreshReq{
		RefreshToken: "isolated-initial-token", SDKAppID: 123,
		UserSigExpiresAt: time.Now().Add(72 * time.Hour).Format(time.RFC3339),
		SessionMeta:      SessionMeta{DeviceID: "isolated-device"},
	}
	type reply struct {
		result *AuthResult
		err    error
	}
	replies := make(chan reply, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := refreshSessionWithDB(context.Background(), db, req)
			replies <- reply{result, err}
		}()
	}
	close(start)
	wg.Wait()
	close(replies)
	var rotated string
	for answer := range replies {
		if answer.err != nil {
			t.Fatal(answer.err)
		}
		if answer.result == nil || answer.result.Token == "" || answer.result.RefreshToken == "" {
			t.Fatal("missing credentials")
		}
		if rotated != "" && rotated != answer.result.RefreshToken {
			t.Fatal("concurrent refresh rotated twice")
		}
		rotated = answer.result.RefreshToken
	}
	var stored model.UserRefreshSession
	if err := db.First(&stored, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RefreshTokenHash != security.HashRefreshToken(rotated) || stored.PreviousRefreshTokenHash != session.RefreshTokenHash {
		t.Fatal("rotation/grace snapshot corrupted")
	}
	expiry, grace := stored.ExpiresAt, *stored.PreviousRefreshExpiresAt
	result, err := refreshSessionWithDB(context.Background(), db, req)
	if err != nil || result.RefreshToken != rotated {
		t.Fatalf("lost-response recovery failed: %v", err)
	}
	if err := db.First(&stored, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !stored.ExpiresAt.Equal(expiry) || !stored.PreviousRefreshExpiresAt.Equal(grace) {
		t.Fatal("retry extended grace or expiry")
	}
	// A current token from a wrong device must commit revocation, not roll it back.
	wrong := req
	wrong.RefreshToken = rotated
	wrong.DeviceID = "other-device"
	if result, err := refreshSessionWithDB(context.Background(), db, wrong); result != nil || err == nil || DescribeRefreshFailure(err).ErrorCode != "SESSION_DEVICE_MISMATCH" {
		t.Fatalf("wrong device classification: %v", err)
	}
	if err := db.First(&stored, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RevokedAt == nil {
		t.Fatal("device mismatch revocation rolled back")
	}
	req.RefreshToken = rotated
	if _, err := refreshSessionWithDB(context.Background(), db, req); err == nil || DescribeRefreshFailure(err).ErrorCode != "SESSION_REVOKED" {
		t.Fatalf("revoked session recovered: %v", err)
	}
	// Explicit administrator/password revocation remains authoritative as well.
	fresh := session
	fresh.ID = 0
	fresh.RefreshTokenHash = security.HashRefreshToken("isolated-new-login")
	if err := db.Create(&fresh).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.NewUserRefreshSessionRepositoryWithDB(db).RevokeAllByUserID(201); err != nil {
		t.Fatal(err)
	}
	req.RefreshToken = "isolated-new-login"
	if _, err := refreshSessionWithDB(context.Background(), db, req); err == nil || DescribeRefreshFailure(err).ErrorCode != "SESSION_REVOKED" {
		t.Fatalf("explicit revoke ineffective: %v", err)
	}
}
