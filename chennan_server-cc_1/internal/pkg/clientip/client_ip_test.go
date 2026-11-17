package clientip

import (
	"net/http"
	"testing"
)

func TestFromRequestUsesForwardedHeaderOnlyFromTrustedProxy(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "172.18.0.10:45678",
		Header: http.Header{
			"X-Forwarded-For": []string{"8.8.8.8, 172.18.0.10"},
			"X-Real-Ip":       []string{"172.18.0.10"},
		},
	}

	got := FromRequest(req, Options{
		TrustedProxyCIDRs: []string{"172.18.0.0/16"},
		HeaderOrder:       []string{"X-Forwarded-For", "X-Real-IP"},
	})
	if got != "8.8.8.8" {
		t.Fatalf("FromRequest() = %q, want %q", got, "8.8.8.8")
	}
}

func TestFromRequestIgnoresSpoofedHeaderFromUntrustedPeer(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "198.51.100.20:45678",
		Header: http.Header{
			"X-Forwarded-For": []string{"8.8.8.8"},
		},
	}

	got := FromRequest(req, Options{
		TrustedProxyCIDRs: []string{"172.18.0.0/16"},
		HeaderOrder:       []string{"X-Forwarded-For"},
	})
	if got != "198.51.100.20" {
		t.Fatalf("FromRequest() = %q, want %q", got, "198.51.100.20")
	}
}

func TestFromRequestPrefersProviderRealIPHeaders(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "127.0.0.1:45678",
		Header: http.Header{
			"Ali-Real-Client-Ip": []string{"1.1.1.1"},
			"X-Forwarded-For":    []string{"8.8.8.8, 127.0.0.1"},
		},
	}

	got := FromRequest(req, Options{
		TrustedProxyCIDRs: []string{"127.0.0.0/8"},
		HeaderOrder:       []string{"Ali-Real-Client-IP", "X-Forwarded-For"},
	})
	if got != "1.1.1.1" {
		t.Fatalf("FromRequest() = %q, want %q", got, "1.1.1.1")
	}
}

func TestFromRequestUsesAliyunESAManagedRealIPHeader(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "127.0.0.1:45678",
		Header: http.Header{
			"True-Client-Ip-Out": []string{"1.1.1.1"},
			"X-Forwarded-For":    []string{"8.8.8.8, 127.0.0.1"},
		},
	}

	got := FromRequest(req, Options{
		TrustedProxyCIDRs: []string{"127.0.0.0/8"},
		HeaderOrder:       []string{"True-Client-IP-OUT", "X-Forwarded-For"},
	})
	if got != "1.1.1.1" {
		t.Fatalf("FromRequest() = %q, want %q", got, "1.1.1.1")
	}
}

func TestFromRequestFallsBackToFirstValidForwardedIP(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "127.0.0.1:45678",
		Header: http.Header{
			"X-Forwarded-For": []string{"10.0.0.7, 172.18.0.10"},
		},
	}

	got := FromRequest(req, Options{
		TrustedProxyCIDRs: []string{"127.0.0.0/8"},
		HeaderOrder:       []string{"X-Forwarded-For"},
	})
	if got != "10.0.0.7" {
		t.Fatalf("FromRequest() = %q, want %q", got, "10.0.0.7")
	}
}
