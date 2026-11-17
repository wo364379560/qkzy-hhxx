package security

import (
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
)

func TestEvaluateAdminUserIPWhitelist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		admin   *model.AdminUser
		client  string
		rules   []model.AdminUserIPWhitelist
		want    bool
		wantErr bool
	}{
		{
			name:    "nil admin returns error",
			admin:   nil,
			client:  "127.0.0.1",
			wantErr: true,
		},
		{
			name: "super admin bypasses whitelist",
			admin: &model.AdminUser{
				Role:               "super_admin",
				IPWhitelistEnabled: true,
			},
			client: "127.0.0.1",
			want:   true,
		},
		{
			name: "manager with empty whitelist denied",
			admin: &model.AdminUser{
				Role:               "admin",
				IPWhitelistEnabled: true,
			},
			client: "127.0.0.1",
			want:   false,
		},
		{
			name: "manager with matching whitelist allowed",
			admin: &model.AdminUser{
				Role:               "admin",
				IPWhitelistEnabled: true,
			},
			client: "127.0.0.1",
			rules: []model.AdminUserIPWhitelist{
				{CIDR: "127.0.0.1/32"},
			},
			want: true,
		},
		{
			name: "assistant with matching whitelist allowed",
			admin: &model.AdminUser{
				Role:               "assistant",
				IPWhitelistEnabled: true,
			},
			client: "10.0.0.1",
			rules: []model.AdminUserIPWhitelist{
				{CIDR: "10.0.0.0/24"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := EvaluateAdminUserIPWhitelist(tc.admin, tc.client, tc.rules)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
