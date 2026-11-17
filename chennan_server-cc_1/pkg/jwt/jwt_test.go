package jwt

import (
	"testing"
	"time"
)

func TestAdminTokenExpiry(t *testing.T) {
	t.Parallel()

	defaultExpiry := 24 * time.Hour
	superExpiry := 180 * 24 * time.Hour

	cases := []struct {
		name string
		role string
		want time.Duration
	}{
		{name: "super admin uses dedicated expiry", role: "super_admin", want: superExpiry},
		{name: "admin uses default expiry", role: "admin", want: defaultExpiry},
		{name: "assistant uses default expiry", role: "assistant", want: defaultExpiry},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := adminTokenExpiry(tc.role, defaultExpiry, superExpiry); got != tc.want {
				t.Fatalf("adminTokenExpiry()=%s want %s", got, tc.want)
			}
		})
	}
}

func TestAdminTokenExpiryFallsBackWhenSuperAdminExpiryEmpty(t *testing.T) {
	t.Parallel()

	defaultExpiry := 24 * time.Hour
	if got := adminTokenExpiry("super_admin", defaultExpiry, 0); got != defaultExpiry {
		t.Fatalf("adminTokenExpiry()=%s want %s", got, defaultExpiry)
	}
}
