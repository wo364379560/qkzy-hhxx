package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSOriginAllowed(t *testing.T) {
	allowed := map[string]struct{}{
		"https://rzh35ht.ruizhihe.com": {},
		"https://*.ruizhihe.com":       {},
		"http://localhost:5173":       {},
	}

	tests := []struct {
		origin string
		want   bool
	}{
		{origin: "https://rzh35ht.ruizhihe.com", want: true},
		{origin: "https://abc.ruizhihe.com", want: true},
		{origin: "http://localhost:5173", want: true},
		{origin: "https://ruizhihe.com", want: false},
		{origin: "https://a.b.ruizhihe.com", want: false},
		{origin: "http://rzh35ht.ruizhihe.com", want: false},
		{origin: "https://evil.ruizhihe.com.example", want: false},
		{origin: "https://not-ruizhihe.com", want: false},
		{origin: "https://rzh35ht.ruizhihe.com/login", want: false},
	}

	for _, tt := range tests {
		if got := corsOriginAllowed(tt.origin, allowed); got != tt.want {
			t.Fatalf("corsOriginAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
		}
	}
}

func TestCORSAdminLoginPreflightReflectsESAOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware([]string{
		"https://api.prfgsu.com",
		"https://rzh35ht.ruizhihe.com",
		"https://*.ruizhihe.com",
		"http://localhost:5173",
	}))
	router.POST("/api/v1/admin/login", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/login", nil)
	req.Header.Set("Origin", "https://rzh35ht.ruizhihe.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://rzh35ht.ruizhihe.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected Access-Control-Allow-Credentials=true")
	}
}

func TestCORSUnknownOriginPreflightHasNoAllowOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware([]string{
		"https://api.prfgsu.com",
		"http://localhost:5173",
	}))
	router.POST("/api/v1/admin/login", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/login", nil)
	req.Header.Set("Origin", "https://rzh35ht.ruizhihe.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin = %q", got)
	}
}
