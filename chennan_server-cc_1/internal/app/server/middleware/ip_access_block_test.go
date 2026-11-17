package middleware

import (
	"net/http"
	"testing"
	"time"
)

func TestIPAccessBlockCacheExpires(t *testing.T) {
	ipAccessBlockCache.Lock()
	ipAccessBlockCache.entries = make(map[string]ipAccessBlockCacheEntry)
	ipAccessBlockCache.Unlock()

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	cacheIPAccessBlock("203.0.113.10", true, now)
	blocked, cached := getCachedIPAccessBlock("203.0.113.10", now.Add(ipAccessBlockCacheTTL-time.Millisecond))
	if !cached || !blocked {
		t.Fatalf("active cached block = (%v, %v), want (true, true)", blocked, cached)
	}
	if _, cached := getCachedIPAccessBlock("203.0.113.10", now.Add(ipAccessBlockCacheTTL)); cached {
		t.Fatal("expired IP block cache entry was still returned")
	}
}

func TestIPAccessBlockCacheDoesNotCacheEmptyAddress(t *testing.T) {
	ipAccessBlockCache.Lock()
	ipAccessBlockCache.entries = make(map[string]ipAccessBlockCacheEntry)
	ipAccessBlockCache.Unlock()

	cacheIPAccessBlock("", true, time.Now())
	if _, cached := getCachedIPAccessBlock("", time.Now()); cached {
		t.Fatal("empty client address must not be cached")
	}
}

func TestIPAccessBlockPolicyCoversAllPublicAuthenticationEntrypoints(t *testing.T) {
	protected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/chat/login"},
		{http.MethodPost, "/api/v1/chat/web-login"},
		{http.MethodPost, "/api/v1/chat/register"},
		{http.MethodPost, "/api/v1/chat/sms/send"},
		{http.MethodPost, "/api/v1/chat/password/reset"},
		{http.MethodPost, "/api/v1/admin/login"},
	}
	for _, tt := range protected {
		if !shouldCheckIPAccessBlock(tt.method, "", tt.path) {
			t.Fatalf("public authentication route bypasses IP block policy: %s %s", tt.method, tt.path)
		}
	}

	excluded := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/webhook/tencent"},
		{http.MethodGet, "/api/v1/public/bootstrap/domain-pool"},
		{http.MethodGet, "/api/v1/chat/files/file-123/download"},
		// 高 QPS 的会话/轮询入口依靠前置内存/Redis 限流，避免随机
		// 攻击 IP 每次额外触发一次 MySQL 手工封禁查询。
		{http.MethodPost, "/api/v1/chat/auth/refresh"},
		{http.MethodGet, "/api/v1/chat/login-challenges/123"},
	}
	for _, tt := range excluded {
		if shouldCheckIPAccessBlock(tt.method, "", tt.path) {
			t.Fatalf("non-authentication public route unexpectedly uses manual IP blocks: %s %s", tt.method, tt.path)
		}
	}
}
