package middleware

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
)

func allowPublicRequest(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error) {
	return ratelimit.Decision{Allowed: true}, nil
}

func TestPublicPolicyCoversHighRiskUnauthenticatedRoutes(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/chat/sms/send"},
		{http.MethodPost, "/api/v1/chat/register"},
		{http.MethodPost, "/api/v1/chat/login"},
		{http.MethodPost, "/api/v1/chat/web-login"},
		{http.MethodPost, "/api/v1/chat/totp/verify"},
		{http.MethodPost, "/api/v1/chat/totp/setup-verify"},
		{http.MethodPost, "/api/v1/chat/auth/refresh"},
		{http.MethodPost, "/api/v1/chat/password/reset"},
		{http.MethodPost, "/api/v1/chat/logout"},
		{http.MethodGet, "/api/v1/chat/login-challenges/123"},
		{http.MethodGet, "/api/v1/chat/files/file-123/download"},
		{http.MethodPost, "/api/v1/chat/login-challenges/123/verify-sms"},
		{http.MethodPost, "/api/v1/chat/login-challenges/123/exchange"},
		{http.MethodPost, "/api/v1/admin/login"},
		{http.MethodPost, "/api/v1/admin/totp/bind"},
		{http.MethodPost, "/api/v1/admin/totp/verify"},
		{http.MethodGet, "/api/v1/public/bootstrap/domain-pool"},
		{http.MethodPost, "/api/v1/webhook/tencent"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			if _, ok := PublicPolicyFor(tt.method, tt.path); !ok {
				t.Fatalf("high-risk public route has no protection policy: %s %s", tt.method, tt.path)
			}
		})
	}
}

func TestPublicPolicyIsMethodSensitive(t *testing.T) {
	if _, ok := PublicPolicyFor(http.MethodGet, "/api/v1/chat/login"); ok {
		t.Fatal("GET /api/v1/chat/login must not inherit the POST login policy")
	}
	if _, ok := PublicPolicyFor(http.MethodPost, "/api/v1/chat/profile/update"); ok {
		t.Fatal("authenticated profile endpoint must not be classified as a public auth endpoint")
	}
	if _, ok := PublicPolicyFor(http.MethodGet, "/api/v1/chat/login-challenges/pending"); ok {
		t.Fatal("authenticated pending challenge list must not be classified as a public endpoint")
	}
	if _, ok := PublicPolicyFor(http.MethodGet, "/api/v1/chat/ping"); ok {
		t.Fatal("cheap ping endpoint must not consume the public auth concurrency budget")
	}
}

func TestSlowExternalAuthEntrypointsHaveIndependentBudgetsAndSharedFuses(t *testing.T) {
	native, _ := PublicPolicyFor(http.MethodPost, "/api/v1/chat/login")
	web, _ := PublicPolicyFor(http.MethodPost, "/api/v1/chat/web-login")
	adminLogin, _ := PublicPolicyFor(http.MethodPost, "/api/v1/admin/login")
	adminTOTP, _ := PublicPolicyFor(http.MethodPost, "/api/v1/admin/totp/verify")
	if native.RateKey == web.RateKey || native.SharedRateKey == "" || native.SharedRateKey != web.SharedRateKey || native.ConcurrencyKey == web.ConcurrencyKey {
		t.Fatal("native and web login must use independent route budgets/gates plus one shared capacity fuse")
	}
	if adminLogin.RateKey == adminTOTP.RateKey || adminLogin.SharedRateKey == "" || adminLogin.SharedRateKey != adminTOTP.SharedRateKey || adminLogin.ConcurrencyKey == adminTOTP.ConcurrencyKey {
		t.Fatal("admin login and TOTP must use independent route budgets/gates plus one shared capacity fuse")
	}
}

func TestPublicProtectionRejectsOversizedBodyBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const maxBodyBytes = 32

	var handlerCalls atomic.Int64
	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:  maxBodyBytes,
		MaxConcurrent: 4,
		Check:         allowPublicRequest,
	}))
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		handlerCalls.Add(1)
		c.Status(http.StatusNoContent)
	})

	body := bytes.Repeat([]byte("x"), maxBodyBytes+1)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("protected handler called %d times for oversized request", got)
	}
}

func TestPublicProtectionAllowsBodyAtConfiguredLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const maxBodyBytes = 32

	var handlerCalls atomic.Int64
	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:  maxBodyBytes,
		MaxConcurrent: 4,
		Check:         allowPublicRequest,
	}))
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		handlerCalls.Add(1)
		c.Status(http.StatusNoContent)
	})

	body := bytes.Repeat([]byte("x"), maxBodyBytes)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := handlerCalls.Load(); got != 1 {
		t.Fatalf("protected handler calls = %d, want 1", got)
	}
}

func TestPublicProtectionFailsClosedWhenDistributedCheckUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var handlerCalls atomic.Int64
	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:  1024,
		MaxConcurrent: 4,
		Check: func(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error) {
			return ratelimit.Decision{}, errors.New("rate-limit backend unavailable")
		},
	}))
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		handlerCalls.Add(1)
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("protected handler called %d times while limiter was unavailable", got)
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("503 response is missing Retry-After")
	}
}

