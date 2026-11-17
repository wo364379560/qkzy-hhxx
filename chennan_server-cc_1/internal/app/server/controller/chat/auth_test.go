package controller_chat

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAuthResultIncludesSDKAppID(t *testing.T) {
	payload, err := json.Marshal(AuthResult{
		UserID:   "u_example",
		SDKAppID: 123456,
	})
	if err != nil {
		t.Fatalf("marshal auth result: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal auth result: %v", err)
	}
	if got := int(result["sdk_app_id"].(float64)); got != 123456 {
		t.Fatalf("unexpected sdk_app_id: %d", got)
	}
}

func TestAuthResultDoesNotOmitSDKAppID(t *testing.T) {
	payload, err := json.Marshal(AuthResult{UserID: "u_example"})
	if err != nil {
		t.Fatalf("marshal auth result: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal auth result: %v", err)
	}
	if _, ok := result["sdk_app_id"]; !ok {
		t.Fatal("sdk_app_id must never be omitted")
	}
}

func TestShouldIssueUserSigFor(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)
	currentSDKAppID := 123456
	farExpiry := now.Add(72 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name string
		req  RefreshReq
		want bool
	}{
		{
			name: "explicit refresh",
			req: RefreshReq{
				NeedUserSigRefresh: true,
				SDKAppID:           currentSDKAppID,
				UserSigExpiresAt:   farExpiry,
			},
			want: true,
		},
		{
			name: "sdk mismatch",
			req: RefreshReq{
				SDKAppID:         currentSDKAppID - 1,
				UserSigExpiresAt: farExpiry,
			},
			want: true,
		},
		{
			name: "missing expiry fails closed",
			req:  RefreshReq{SDKAppID: currentSDKAppID},
			want: true,
		},
		{
			name: "invalid expiry fails closed",
			req: RefreshReq{
				SDKAppID:         currentSDKAppID,
				UserSigExpiresAt: "invalid",
			},
			want: true,
		},
		{
			name: "same sdk with fresh sig",
			req: RefreshReq{
				SDKAppID:         currentSDKAppID,
				UserSigExpiresAt: farExpiry,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIssueUserSigFor(tt.req, currentSDKAppID, now, 24*time.Hour); got != tt.want {
				t.Fatalf("shouldIssueUserSigFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoginRejectsCodeLogin(t *testing.T) {
	t.Parallel()

	_, err := Login(context.Background(), LoginReq{
		Phone: "13012345678",
		Code:  "123456",
	})
	if err == nil {
		t.Fatal("expected code login to be rejected")
	}
	if err.Error() != "验证码登录已关闭，请使用密码登录" {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestSendSMSRejectsLoginScene(t *testing.T) {
	t.Parallel()

	err := SendSMS(context.Background(), SendSMSReq{
		Phone: "13012345678",
		Scene: "login",
	})
	if err == nil {
		t.Fatal("expected login scene SMS to be rejected")
	}
	if err.Error() != "验证码登录已关闭，请使用密码登录" {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestSendSMSRejectsInvalidPhone(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"   ",
		"abc",
		"123",
		"23012345678",
		"1301234567a",
	}
	for _, phone := range tests {
		phone := phone
		t.Run(phone, func(t *testing.T) {
			t.Parallel()

			err := SendSMS(context.Background(), SendSMSReq{
				Phone: phone,
				Scene: "register",
			})
			if err == nil {
				t.Fatal("expected invalid phone to be rejected")
			}
			if err.Error() != "手机号格式错误" {
				t.Fatalf("unexpected error: %q", err.Error())
			}
		})
	}
}
