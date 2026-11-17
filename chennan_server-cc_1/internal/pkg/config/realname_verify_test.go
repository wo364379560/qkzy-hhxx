package config

import "testing"

func TestParseRealnameVerifyEnabled(t *testing.T) {
	t.Parallel()

	enabled := []string{"true", "TRUE", "1", "on", "yes", " t "}
	for _, value := range enabled {
		if !parseRealnameVerifyEnabled(value) {
			t.Fatalf("expected enabled for %q", value)
		}
	}

	disabled := []string{"", "false", "0", "off", "关闭", "no", "unexpected"}
	for _, value := range disabled {
		if parseRealnameVerifyEnabled(value) {
			t.Fatalf("expected disabled for %q", value)
		}
	}
}