func TestPublicProtectionReturnsTooManyRequestsWhenCheckDenies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var handlerCalls atomic.Int64
	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:  1024,
		MaxConcurrent: 4,
		Check: func(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error) {
			return ratelimit.Decision{Allowed: false}, ratelimit.ErrLimited
		},
	}))
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		handlerCalls.Add(1)
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("protected handler called %d times after rate-limit denial", got)
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("429 response is missing Retry-After")
	}
}

func TestPublicProtectionChecksIPBeforeConsumingGlobalBudgets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var checkedSubjects []string
	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:  1024,
		MaxConcurrent: 4,
		Check: func(_ context.Context, subject string, _ []ratelimit.Rule) (ratelimit.Decision, error) {
			checkedSubjects = append(checkedSubjects, subject)
			if strings.HasPrefix(subject, "ip:") {
				return ratelimit.Decision{Allowed: false}, ratelimit.ErrLimited
			}
			return ratelimit.Decision{Allowed: true}, nil
		},
	}))
	router.POST("/api/v1/chat/register", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/register", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if len(checkedSubjects) != 1 || !strings.HasPrefix(checkedSubjects[0], "ip:") {
		t.Fatalf("checked subjects = %v, want only the IP budget", checkedSubjects)
	}
}

func TestPublicProtectionConcurrencyGateRejectsExcessWork(t *testing.T) {
	gin.SetMode(gin.TestMode)

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	var handlerCalls atomic.Int64

	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:  1024,
		MaxConcurrent: 1,
		Check:         allowPublicRequest,
	}))
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		call := handlerCalls.Add(1)
		if call == 1 {
			entered <- struct{}{}
			<-release
		}
		c.Status(http.StatusNoContent)
	})

	go func() {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		firstDone <- response
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not enter protected handler")
	}

	secondResponse := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(secondResponse, secondRequest)

	if secondResponse.Code != http.StatusServiceUnavailable {
		close(release)
		t.Fatalf("second status = %d, want %d", secondResponse.Code, http.StatusServiceUnavailable)
	}
	if secondResponse.Header().Get("Retry-After") == "" {
		close(release)
		t.Fatal("concurrency rejection is missing Retry-After")
	}
	if got := handlerCalls.Load(); got != 1 {
		close(release)
		t.Fatalf("handler calls while gate was full = %d, want 1", got)
	}

	close(release)
	select {
	case response := <-firstDone:
		if response.Code != http.StatusNoContent {
			t.Fatalf("first status = %d, want %d", response.Code, http.StatusNoContent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not finish after releasing the gate")
	}

	thirdResponse := httptest.NewRecorder()
	thirdRequest := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
	thirdRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(thirdResponse, thirdRequest)
	if thirdResponse.Code != http.StatusNoContent {
		t.Fatalf("request after gate release status = %d, want %d", thirdResponse.Code, http.StatusNoContent)
	}
}

func TestPublicProtectionGlobalGateBoundsWorkAcrossRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	var handlerCalls atomic.Int64

	router := gin.New()
	router.Use(PublicProtection(PublicProtectionOptions{
		MaxBodyBytes:        1024,
		MaxConcurrent:       4,
		GlobalMaxConcurrent: 1,
		Check:               allowPublicRequest,
	}))
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		handlerCalls.Add(1)
		entered <- struct{}{}
		<-release
		c.Status(http.StatusNoContent)
	})
	router.POST("/api/v1/chat/register", func(c *gin.Context) {
		handlerCalls.Add(1)
		c.Status(http.StatusNoContent)
	})

	go func() {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		firstDone <- response
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first route did not enter handler")
	}

	secondResponse := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/v1/chat/register", bytes.NewBufferString(`{}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusServiceUnavailable {
		close(release)
		t.Fatalf("cross-route status = %d, want %d", secondResponse.Code, http.StatusServiceUnavailable)
	}
	if got := handlerCalls.Load(); got != 1 {
		close(release)
		t.Fatalf("handlers entered while global gate was full = %d, want 1", got)
	}

	close(release)
	select {
	case response := <-firstDone:
		if response.Code != http.StatusNoContent {
			t.Fatalf("first status = %d, want %d", response.Code, http.StatusNoContent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first route did not finish after release")
	}
}

func TestUserLoginDoesNotConsumeIPBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/api/v1/chat/login", "/api/v1/chat/web-login"} {
		t.Run(path, func(t *testing.T) {
			checks := 0
			router := gin.New()
			router.Use(PublicProtection(PublicProtectionOptions{
				Check: func(_ context.Context, subject string, _ []ratelimit.Rule) (ratelimit.Decision, error) {
					checks++
					if strings.HasPrefix(subject, "ip:") {
						t.Error("user login must not consume an IP budget")
						return ratelimit.Decision{Allowed: false}, ratelimit.ErrLimited
					}
					return ratelimit.Decision{Allowed: true}, nil
				},
			}))
			router.POST(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`)))
			if rec.Code != http.StatusNoContent || checks == 0 {
				t.Fatalf("status=%d checks=%d; global protection must remain active", rec.Code, checks)
			}
		})
	}
	admin, _ := PublicPolicyFor(http.MethodPost, "/api/v1/admin/login")
	if len(admin.IPRules) == 0 {
		t.Fatal("admin login must retain IP limits")
	}
}
