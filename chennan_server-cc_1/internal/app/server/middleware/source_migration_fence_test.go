package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSourceMigrationFenceBlocksBusinessAndAllowsControlPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	marker := filepath.Join(t.TempDir(), "fence.json")
	t.Setenv("CHENGNAN_SOURCE_MIGRATION_FENCE_FILE", marker)
	if err := os.WriteFile(marker, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(SourceMigrationFence())
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/chat/ping"},
		{http.MethodGet, "/api/v1/public/bootstrap/domain-pool"},
		{http.MethodPost, "/api/v1/webhook/tencent"},
		{http.MethodPost, "/api/v1/chat/login"},
		{http.MethodPost, "/api/v1/chat/register"},
		{http.MethodPost, "/api/v1/admin/users/batch"},
	} {
		router.Handle(route.method, route.path, func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
	}

	for _, path := range []string{
		"/api/v1/chat/ping",
		"/api/v1/public/bootstrap/domain-pool",
		"/api/v1/webhook/tencent",
	} {
		method := http.MethodGet
		if path == "/api/v1/webhook/tencent" {
			method = http.MethodPost
		}
		request := httptest.NewRequest(method, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("allowed path %s returned %d", path, response.Code)
		}
	}

	for _, path := range []string{
		"/api/v1/chat/login",
		"/api/v1/chat/register",
		"/api/v1/admin/users/batch",
	} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("blocked path %s returned %d", path, response.Code)
		}
		if response.Header().Get("Retry-After") != "60" {
			t.Fatalf("blocked path %s missed Retry-After", path)
		}
	}
}

func TestSourceMigrationFenceInactiveWithoutMarker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(
		"CHENGNAN_SOURCE_MIGRATION_FENCE_FILE",
		filepath.Join(t.TempDir(), "missing.json"),
	)
	router := gin.New()
	router.Use(SourceMigrationFence())
	router.POST("/api/v1/chat/login", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/login", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("inactive fence returned %d", response.Code)
	}
}
