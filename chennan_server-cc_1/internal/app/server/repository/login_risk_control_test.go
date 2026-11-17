package repository

import (
	"testing"
	"time"
)

func TestNextAccountPasswordCooldownEscalates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name              string
		failureCount      int
		currentLevel      string
		hadPriorCooldown  bool
		wantFailureCount  int
		wantLevel         string
		wantTriggered     bool
		wantCooldownAfter time.Duration
	}{
		{
			name:             "four failures keep watching",
			failureCount:     4,
			currentLevel:     loginRiskLevelLow,
			wantFailureCount: 4,
			wantLevel:        loginRiskLevelLow,
			wantTriggered:    false,
		},
		{
			name:              "fifth failure locks three minutes",
			failureCount:      5,
			currentLevel:      loginRiskLevelLow,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelLow,
			wantTriggered:     true,
			wantCooldownAfter: 3 * time.Minute,
		},
		{
			name:              "second cooldown locks ten minutes",
			failureCount:      5,
			currentLevel:      loginRiskLevelLow,
			hadPriorCooldown:  true,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelMedium,
			wantTriggered:     true,
			wantCooldownAfter: 10 * time.Minute,
		},
		{
			name:              "third cooldown locks one hour",
			failureCount:      5,
			currentLevel:      loginRiskLevelMedium,
			hadPriorCooldown:  true,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelHigh,
			wantTriggered:     true,
			wantCooldownAfter: 5 * time.Hour,
		},
		{
			name:              "max cooldown stays one hour",
			failureCount:      5,
			currentLevel:      loginRiskLevelHigh,
			hadPriorCooldown:  true,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelHigh,
			wantTriggered:     true,
			wantCooldownAfter: 5 * time.Hour,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotFailureCount, gotLevel, gotUntil, gotTriggered := nextAccountPasswordCooldown(tt.failureCount, tt.currentLevel, tt.hadPriorCooldown, now)
			if gotFailureCount != tt.wantFailureCount {
				t.Fatalf("failure count = %d, want %d", gotFailureCount, tt.wantFailureCount)
			}
			if gotLevel != tt.wantLevel {
				t.Fatalf("level = %q, want %q", gotLevel, tt.wantLevel)
			}
			if gotTriggered != tt.wantTriggered {
				t.Fatalf("triggered = %v, want %v", gotTriggered, tt.wantTriggered)
			}
			if tt.wantTriggered {
				if gotUntil.Sub(now) != tt.wantCooldownAfter {
					t.Fatalf("cooldown = %s, want %s", gotUntil.Sub(now), tt.wantCooldownAfter)
				}
			}
		})
	}
}

