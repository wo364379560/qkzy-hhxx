package controller_chat

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func refreshTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		conn.Close()
	})
	return db, mock
}
func configureRefreshTest(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-only-refresh-recovery-key-20261026")
	t.Setenv("TENCENT_IM_SDK_APP_ID", "123")
	if err := config.Setup(); err != nil {
		t.Fatal(err)
	}
}
func recoverySession(t *testing.T) model.UserRefreshSession {
	t.Helper()
	now := time.Now()
	grace := now.Add(time.Minute)
	ciphertext, err := security.SealRefreshToken("current-token", config.Cfg.Get().JWT.Secret, refreshRecoveryBinding(7, "device"))
	if err != nil {
		t.Fatal(err)
	}
	return model.UserRefreshSession{ID: 7, UserID: 3, DeviceID: "device", RefreshTokenHash: security.HashRefreshToken("current-token"), PreviousRefreshTokenHash: security.HashRefreshToken("old-token"), PreviousRefreshExpiresAt: &grace, RefreshTokenCiphertext: ciphertext, ExpiresAt: now.Add(time.Hour)}
}
func sessionRows(s model.UserRefreshSession) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "device_id", "refresh_token_hash", "previous_refresh_token_hash", "previous_refresh_expires_at", "refresh_token_ciphertext", "expires_at", "revoked_at", "created_at"}).AddRow(s.ID, s.UserID, s.DeviceID, s.RefreshTokenHash, s.PreviousRefreshTokenHash, s.PreviousRefreshExpiresAt, s.RefreshTokenCiphertext, s.ExpiresAt, s.RevokedAt, s.CreatedAt)
}

