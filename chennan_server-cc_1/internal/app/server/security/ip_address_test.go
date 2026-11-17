package security

import "testing"

func TestNormalizeSingleIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "trim ipv4",
			input: " 1.2.3.4 ",
			want:  "1.2.3.4",
		},
		{
			name:  "normalize ipv6",
			input: "2001:0db8:0000:0000:0000:ff00:0042:8329",
			want:  "2001:db8::ff00:42:8329",
		},
		{
			name:    "reject empty",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "reject cidr",
			input:   "10.0.0.0/24",
			wantErr: true,
		},
		{
			name:    "reject invalid",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeSingleIP(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestMatchSingleIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		clientIP  string
		allowedIP string
		want      bool
		wantErr   bool
	}{
		{
			name:      "match same ipv4",
			clientIP:  "1.2.3.4",
			allowedIP: "1.2.3.4",
			want:      true,
		},
		{
			name:      "match normalized ipv6",
			clientIP:  "2001:db8::1",
			allowedIP: "2001:0db8:0:0:0:0:0:1",
			want:      true,
		},
		{
			name:      "reject mismatch",
			clientIP:  "1.2.3.4",
			allowedIP: "1.2.3.5",
			want:      false,
		},
		{
			name:      "reject invalid client ip",
			clientIP:  "bad-ip",
			allowedIP: "1.2.3.4",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := MatchSingleIP(tc.clientIP, tc.allowedIP)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestNormalizeSingleIPList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "supports newline and dedupe",
			input: " 1.2.3.4 \n1.2.3.4\n2001:db8::1",
			want:  []string{"1.2.3.4", "2001:db8::1"},
		},
		{
			name:  "supports comma and semicolon",
			input: "1.2.3.4, 5.6.7.8；9.9.9.9",
			want:  []string{"1.2.3.4", "5.6.7.8", "9.9.9.9"},
		},
		{
			name:    "reject empty",
			input:   " \n , ",
			wantErr: true,
		},
		{
			name:    "reject cidr",
			input:   "10.0.0.0/24",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeSingleIPList(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d items, got %d (%v)", len(tc.want), len(got), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("expected %q at index %d, got %q", tc.want[i], i, got[i])
				}
			}
		})
	}
}

func TestMatchSingleIPListText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		clientIP  string
		allowedIP string
		want      bool
		wantErr   bool
	}{
		{
			name:      "match one of multiple ips",
			clientIP:  "5.6.7.8",
			allowedIP: "1.2.3.4\n5.6.7.8",
			want:      true,
		},
		{
			name:      "reject unmatched ip",
			clientIP:  "5.6.7.9",
			allowedIP: "1.2.3.4\n5.6.7.8",
			want:      false,
		},
		{
			name:      "reject invalid whitelist",
			clientIP:  "5.6.7.8",
			allowedIP: "bad-ip",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := MatchSingleIPListText(tc.clientIP, tc.allowedIP)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
