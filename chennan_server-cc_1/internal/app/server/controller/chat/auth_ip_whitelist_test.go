package controller_chat

import (
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
)

func TestEnsureUserIPWhitelistAllowed(t *testing.T) {
	tests := []struct {
		name      string
		user      *model.User
		clientIP  string
		wantError string
	}{
		{
			name:     "disabled whitelist passes",
			user:     &model.User{IPWhitelistEnabled: false},
			clientIP: "203.0.113.10",
		},
		{
			name:     "matching whitelist passes",
			user:     &model.User{IPWhitelistEnabled: true, IPWhitelistIP: "203.0.113.10\n203.0.113.11"},
			clientIP: "203.0.113.10",
		},
		{
			name:      "denied whitelist includes normalized client ip",
			user:      &model.User{IPWhitelistEnabled: true, IPWhitelistIP: "203.0.113.10"},
			clientIP:  "203.0.113.12",
			wantError: "当前登录 IP（203.0.113.12）不在白名单内，请联系管理员",
		},
		{
			name:      "unknown client ip",
			user:      &model.User{IPWhitelistEnabled: true, IPWhitelistIP: "203.0.113.10"},
			clientIP:  "",
			wantError: "当前登录 IP 无法识别，请稍后重试",
		},
		{
			name:      "invalid whitelist config",
			user:      &model.User{IPWhitelistEnabled: true, IPWhitelistIP: "not-an-ip"},
			clientIP:  "203.0.113.10",
			wantError: "账号白名单配置不完整，请联系管理员",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureUserIPWhitelistAllowed(tc.user, tc.clientIP)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantError)
			}
			if got := err.Error(); got != tc.wantError {
				t.Fatalf("unexpected error: got %q want %q", got, tc.wantError)
			}
		})
	}
}
