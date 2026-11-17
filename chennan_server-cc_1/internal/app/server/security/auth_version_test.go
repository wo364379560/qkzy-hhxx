package security

import "testing"

func TestNormalizeAuthVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input int
		want  int
	}{
		{name: "positive", input: 3, want: 3},
		{name: "zero defaults to one", input: 0, want: 1},
		{name: "negative defaults to one", input: -2, want: 1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeAuthVersion(tc.input); got != tc.want {
				t.Fatalf("NormalizeAuthVersion(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestNextAuthVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input int
		want  int
	}{
		{name: "increments existing version", input: 4, want: 5},
		{name: "zero starts from two", input: 0, want: 2},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := NextAuthVersion(tc.input); got != tc.want {
				t.Fatalf("NextAuthVersion(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestMatchesAuthVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		tokenVersion   int
		currentVersion int
		want           bool
	}{
		{name: "same explicit version", tokenVersion: 2, currentVersion: 2, want: true},
		{name: "legacy zero matches default", tokenVersion: 0, currentVersion: 1, want: true},
		{name: "version mismatch", tokenVersion: 1, currentVersion: 2, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchesAuthVersion(tc.tokenVersion, tc.currentVersion); got != tc.want {
				t.Fatalf(
					"MatchesAuthVersion(%d, %d) = %v, want %v",
					tc.tokenVersion,
					tc.currentVersion,
					got,
					tc.want,
				)
			}
		})
	}
}
