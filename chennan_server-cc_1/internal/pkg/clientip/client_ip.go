package clientip

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/pkg/config"
)

type Options struct {
	TrustedProxyCIDRs []string
	HeaderOrder       []string
}

// FromContext returns the real client IP after validating the direct peer is a trusted proxy.
func FromContext(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	ip := FromRequest(c.Request, configuredOptions())
	if ip != "" {
		return ip
	}
	return c.ClientIP()
}

// FromRequest resolves a client IP from trusted proxy headers, then falls back to RemoteAddr.
func FromRequest(r *http.Request, opts Options) string {
	if r == nil {
		return ""
	}

	fallback := remoteAddrIP(r.RemoteAddr)
	if !fallback.IsValid() {
		return ""
	}
	if !isTrustedProxy(fallback, trustedProxyCIDRs(opts.TrustedProxyCIDRs)) {
		return fallback.String()
	}

	for _, header := range headerOrder(opts.HeaderOrder) {
		value := r.Header.Get(header)
		if value == "" {
			continue
		}
		if ip := ipFromHeader(header, value); ip != "" {
			return ip
		}
	}

	return fallback.String()
}

// TrustedProxyCIDRs returns configured CIDRs or the safe default proxy ranges for local reverse proxies.
func TrustedProxyCIDRs(cidrs []string) []string {
	return trustedProxyCIDRs(cidrs)
}

func configuredOptions() Options {
	app := config.Cfg.Get().App
	return Options{
		TrustedProxyCIDRs: app.TrustedProxyCIDRs,
		HeaderOrder:       app.RealIPHeaderOrder,
	}
}

func trustedProxyCIDRs(cidrs []string) []string {
	if len(cidrs) == 0 {
		return append([]string(nil), config.DefaultTrustedProxyCIDRs...)
	}
	return append([]string(nil), cidrs...)
}

func headerOrder(headers []string) []string {
	if len(headers) == 0 {
		return append([]string(nil), config.DefaultRealIPHeaderOrder...)
	}
	return append([]string(nil), headers...)
}

func ipFromHeader(header, value string) string {
	if strings.EqualFold(header, "X-Forwarded-For") {
		return ipFromForwardedFor(value)
	}
	if addr, ok := parseAddr(value); ok {
		return addr.String()
	}
	return ""
}

func ipFromForwardedFor(value string) string {
	var firstValid netip.Addr
	for _, part := range strings.Split(value, ",") {
		addr, ok := parseAddr(part)
		if !ok {
			continue
		}
		if !firstValid.IsValid() {
			firstValid = addr
		}
		if isPublicClientIP(addr) {
			return addr.String()
		}
	}
	if firstValid.IsValid() {
		return firstValid.String()
	}
	return ""
}

func isTrustedProxy(addr netip.Addr, cidrs []string) bool {
	if !addr.IsValid() {
		return false
	}
	for _, raw := range cidrs {
		prefix, ok := parseTrustedPrefix(raw)
		if !ok {
			continue
		}
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func parseTrustedPrefix(raw string) (netip.Prefix, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return netip.Prefix{}, false
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), true
	}
	addr, ok := parseAddr(value)
	if !ok {
		return netip.Prefix{}, false
	}
	bits := 128
	if addr.Is4() {
		bits = 32
	}
	return netip.PrefixFrom(addr, bits), true
}

func remoteAddrIP(remoteAddr string) netip.Addr {
	if addr, ok := parseAddr(remoteAddr); ok {
		return addr
	}
	return netip.Addr{}
}

func parseAddr(raw string) (netip.Addr, bool) {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "\"")
	if value == "" || strings.EqualFold(value, "unknown") {
		return netip.Addr{}, false
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	} else if strings.Count(value, ":") == 1 && strings.Contains(value, ".") {
		if host, _, splitErr := net.SplitHostPort(value); splitErr == nil {
			value = host
		}
	}

	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func isPublicClientIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() || addr.IsMulticast() {
		return false
	}
	if prefix, ok := parseTrustedPrefix("100.64.0.0/10"); ok && prefix.Contains(addr) {
		return false
	}
	return addr.IsGlobalUnicast()
}
