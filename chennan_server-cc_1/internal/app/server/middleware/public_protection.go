package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
)

const (
	defaultPublicBodyBytes = int64(16 * 1024)
	defaultRetryAfter      = "60"
)

type PublicPolicy struct {
	Action         string
	RateKey        string
	SharedRateKey  string
	ConcurrencyKey string
	MaxBodyBytes   int64
	MaxConcurrent  int
	GlobalRules    []ratelimit.Rule
	SharedRules    []ratelimit.Rule
	IPRules        []ratelimit.Rule
}

type PublicProtectionOptions struct {
	MaxBodyBytes        int64
	MaxConcurrent       int
	GlobalMaxConcurrent int
	Check               func(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error)
}

// PublicProtection is a cheap front gate for unauthenticated, expensive
// endpoints. It must run before middleware that queries MySQL. It combines a
// non-blocking concurrency fuse, a distributed-botnet global limit, an IP
// limit, and a strict request-body cap.
func PublicProtection(options ...PublicProtectionOptions) gin.HandlerFunc {
	var option PublicProtectionOptions
	if len(options) > 0 {
		option = options[0]
	}
	check := option.Check
	if check == nil {
		check = ratelimit.Check
	}
	globalMaxConcurrent := option.GlobalMaxConcurrent
	if globalMaxConcurrent <= 0 {
		// Keep aggregate unauthenticated work below the default MySQL pool size.
		globalMaxConcurrent = 40
	}
	globalGate := make(chan struct{}, globalMaxConcurrent)

	var gatesMu sync.Mutex
	gates := make(map[string]chan struct{})
	gateFor := func(policy PublicPolicy) chan struct{} {
		maxConcurrent := policy.MaxConcurrent
		if option.MaxConcurrent > 0 {
			maxConcurrent = option.MaxConcurrent
		}
		if maxConcurrent <= 0 {
			maxConcurrent = 1
		}
		gatesMu.Lock()
		defer gatesMu.Unlock()
		gateKey := strings.TrimSpace(policy.ConcurrencyKey)
		if gateKey == "" {
			gateKey = policy.Action
		}
		if gate, exists := gates[gateKey]; exists {
			return gate
		}
		gate := make(chan struct{}, maxConcurrent)
		gates[gateKey] = gate
		return gate
	}

	return func(c *gin.Context) {
		policy, protected := PublicPolicyFor(c.Request.Method, c.Request.URL.Path)
		if !protected {
			c.Next()
			return
		}

		select {
		case globalGate <- struct{}{}:
			defer func() { <-globalGate }()
		default:
			ratelimit.LogOverload(policyLogAction(policy), "global_concurrency_gate_full", clientip.FromContext(c))
			handlers.MarkAccessLogSampled(c)
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    http.StatusServiceUnavailable,
				"message": "服务繁忙，请稍后再试",
			})
			return
		}

		gate := gateFor(policy)
		select {
		case gate <- struct{}{}:
			defer func() { <-gate }()
		default:
			ratelimit.LogOverload(policyLogAction(policy), "concurrency_gate_full", clientip.FromContext(c))
			handlers.MarkAccessLogSampled(c)
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    http.StatusServiceUnavailable,
				"message": "服务繁忙，请稍后再试",
			})
			return
		}
		ip := clientip.FromContext(c)
		if strings.TrimSpace(ip) == "" {
			ip = "unknown"
		}
		rateKey := strings.TrimSpace(policy.RateKey)
		if rateKey == "" {
			rateKey = policy.Action
		}
		subjects := []ratelimit.SubjectRules{
			{Subject: ratelimit.HashSubject("ip", ip), Rules: routeRules(rateKey, policy.IPRules)},
			{Subject: "endpoint:" + rateKey, Rules: routeRules(rateKey, policy.GlobalRules)},
		}
		if policy.SharedRateKey != "" && len(policy.SharedRules) > 0 {
			subjects = append(subjects, ratelimit.SubjectRules{
				Subject: "shared_endpoint:" + policy.SharedRateKey,
				Rules:   policy.SharedRules,
			})
		}
		for _, item := range subjects {
			if len(item.Rules) == 0 {
				continue
			}
			decision, err := check(c.Request.Context(), item.Subject, item.Rules)
			if err != nil && !errors.Is(err, ratelimit.ErrLimited) {
				ratelimit.LogOverload(policyLogAction(policy), "rate_limit_check_failed", ip)
				handlers.MarkAccessLogSampled(c)
				c.Header("Retry-After", "1")
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"code":    http.StatusServiceUnavailable,
					"message": "服务繁忙，请稍后再试",
				})
				return
			}
			if errors.Is(err, ratelimit.ErrLimited) || !decision.Allowed {
				ratelimit.LogLimited(policyLogAction(policy), item.Subject, ip, decision)
				handlers.MarkAccessLogSampled(c)
				c.Header("Retry-After", defaultRetryAfter)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code":    http.StatusTooManyRequests,
					"message": ratelimit.DefaultMessage,
				})
				return
			}
		}

		maxBodyBytes := policy.MaxBodyBytes
		if option.MaxBodyBytes > 0 {
			maxBodyBytes = option.MaxBodyBytes
		}
		if maxBodyBytes <= 0 {
			maxBodyBytes = defaultPublicBodyBytes
		}
		if c.Request.Body != nil && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			if c.Request.ContentLength > maxBodyBytes {
				abortBodyTooLarge(c)
				return
			}
			body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyBytes+1))
			_ = c.Request.Body.Close()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"code":    http.StatusBadRequest,
					"message": "请求体读取失败",
				})
				return
			}
			if int64(len(body)) > maxBodyBytes {
				abortBodyTooLarge(c)
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			c.Request.ContentLength = int64(len(body))
		}

		c.Next()
	}
}

