package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestShouldSkipAccessLog(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
		want   bool
	}{
		{
			name:   "successful get query is skipped",
			method: http.MethodGet,
			path:   "/api/v1/chat/groups/pin-message",
			status: http.StatusOK,
			want:   true,
		},
		{
			name:   "failed get query is kept",
			method: http.MethodGet,
			path:   "/api/v1/chat/groups/pin-message",
			status: http.StatusInternalServerError,
			want:   false,
		},
		{
			name:   "successful options is skipped",
			method: http.MethodOptions,
			path:   "/api/v1/chat/login",
			status: http.StatusNoContent,
			want:   true,
		},
		{
			name:   "successful refresh is skipped",
			method: http.MethodPost,
			path:   "/api/v1/chat/auth/refresh",
			status: http.StatusOK,
			want:   true,
		},
		{
			name:   "successful login write is kept",
			method: http.MethodPost,
			path:   "/api/v1/chat/login",
			status: http.StatusOK,
			want:   false,
		},
		{
			name:   "rate limited request is kept without explicit mark",
			method: http.MethodPost,
			path:   "/api/v1/chat/login",
			status: http.StatusTooManyRequests,
			want:   false,
		},
		{
			name:   "service unavailable is kept until explicitly marked",
			method: http.MethodPost,
			path:   "/api/v1/chat/login",
			status: http.StatusServiceUnavailable,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSkipAccessLog(tt.method, tt.path, tt.status); got != tt.want {
				t.Fatalf("shouldSkipAccessLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkAccessLogSampledIsExplicit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	if AccessLogSampled(ctx) {
		t.Fatal("new context unexpectedly marked as sampled")
	}

	MarkAccessLogSampled(ctx)
	if !AccessLogSampled(ctx) {
		t.Fatal("MarkAccessLogSampled did not set the explicit marker")
	}
}

func TestTruncateLogValue(t *testing.T) {
	if got := truncateLogValue("123456", 4); got != "1234" {
		t.Fatalf("truncateLogValue() = %q, want %q", got, "1234")
	}
}
