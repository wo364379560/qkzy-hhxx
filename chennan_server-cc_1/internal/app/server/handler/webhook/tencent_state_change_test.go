package handler_webhook

import (
	"testing"
	"time"
)

func TestIsTencentOfflineAction(t *testing.T) {
	tests := []struct {
		action string
		want   bool
	}{
		{action: "Logout", want: true},
		{action: "Disconnect", want: true},
		{action: " logout ", want: true},
		{action: "Login", want: false},
		{action: "", want: false},
	}

	for _, tt := range tests {
		if got := isTencentOfflineAction(tt.action); got != tt.want {
			t.Fatalf("isTencentOfflineAction(%q)=%v want %v", tt.action, got, tt.want)
		}
	}
}

func TestTencentStateEventTime(t *testing.T) {
	seconds := int64(1720000000)
	if got := tencentStateEventTime(seconds); !got.Equal(time.Unix(seconds, 0)) {
		t.Fatalf("seconds event time=%s", got)
	}

	millis := int64(1720000000123)
	if got := tencentStateEventTime(millis); !got.Equal(time.UnixMilli(millis)) {
		t.Fatalf("millis event time=%s", got)
	}
}
