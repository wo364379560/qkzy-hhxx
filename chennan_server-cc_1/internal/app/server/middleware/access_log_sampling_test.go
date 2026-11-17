package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
)

func TestPublicProtectionMarksSampledRejections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		check      func(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error)
		wantStatus int
	}{
		{
			name: "rate limited",
			check: func(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error) {
				return ratelimit.Decision{Allowed: false}, ratelimit.ErrLimited
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "limiter unavailable",
			check: func(context.Context, string, []ratelimit.Rule) (ratelimit.Decision, error) {
				return ratelimit.Decision{}, errors.New("limiter unavailable")
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sampled bool
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Next()
				sampled = handlers.AccessLogSampled(c)
			})
			router.Use(PublicProtection(PublicProtectionOptions{
				MaxBodyBytes:        1024,
				MaxConcurrent:       4,
				GlobalMaxConcurrent: 8,
				Check:               tt.check,
			}))
			router.POST("/api/v1/chat/login", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", bytes.NewBufferString(`{}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if !sampled {
				t.Fatal("sampled rejection did not set the explicit access-log marker")
			}
		})
	}
}

func TestUserRateLimitMarksSampled429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var sampled bool
	subject := fmt.Sprintf("access-log-sampling-test-user-%d", time.Now().UnixNano())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() == http.StatusTooManyRequests {
			sampled = handlers.AccessLogSampled(c)
		}
	})
	router.Use(func(c *gin.Context) {
		c.Set("im_user_id", subject)
		c.Next()
	})
	router.Use(UserRateLimit("access_log_sampling_test", []ratelimit.Rule{
		{Scope: "test:access_log_sampling:user", Limit: 1, Window: time.Hour},
	}))
	router.POST("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/limited", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/limited", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("UserRateLimit 429 is missing Retry-After")
	}
	if !sampled {
		t.Fatal("UserRateLimit 429 did not set the explicit access-log marker")
	}
}
