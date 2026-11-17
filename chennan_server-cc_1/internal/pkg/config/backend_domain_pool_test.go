package config

import "testing"

func TestBackendOriginsRequireHTTPSOriginWithoutAPIPrefix(t *testing.T) {
	for _, input := range []string{
		"http://api.prfgsu.com", "https://api.prfgsu.com/api/v1", "https://api.prfgsu.com?x=1",
		"https://api.prfgsu.com?", "https://api.prfgsu.com/#x", "https://u:p@api.prfgsu.com",
		"https://api.prfgsu.com:8443", "https://127.0.0.1", "https://127.000.000.001", "https://[::1]", "https://relay.local", "https://localhost",
	} {
		if _, err := normalizeBackendOrigin(input); err == nil {
			t.Errorf("accepted invalid origin %q", input)
		}
	}
	for _, input := range []string{"https://api.prfgsu.com", "https://api1.prfgsu.com", "https://api.uktjwg.cn", "https://api1.uktjwg.cn", "https://api.ruizhihe.com"} {
		if got, err := normalizeBackendOrigin(input + "/"); err != nil || got != input {
			t.Errorf("%q => %q, %v", input, got, err)
		}
	}
	got, err := normalizeBackendOrigin(" https://API1.PRFGSU.COM:443/ ")
	if err != nil || got != "https://api1.prfgsu.com" {
		t.Fatalf("normalization: %q %v", got, err)
	}
}

func TestBackendDomainPoolMatchesClientTTLAndSizeLimits(t *testing.T) {
	for _, ttl := range []int{1, 59, 86401} {
		settings := BackendDomainPoolSettings{TTLSeconds: ttl, APIOrigins: []string{"https://api.prfgsu.com"}}
		if err := NormalizeBackendDomainPoolSettings(&settings); err == nil {
			t.Errorf("accepted ttl %d that clients cannot apply", ttl)
		}
	}
	settings := BackendDomainPoolSettings{APIOrigins: []string{"https://API1.PRFGSU.COM:443/", "https://api1.prfgsu.com"}}
	if err := NormalizeBackendDomainPoolSettings(&settings); err != nil || settings.TTLSeconds != 3600 || len(settings.APIOrigins) != 1 {
		t.Fatalf("normalized document incompatible with clients: %+v, %v", settings, err)
	}
	settings.APIOrigins = make([]string, 17)
	if err := NormalizeBackendDomainPoolSettings(&settings); err == nil {
		t.Fatal("accepted oversized pool")
	}
}
