package config

import (
	"testing"
	"time"
)

func TestNormalizeChatQuietHoursPolicy(t *testing.T) {
	previous := Cfg.Get()
	defer Cfg.set(&previous)

	Cfg.set(&Options{
		System: System{
			TimeZone: "Asia/Shanghai",
		},
	})

	tests := []struct {
		name    string
		input   ChatQuietHoursPolicy
		wantErr string
		check   func(t *testing.T, policy ChatQuietHoursPolicy)
	}{
		{
			name:  "fills defaults",
			input: ChatQuietHoursPolicy{},
			check: func(t *testing.T, policy ChatQuietHoursPolicy) {
				t.Helper()
				if policy.TimeZone != "Asia/Shanghai" {
					t.Fatalf("expected timezone Asia/Shanghai, got %q", policy.TimeZone)
				}
				if policy.Start != defaultQuietHoursStart {
					t.Fatalf("expected start %q, got %q", defaultQuietHoursStart, policy.Start)
				}
				if policy.End != defaultQuietHoursEnd {
					t.Fatalf("expected end %q, got %q", defaultQuietHoursEnd, policy.End)
				}
			},
		},
		{
			name: "rejects invalid timezone",
			input: ChatQuietHoursPolicy{
				TimeZone: "No/SuchZone",
				Start:    "00:00",
				End:      "08:00",
			},
			wantErr: "时区配置无效",
		},
		{
			name: "rejects invalid start",
			input: ChatQuietHoursPolicy{
				TimeZone: "Asia/Shanghai",
				Start:    "24:00",
				End:      "08:00",
			},
			wantErr: "开始时间格式无效，请使用 HH:MM",
		},
		{
			name: "rejects same time",
			input: ChatQuietHoursPolicy{
				TimeZone: "Asia/Shanghai",
				Start:    "08:00",
				End:      "08:00",
			},
			wantErr: "开始时间和结束时间不能相同",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			policy := tc.input
			err := normalizeChatQuietHoursPolicy(&policy)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, policy)
			}
		})
	}
}

func TestIsChatQuietHoursActionBlockedNow(t *testing.T) {
	t.Parallel()

	policy := ChatQuietHoursPolicy{
		Enabled:          true,
		TimeZone:         "Asia/Shanghai",
		Start:            "00:00",
		End:              "08:00",
		BlockRegister:    true,
		BlockSendMessage: true,
		BlockCreateGroup: true,
		BlockAddFriend:   true,
	}

	blockedAt := time.Date(2026, 4, 23, 2, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	blocked, err := IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionRegister, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected register action to be blocked")
	}

	blocked, err = IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionSendMessage, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected send_message action to be blocked")
	}

	blocked, err = IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionCreateGroup, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected create_group action to be blocked")
	}

	blocked, err = IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionApplyJoinGroup, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected apply_join_group action to be blocked")
	}

	blocked, err = IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionInviteJoinGroup, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected invite_join_group action to be blocked")
	}

	blocked, err = IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionAddFriend, blockedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Fatal("expected add_friend action to be blocked")
	}

	allowedAt := time.Date(2026, 4, 23, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	blocked, err = IsChatQuietHoursActionBlockedNow(policy, ChatQuietHoursActionSendMessage, allowedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocked {
		t.Fatal("expected send_message action to be allowed after quiet hours")
	}
}

func TestChatQuietHoursBlockedMessageForSendMessage(t *testing.T) {
	t.Parallel()

	policy := ChatQuietHoursPolicy{
		Start: "00:00",
		End:   "08:30",
	}

	got := ChatQuietHoursBlockedMessage(ChatQuietHoursActionSendMessage, policy)
	want := "当前处于静默时段（00:00-08:30），无法发送消息，请于08:30之后重试"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
