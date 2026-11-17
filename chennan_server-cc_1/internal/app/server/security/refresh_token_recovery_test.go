package security

import "testing"

func TestRefreshRecoveryCipherBindsSessionAndKey(t *testing.T) {
	secret := "test-only-refresh-key-12345678"
	sealed, err := SealRefreshToken("token", secret, "7:device")
	if err != nil {
		t.Fatal(err)
	}
	if token, err := OpenRefreshToken(sealed, secret, "7:device"); err != nil || token != "token" {
		t.Fatalf("roundtrip=%q %v", token, err)
	}
	for _, tc := range []struct{ key, binding string }{{secret, "8:device"}, {secret, "7:other"}, {"different-test-only-key-12345678", "7:device"}} {
		if _, err := OpenRefreshToken(sealed, tc.key, tc.binding); err == nil {
			t.Fatal("accepted wrong key or session binding")
		}
	}
	if _, err := SealRefreshToken("token", "", "7:device"); err == nil {
		t.Fatal("accepted missing key")
	}
}
