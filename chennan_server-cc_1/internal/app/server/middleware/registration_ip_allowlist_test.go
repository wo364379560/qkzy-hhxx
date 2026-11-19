package middleware

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegistrationHourLimitAndLiveExemption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	limiter := ratelimit.NewLocalLimiter(ratelimit.LocalOptions{Limit: ratelimit.PublicRegisterIPRules[0].Limit, Window: time.Hour, Now: func() time.Time { return now }})
	allowed, lookupCalls, handlerCalls := false, 0, 0
	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		Check: func(_ context.Context, subject string, rules []ratelimit.Rule) (ratelimit.Decision, error) {
			if strings.HasPrefix(subject, "ip:") {
				d := limiter.Check(subject, 1)
				d.Rule = rules[0]
				if !d.Allowed {
					return d, ratelimit.ErrLimited
				}
				return d, nil
			}
			return ratelimit.Decision{Allowed: true}, nil
		},
		CheckRegistrationIP: func(_ context.Context, ip string) (bool, error) {
			lookupCalls++
			if ip != "8.8.8.8" {
				t.Fatalf("unexpected IP %s", ip)
			}
			return allowed, nil
		},
	}))
	router.POST("/api/v1/chat/register", func(c *gin.Context) { handlerCalls++; c.Status(204) })
	request := func() int {
		r := httptest.NewRequest("POST", "/api/v1/chat/register", strings.NewReader(`{}`))
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Real-IP", "8.8.8.8")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w.Code
	}
	for i := 0; i < 50; i++ {
		if got := request(); got != 204 {
			t.Fatalf("registration %d: status=%d", i+1, got)
		}
	}
	if lookupCalls != 0 {
		t.Fatal("ordinary requests must not query allowlist")
	}
	if got := request(); got != 429 {
		t.Fatalf("51st request=%d want429", got)
	}
	allowed = true
	if got := request(); got != 204 {
		t.Fatalf("allowlisted IP=%d want204", got)
	}
	allowed = false
	if got := request(); got != 429 {
		t.Fatalf("removed IP=%d want429", got)
	}
	if handlerCalls != 51 {
		t.Fatalf("unexpected handler calls %d", handlerCalls)
	}
	now = now.Add(time.Hour)
	if got := request(); got != 204 {
		t.Fatalf("new window=%d want204", got)
	}
}

func TestRegistrationExemptionDoesNotBypassOtherProtections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, remote, header, scope string
		globalDenied, lookupError   bool
		want                        int
	}{
		{name: "trusted", remote: "127.0.0.1:1", header: "8.8.8.8", want: 204},
		{name: "untrusted spoof", remote: "8.8.4.4:1", header: "8.8.8.8", want: 429},
		{name: "missing header", remote: "127.0.0.1:1", want: 429},
		{name: "ambiguous header", remote: "127.0.0.1:1", header: "8.8.8.8, 8.8.4.4", want: 429},
		{name: "storage unavailable", remote: "127.0.0.1:1", header: "8.8.8.8", lookupError: true, want: 429},
		{name: "global fuse", remote: "127.0.0.1:1", header: "8.8.8.8", globalDenied: true, want: 429},
		{name: "other IP rule", remote: "127.0.0.1:1", header: "8.8.8.8", scope: "route:register:another_rule", want: 429},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lookups := 0
			router := gin.New()
			router.Use(PublicProtection(PublicProtectionOptions{
				Check: func(_ context.Context, subject string, rules []ratelimit.Rule) (ratelimit.Decision, error) {
					if strings.HasPrefix(subject, "ip:") {
						rule := rules[0]
						if tt.scope != "" {
							rule.Scope = tt.scope
						}
						return ratelimit.Decision{Allowed: false, Rule: rule}, ratelimit.ErrLimited
					}
					if tt.globalDenied {
						return ratelimit.Decision{Allowed: false}, ratelimit.ErrLimited
					}
					return ratelimit.Decision{Allowed: true}, nil
				},
				CheckRegistrationIP: func(context.Context, string) (bool, error) {
					lookups++
					if tt.lookupError {
						return false, errors.New("unavailable")
					}
					return true, nil
				},
			}))
			router.POST("/api/v1/chat/register", func(c *gin.Context) { c.Status(204) })
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/register", strings.NewReader(`{}`))
			req.RemoteAddr = tt.remote
			req.Header.Set("X-Real-IP", tt.header)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want%d", rec.Code, tt.want)
			}
			if (tt.globalDenied || tt.scope != "" || tt.remote == "8.8.4.4:1" || tt.header == "" || strings.Contains(tt.header, ",")) && lookups != 0 {
				t.Fatal("untrusted or globally rejected requests must not query storage")
			}
		})
	}
}
