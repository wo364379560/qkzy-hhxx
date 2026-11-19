package handler_admin

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistrationIPNonSuperAdminForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{"", "admin", "assistant"} {
		for _, h := range []gin.HandlerFunc{ListRegistrationIPs, ChangeRegistrationIP} {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/api/v1/admin/security/registration-ips", strings.NewReader(`{"ip":"8.8.8.8","current_password":"unused"}`))
			c.Set("admin_role", role)
			h(c)
			if rec.Code != 403 {
				t.Fatalf("role=%q status=%d", role, rec.Code)
			}
		}
	}
}
