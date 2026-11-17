package server

import "testing"

func TestReadOnlyStandbyEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false},
		{name: "zero", value: "0", want: false},
		{name: "false", value: "false", want: false},
		{name: "true", value: "true", want: true},
		{name: "one", value: "1", want: true},
		{name: "yes with spaces", value: " YES ", want: true},
		{name: "on mixed case", value: "On", want: true},
		{name: "invalid fails closed", value: "standby", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CHENGNAN_STANDBY_READ_ONLY", tt.value)
			if got := readOnlyStandbyEnabled(); got != tt.want {
				t.Fatalf("readOnlyStandbyEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
