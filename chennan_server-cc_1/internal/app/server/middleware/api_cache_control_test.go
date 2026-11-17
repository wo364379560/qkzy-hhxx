package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPINoStoreIncludesBusinessErrorsAndBootstrap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, status := range []int{200, 401, 403, 429, 500, 503} {
		for _, path := range []string{"/chat/ping", "/public/bootstrap/domain-pool", "/chat/profile", "/admin/users"} {
			r := gin.New()
			v1 := r.Group("/api/v1", APINoStore())
			v1.GET(path, func(c *gin.Context) { c.AbortWithStatusJSON(status, gin.H{"code": status}) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1"+path, nil))
			if w.Code != status || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("%s status %d: code=%d cache=%q", path, status, w.Code, w.Header().Get("Cache-Control"))
			}
		}
	}
}