func policyLogAction(policy PublicPolicy) string {
	if strings.TrimSpace(policy.RateKey) != "" {
		return policy.RateKey
	}
	return policy.Action
}

func routeRules(rateKey string, rules []ratelimit.Rule) []ratelimit.Rule {
	if len(rules) == 0 {
		return nil
	}
	namespaced := make([]ratelimit.Rule, 0, len(rules))
	for _, rule := range rules {
		rule.Scope = "route:" + rateKey + ":" + rule.Scope
		namespaced = append(namespaced, rule)
	}
	return namespaced
}

func abortBodyTooLarge(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
		"code":    http.StatusRequestEntityTooLarge,
		"message": "请求体过大",
	})
}

func PublicPolicyFor(method, path string) (PublicPolicy, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	if method == http.MethodPost {
		switch path {
		case "/api/v1/chat/login":
			return PublicPolicy{
				Action:         "login",
				RateKey:        "native_login",
				SharedRateKey:  "login",
				ConcurrencyKey: "native_login",
				MaxBodyBytes:   16 * 1024,
				MaxConcurrent:  16,
				GlobalRules:    ratelimit.PublicLoginGlobalRules,
				SharedRules:    ratelimit.PublicLoginSharedGlobalRules,
				// 用户登录不按 IP 限频，避免同一出口下多个正常用户互相影响。
			}, true
		case "/api/v1/chat/web-login":
			return PublicPolicy{
				Action:         "login",
				RateKey:        "web_login",
				SharedRateKey:  "login",
				ConcurrencyKey: "web_login",
				MaxBodyBytes:   16 * 1024,
				MaxConcurrent:  16,
				GlobalRules:    ratelimit.PublicLoginGlobalRules,
				SharedRules:    ratelimit.PublicLoginSharedGlobalRules,
				// 用户登录不按 IP 限频，避免同一出口下多个正常用户互相影响。
			}, true
		case "/api/v1/chat/register":
			return PublicPolicy{
				Action:        "register",
				MaxBodyBytes:  16 * 1024,
				MaxConcurrent: 4,
				GlobalRules:   ratelimit.PublicRegisterGlobalRules,
				IPRules:       ratelimit.PublicRegisterIPRules,
			}, true
		case "/api/v1/chat/sms/send":
			return PublicPolicy{
				Action:        "sms_send",
				MaxBodyBytes:  8 * 1024,
				MaxConcurrent: 4,
				GlobalRules:   ratelimit.PublicSMSGlobalRules,
				IPRules:       ratelimit.PublicSMSIPRules,
			}, true
		case "/api/v1/chat/password/reset":
			return PublicPolicy{
				Action:        "password_reset",
				MaxBodyBytes:  16 * 1024,
				MaxConcurrent: 4,
				GlobalRules:   ratelimit.PublicPasswordResetGlobalRules,
				IPRules:       ratelimit.PublicPasswordResetIPRules,
			}, true
		case "/api/v1/chat/auth/refresh":
			return PublicPolicy{
				Action:         "refresh_session",
				RateKey:        "refresh",
				SharedRateKey:  "refresh_session",
				ConcurrencyKey: "refresh",
				MaxBodyBytes:   32 * 1024,
				MaxConcurrent:  32,
				GlobalRules:    ratelimit.PublicRefreshGlobalRules,
				SharedRules:    ratelimit.PublicRefreshSharedGlobalRules,
				IPRules:        ratelimit.PublicRefreshIPRules,
			}, true
		case "/api/v1/chat/logout":
			return PublicPolicy{
				Action:         "refresh_session",
				RateKey:        "logout",
				SharedRateKey:  "refresh_session",
				ConcurrencyKey: "logout",
				MaxBodyBytes:   32 * 1024,
				MaxConcurrent:  16,
				GlobalRules:    ratelimit.PublicRefreshGlobalRules,
				SharedRules:    ratelimit.PublicRefreshSharedGlobalRules,
				IPRules:        ratelimit.PublicRefreshIPRules,
			}, true
		case "/api/v1/chat/totp/verify", "/api/v1/chat/totp/setup-verify":
			return PublicPolicy{
				Action:        "user_totp_verify",
				MaxBodyBytes:  16 * 1024,
				MaxConcurrent: 8,
				GlobalRules:   ratelimit.PublicVerifyGlobalRules,
				IPRules:       ratelimit.PublicVerifyIPRules,
			}, true
		case "/api/v1/admin/login":
			return PublicPolicy{
				Action:         "admin_auth",
				RateKey:        "admin_login",
				SharedRateKey:  "admin_auth",
				ConcurrencyKey: "admin_login",
				MaxBodyBytes:   16 * 1024,
				MaxConcurrent:  4,
				GlobalRules:    ratelimit.PublicAdminLoginGlobalRules,
				SharedRules:    ratelimit.PublicAdminAuthSharedGlobalRules,
				IPRules:        ratelimit.PublicAdminLoginIPRules,
			}, true
		case "/api/v1/admin/totp/bind", "/api/v1/admin/totp/verify":
			return PublicPolicy{
				Action:         "admin_auth",
				RateKey:        "admin_totp",
				SharedRateKey:  "admin_auth",
				ConcurrencyKey: "admin_totp",
				MaxBodyBytes:   16 * 1024,
				MaxConcurrent:  4,
				GlobalRules:    ratelimit.PublicAdminLoginGlobalRules,
				SharedRules:    ratelimit.PublicAdminAuthSharedGlobalRules,
				IPRules:        ratelimit.PublicAdminLoginIPRules,
			}, true
		case "/api/v1/webhook/tencent":
			return PublicPolicy{
				Action:        "tencent_webhook",
				MaxBodyBytes:  256 * 1024,
				MaxConcurrent: 64,
				GlobalRules:   ratelimit.PublicWebhookGlobalRules,
			}, true
		}
		if strings.HasPrefix(path, "/api/v1/chat/login-challenges/") &&
			(strings.HasSuffix(path, "/verify-sms") || strings.HasSuffix(path, "/exchange")) {
			return PublicPolicy{
				Action:        "login_challenge",
				MaxBodyBytes:  16 * 1024,
				MaxConcurrent: 8,
				GlobalRules:   ratelimit.PublicChallengeGlobalRules,
				IPRules:       ratelimit.PublicChallengeIPRules,
			}, true
		}
	}
	if method == http.MethodGet {
		if path == "/api/v1/public/bootstrap/domain-pool" {
			return PublicPolicy{
				Action:        "public_bootstrap",
				MaxBodyBytes:  1,
				MaxConcurrent: 16,
				GlobalRules:   ratelimit.PublicBootstrapGlobalRules,
				IPRules:       ratelimit.PublicBootstrapIPRules,
			}, true
		}
		challengePart := strings.TrimPrefix(path, "/api/v1/chat/login-challenges/")
		if challengePart != path && challengePart != "" && challengePart != "pending" && !strings.Contains(challengePart, "/") {
			return PublicPolicy{
				Action:        "login_challenge_status",
				MaxBodyBytes:  1,
				MaxConcurrent: 32,
				GlobalRules:   ratelimit.PublicChallengeStatusGlobalRules,
				IPRules:       ratelimit.PublicChallengeStatusIPRules,
			}, true
		}
		filePart := strings.TrimPrefix(path, "/api/v1/chat/files/")
		if filePart != path && strings.HasSuffix(filePart, "/download") && strings.Count(filePart, "/") == 1 {
			return PublicPolicy{
				Action:        "attachment_download",
				MaxBodyBytes:  1,
				MaxConcurrent: 32,
				GlobalRules:   ratelimit.PublicDownloadGlobalRules,
				IPRules:       ratelimit.PublicDownloadIPRules,
			}, true
		}
	}
	return PublicPolicy{}, false
}