func TestNextAdminPasswordCooldownEscalates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name              string
		failureCount      int
		currentLevel      string
		hadPriorCooldown  bool
		wantFailureCount  int
		wantLevel         string
		wantTriggered     bool
		wantCooldownAfter time.Duration
	}{
		{
			name:             "second first-stage failure keeps watching",
			failureCount:     2,
			currentLevel:     loginRiskLevelLow,
			wantFailureCount: 2,
			wantLevel:        loginRiskLevelLow,
			wantTriggered:    false,
		},
		{
			name:              "third first-stage failure locks one hour",
			failureCount:      3,
			currentLevel:      loginRiskLevelLow,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelLow,
			wantTriggered:     true,
			wantCooldownAfter: time.Hour,
		},
		{
			name:             "first retry after cooldown keeps watching",
			failureCount:     1,
			currentLevel:     loginRiskLevelLow,
			hadPriorCooldown: true,
			wantFailureCount: 1,
			wantLevel:        loginRiskLevelLow,
			wantTriggered:    false,
		},
		{
			name:              "second cooldown locks one hour",
			failureCount:      2,
			currentLevel:      loginRiskLevelLow,
			hadPriorCooldown:  true,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelMedium,
			wantTriggered:     true,
			wantCooldownAfter: time.Hour,
		},
		{
			name:              "third cooldown locks three hours",
			failureCount:      2,
			currentLevel:      loginRiskLevelMedium,
			hadPriorCooldown:  true,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelHigh,
			wantTriggered:     true,
			wantCooldownAfter: 3 * time.Hour,
		},
		{
			name:              "max cooldown remains three hours",
			failureCount:      2,
			currentLevel:      loginRiskLevelHigh,
			hadPriorCooldown:  true,
			wantFailureCount:  0,
			wantLevel:         loginRiskLevelHigh,
			wantTriggered:     true,
			wantCooldownAfter: 3 * time.Hour,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotFailureCount, gotLevel, gotUntil, gotTriggered := nextAdminPasswordCooldown(tt.failureCount, tt.currentLevel, tt.hadPriorCooldown, now)
			if gotFailureCount != tt.wantFailureCount {
				t.Fatalf("failure count = %d, want %d", gotFailureCount, tt.wantFailureCount)
			}
			if gotLevel != tt.wantLevel {
				t.Fatalf("level = %q, want %q", gotLevel, tt.wantLevel)
			}
			if gotTriggered != tt.wantTriggered {
				t.Fatalf("triggered = %v, want %v", gotTriggered, tt.wantTriggered)
			}
			if tt.wantTriggered {
				if gotUntil.Sub(now) != tt.wantCooldownAfter {
					t.Fatalf("cooldown = %s, want %s", gotUntil.Sub(now), tt.wantCooldownAfter)
				}
			}
		})
	}
}

func TestNextAdminPasswordCooldownPreservesEscalationBetweenRetries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	nextCount, level, _, triggered := nextAdminPasswordCooldown(3, loginRiskLevelLow, false, now)
	if !triggered || nextCount != 0 || level != loginRiskLevelLow {
		t.Fatalf("first cooldown = count:%d level:%s triggered:%v", nextCount, level, triggered)
	}

	status, level, _ := nextAdminPasswordRisk(1, level, true)
	if status != loginRiskStatusWatching || level != loginRiskLevelLow {
		t.Fatalf("first retry after first cooldown = status:%s level:%s", status, level)
	}

	nextCount, level, _, triggered = nextAdminPasswordCooldown(2, level, true, now)
	if !triggered || nextCount != 0 || level != loginRiskLevelMedium {
		t.Fatalf("second cooldown = count:%d level:%s triggered:%v", nextCount, level, triggered)
	}

	status, level, _ = nextAdminPasswordRisk(1, level, true)
	if status != loginRiskStatusWatching || level != loginRiskLevelMedium {
		t.Fatalf("first retry after second cooldown should preserve medium, got status:%s level:%s", status, level)
	}

	nextCount, level, until, triggered := nextAdminPasswordCooldown(2, level, true, now)
	if !triggered || nextCount != 0 || level != loginRiskLevelHigh || until.Sub(now) != 3*time.Hour {
		t.Fatalf("third cooldown = count:%d level:%s cooldown:%s triggered:%v", nextCount, level, until.Sub(now), triggered)
	}
}

func TestIsPublicIPAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ipAddress string
		want      bool
	}{
		{name: "public ipv4", ipAddress: "212.107.28.51", want: true},
		{name: "public ipv6", ipAddress: "2406:2d40:80ee:b208:bdb9:4cef:3b48:f7f", want: true},
		{name: "private ipv4", ipAddress: "192.168.1.10", want: false},
		{name: "loopback", ipAddress: "127.0.0.1", want: false},
		{name: "docker gateway", ipAddress: "172.18.0.1", want: false},
		{name: "invalid", ipAddress: "not-an-ip", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isPublicIPAddress(tt.ipAddress); got != tt.want {
				t.Fatalf("isPublicIPAddress(%q) = %v, want %v", tt.ipAddress, got, tt.want)
			}
		})
	}
}
