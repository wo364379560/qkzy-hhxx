package controller_admin

import "testing"

func TestNormalizeGlobalLoginPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty keeps existing password", input: "", want: ""},
		{name: "trims password", input: "  abcdefgh  ", want: "abcdefgh"},
		{name: "rejects short password", input: "abcdefg", wantErr: true},
		{name: "rejects long password", input: "abcdefghijklmnopqrstuvwxyz1234567", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeGlobalLoginPassword(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
