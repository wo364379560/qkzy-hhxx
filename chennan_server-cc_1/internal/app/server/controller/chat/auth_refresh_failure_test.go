package controller_chat

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
)

func TestRefreshRejectionClassificationAndRevocationCommit(t *testing.T) {
	configureRefreshTest(t)
	for _, kind := range []string{"revoked", "expired", "device_mismatch", "lookup_failure", "not_found"} {
		t.Run(kind, func(t *testing.T) {
			db, mock := refreshTestDB(t)
			session := recoverySession(t)
			req := RefreshReq{RefreshToken: "current-token", SessionMeta: SessionMeta{DeviceID: "device"}}
			now := time.Now()
			want := ""
			mock.ExpectBegin()
			query := mock.ExpectQuery("SELECT .* FROM `user_refresh_sessions`.*FOR UPDATE")
			switch kind {
			case "revoked":
				session.RevokedAt = &now
				query.WillReturnRows(sessionRows(session))
				mock.ExpectCommit()
				want = "SESSION_REVOKED"
			case "expired":
				session.ExpiresAt = now.Add(-time.Hour)
				query.WillReturnRows(sessionRows(session))
				mock.ExpectExec("UPDATE `user_refresh_sessions`").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				want = "SESSION_EXPIRED"
			case "device_mismatch":
				req.DeviceID = "other-device"
				query.WillReturnRows(sessionRows(session))
				mock.ExpectExec("UPDATE `user_refresh_sessions`").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				want = "SESSION_DEVICE_MISMATCH"
			case "lookup_failure":
				query.WillReturnError(errors.New("temporary database outage"))
				mock.ExpectRollback()
				want = "SESSION_TEMPORARY"
			case "not_found":
				query.WillReturnRows(sqlmock.NewRows([]string{"id"}))
				mock.ExpectRollback()
				want = "SESSION_INVALID"
			}
			result, err := refreshSessionWithDB(context.Background(), db, req)
			if result != nil || err == nil {
				t.Fatalf("rejection issued credentials or no error")
			}
			details := DescribeRefreshFailure(err)
			if details.ErrorCode != want || details.Retryable != (kind == "lookup_failure") || details.Stage == "" {
				t.Fatalf("classification=%+v", details)
			}
		})
	}
}

func TestRefreshTemporaryErrorsStayTemporaryAtEveryStage(t *testing.T) {
	for _, stage := range []string{"session_lookup", "rotation_recovery", "device_binding", "account_policy", "issue_access_token", "issue_user_sig", "rotate_session", "commit"} {
		cause := errors.New("sensitive database detail")
		err := classifyRefreshFailure(cause, stage)
		details := DescribeRefreshFailure(err)
		if !errors.Is(err, cause) || details.ErrorCode != "SESSION_TEMPORARY" || !details.Retryable {
			t.Fatalf("%s: %+v", stage, details)
		}
	}
	for _, stage := range []string{"rotation_recovery", "account_policy"} {
		if details := DescribeRefreshFailure(classifyRefreshFailure(AuthErr("rejected"), stage)); details.Retryable {
			t.Fatalf("explicit auth rejection became retryable: %+v", details)
		}
	}
}
