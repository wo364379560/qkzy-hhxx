package controller_chat

import (
	"testing"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/config"
)

func TestRegistrationBlockedMessageMatchesPolicyWindow(t *testing.T) {
	t.Parallel()

	policy := config.ChatQuietHoursPolicy{
		Enabled:          true,
		TimeZone:         "Asia/Shanghai",
		Start:            "00:00",
		End:              "08:00",
		BlockRegister:    true,
		BlockSendMessage: true,
	}

	blockedAt := time.Date(2026, 4, 23, 2, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	blocked, err := config.IsChatQuietHoursActionBlockedNow(policy, config.ChatQuietHoursActionRegister, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected register action to be blocked")
	}

	authErr := AuthErr(config.ChatQuietHoursBlockedMessage(config.ChatQuietHoursActionRegister, policy))
	if authErr.Error() != "当前处于系统静默时段（00:00-08:00），暂不开放新用户注册，请稍后再试" {
		t.Fatalf("unexpected registration blocked message: %q", authErr.Error())
	}
}