func TestRefreshLostResponseReturnsSameTokenWithoutAnotherRotation(t *testing.T) {
	configureRefreshTest(t)
	for _, token := range []string{"old-token", "current-token"} {
		t.Run(token, func(t *testing.T) {
			db, mock := refreshTestDB(t)
			session := recoverySession(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT .* FROM `user_refresh_sessions`.*FOR UPDATE").WillReturnRows(sessionRows(session))
			mock.ExpectQuery("SELECT .* FROM `users`").WillReturnRows(sqlmock.NewRows([]string{"id", "im_user_id", "owner_id", "status", "auth_version"}).AddRow(3, "u_test", 0, "active", 1))
			// No UPDATE is expected: neither token rotation nor grace/expiry extension.
			mock.ExpectCommit()
			result, err := refreshSessionWithDB(context.Background(), db, RefreshReq{RefreshToken: token, SDKAppID: 123, UserSigExpiresAt: time.Now().Add(72 * time.Hour).Format(time.RFC3339), SessionMeta: SessionMeta{DeviceID: "device"}})
			if err != nil {
				t.Fatal(err)
			}
			if result.RefreshToken != "current-token" || result.Token == "" || result.UserSig != "" {
				t.Fatalf("unexpected recovered response: token=%v userSig=%v", result.RefreshToken != "", result.UserSig != "")
			}
		})
	}
}

func TestPreviousRefreshTokenRequiresBoundDeviceAndLiveSession(t *testing.T) {
	configureRefreshTest(t)
	for _, name := range []string{"missing device", "different device", "expired grace", "revoked", "expired session", "older rotation"} {
		t.Run(name, func(t *testing.T) {
			s := recoverySession(t)
			now := time.Now()
			device := "device"
			hash := s.PreviousRefreshTokenHash
			switch name {
			case "missing device":
				device = ""
			case "different device":
				device = "other"
			case "expired grace":
				past := now.Add(-time.Second)
				s.PreviousRefreshExpiresAt = &past
			case "revoked":
				s.RevokedAt = &now
			case "expired session":
				s.ExpiresAt = now.Add(-time.Second)
			case "older rotation":
				hash = security.HashRefreshToken("older-token")
			}
			_, err := recoverRotatedRefreshToken(&s, hash, device, now)
			var authErr AuthErr
			if !errors.As(err, &authErr) {
				t.Fatalf("want definitive rejection, got %v", err)
			}
		})
	}
}

func TestRefreshOwnerDatabaseFailureIsNotAuthenticationFailure(t *testing.T) {
	db, mock := refreshTestDB(t)
	connectionErr := errors.New("database temporarily unavailable")
	mock.ExpectQuery("SELECT .* FROM `admin_users`").WillReturnError(connectionErr)
	err := ensureOwnerActiveWithDB(db, 5)
	var authErr AuthErr
	if !errors.Is(err, connectionErr) || errors.As(err, &authErr) {
		t.Fatalf("transient error became invalid session: %v", err)
	}
}

func TestRefreshRecoveryRejectsTamperedStoredToken(t *testing.T) {
	configureRefreshTest(t)
	s := recoverySession(t)
	s.RefreshTokenHash = security.HashRefreshToken("unrelated-current")
	if _, err := recoverRotatedRefreshToken(&s, s.PreviousRefreshTokenHash, "device", time.Now()); err == nil {
		t.Fatal("accepted mismatching current token")
	}
}

func TestDelayedLogoutCannotRevokeNewLoginAfterOldSessionWasRevoked(t *testing.T) {
	configureRefreshTest(t)
	db, mock := refreshTestDB(t)
	old := recoverySession(t)
	now := time.Now()
	old.RevokedAt = &now
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM `user_refresh_sessions`.*FOR UPDATE").WillReturnRows(sessionRows(old))
	// A subsequent login has already revoked the old session. There must be no
	// UPDATE to any sessions or device rows, even with a matching client device.
	mock.ExpectCommit()
	if err := logoutWithDB(context.Background(), db, "current-token", "device"); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutThroughGraceOnlyTouchesSessionsThroughItsOwnID(t *testing.T) {
	configureRefreshTest(t)
	db, mock := refreshTestDB(t)
	old := recoverySession(t)
	old.CreatedAt = time.Now().Add(-time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM `user_refresh_sessions`.*previous_refresh_token_hash.*FOR UPDATE").WithArgs(security.HashRefreshToken("old-token"), security.HashRefreshToken("old-token"), sqlmock.AnyArg(), 1).WillReturnRows(sessionRows(old))
	mock.ExpectExec("UPDATE `user_refresh_sessions` SET .* WHERE id = \\? AND revoked_at IS NULL").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `user_refresh_sessions` SET .* WHERE user_id = \\? AND device_id = \\? AND id <= \\? AND revoked_at IS NULL").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint(3), "device", uint(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `user_devices` SET .* WHERE \\(user_id = \\? AND device_id = \\? AND last_login_at <= \\?\\) AND NOT EXISTS \\(SELECT 1 FROM `user_refresh_sessions` WHERE user_id = \\? AND device_id = \\? AND id > \\?\\)").WithArgs("offline", sqlmock.AnyArg(), uint(3), "device", old.CreatedAt, uint(3), "device", uint(7)).WillReturnResult(sqlmock.NewResult(0, 0))
	expectLogoutPresenceWrite(mock, old.UserID, 0)
	mock.ExpectCommit()
	if err := logoutWithDB(context.Background(), db, "old-token", "device"); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyLogoutWithClaimedDeviceStillRevokesItsUnboundSession(t *testing.T) {
	configureRefreshTest(t)
	db, mock := refreshTestDB(t)
	old := recoverySession(t)
	old.DeviceID = ""
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM `user_refresh_sessions`.*FOR UPDATE").WillReturnRows(sessionRows(old))
	mock.ExpectExec("UPDATE `user_refresh_sessions` SET .* WHERE id = \\? AND revoked_at IS NULL").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `user_refresh_sessions` SET .* WHERE user_id = \\? AND device_id = \\? AND id <= \\? AND revoked_at IS NULL").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint(3), "device", uint(7)).WillReturnResult(sqlmock.NewResult(0, 0))
	expectLogoutPresenceWrite(mock, old.UserID, 1)
	mock.ExpectCommit()
	if err := logoutWithDB(context.Background(), db, "current-token", "device"); err != nil {
		t.Fatal(err)
	}
}

// The conditional write also protects another device/new login: only the last
// unrevoked, unexpired session may transition the account to offline.
func expectLogoutPresenceWrite(mock sqlmock.Sqlmock, userID uint, affected int64) {
	mock.ExpectExec("UPDATE `users` SET .*last_offline_at.*last_online_at.*last_presence_checked_at.*last_presence_status.* WHERE \\(id = \\? AND status = \\?\\) AND \\(NOT EXISTS \\(SELECT 1 FROM user_refresh_sessions.*user_id = \\? AND revoked_at IS NULL AND expires_at > \\?\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "offline", sqlmock.AnyArg(), userID, "active", userID, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, affected))
}

type logoutTimeAfter struct{ after time.Time }

func (a logoutTimeAfter) Match(value driver.Value) bool {
	at, ok := value.(time.Time)
	return ok && !at.Before(a.after) && !at.After(time.Now().Add(time.Second))
}

func TestLogoutAfterSixMinutesRecordsExitTimeNotLoginTime(t *testing.T) {
	configureRefreshTest(t)
	db, mock := refreshTestDB(t)
	session := recoverySession(t)
	session.CreatedAt = time.Now().Add(-6 * time.Minute)
	session.DeviceID = ""
	started := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM `user_refresh_sessions`.*FOR UPDATE").WillReturnRows(sessionRows(session))
	mock.ExpectExec("UPDATE `user_refresh_sessions` SET .* WHERE id = \\? AND revoked_at IS NULL").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `users` SET .*last_offline_at.*last_online_at.*last_presence_checked_at.*last_presence_status.*NOT EXISTS.*revoked_at IS NULL AND expires_at >").
		WithArgs(logoutTimeAfter{started}, logoutTimeAfter{started}, logoutTimeAfter{started}, "offline", sqlmock.AnyArg(), session.UserID, "active", session.UserID, logoutTimeAfter{started}).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := logoutWithDB(context.Background(), db, "current-token", ""); err != nil {
		t.Fatal(err)
	}
}
