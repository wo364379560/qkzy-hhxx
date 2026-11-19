package controller_admin

import "testing"

func TestUserResetPasswordRequiresServerConfiguration(t *testing.T) {
	if _, err := resolveUserResetPassword(ResetUserPasswordReq{Mode: "default"}, ""); err == nil {
		t.Fatal("unset default must fail closed")
	}
	if _, err := resolveUserResetPassword(ResetUserPasswordReq{Mode: "default"}, "short"); err == nil {
		t.Fatal("invalid configuration accepted")
	}
	configured := "test-only-default"
	if got, err := resolveUserResetPassword(ResetUserPasswordReq{Mode: "default"}, configured); err != nil || got != configured {
		t.Fatal("server configuration not used")
	}
	explicit := "test-only-explicit"
	if got, err := resolveUserResetPassword(ResetUserPasswordReq{Mode: "set", NewPassword: &explicit}, ""); err != nil || got != explicit {
		t.Fatal("explicit reset must remain available")
	}
}
