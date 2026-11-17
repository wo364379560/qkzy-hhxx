package authlimit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
)

func newAuthLimitTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", nil)
	return ctx, response
}

func TestEnforceDecisionMarksLimitedRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, response := newAuthLimitTestContext()
	decision := ratelimit.Decision{
		Allowed: false,
		Rule: ratelimit.Rule{
			Scope: "test:login:account",
			Limit: 1,
		},
		Count: 2,
		Limit: 1,
	}

	if enforceDecision(ctx, "test_login", decision, ratelimit.ErrLimited) {
		t.Fatal("limited decision unexpectedly allowed")
	}
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if !handlers.AccessLogSampled(ctx) {
		t.Fatal("limited rejection did not mark its access log as sampled")
	}
}

func TestEnforceDecisionMarksLimiterFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, response := newAuthLimitTestContext()

	if enforceDecision(ctx, "test_login", ratelimit.Decision{}, errors.New("limiter unavailable")) {
		t.Fatal("limiter failure unexpectedly allowed")
	}
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !handlers.AccessLogSampled(ctx) {
		t.Fatal("limiter failure did not mark its access log as sampled")
	}
}

func TestAbortIfAuthWorkBusyMarksOverload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, response := newAuthLimitTestContext()

	if !AbortIfAuthWorkBusy(ctx, authwork.ErrBusy) {
		t.Fatal("ErrBusy was not handled")
	}
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !handlers.AccessLogSampled(ctx) {
		t.Fatal("auth-work overload did not mark its access log as sampled")
	}
}
