package handler_admin

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateManagerLegacyPayload(t *testing.T) {
	for _, tc := range []struct {
		name, payload, want string
		invalid             bool
	}{
		{"legacy", `{"username":"tester","password":"abcdefgh"}`, "tester", false},
		{"nickname", `{"username":"tester","password":"abcdefgh","display_name":" 测试管理员 "}`, "测试管理员", false},
		{"blank", `{"username":"tester","password":"abcdefgh","display_name":"  "}`, "tester", false},
		{"short username", `{"username":"ab","password":"abcdefgh"}`, "", true},
		{"short password", `{"username":"tester","password":"short"}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", strings.NewReader(tc.payload))
			c.Request.Header.Set("Content-Type", "application/json")
			var req createManagerReq
			err := c.ShouldBindJSON(&req)
			if (err != nil) != tc.invalid {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if tc.invalid {
				return
			}
			req.normalizeDisplayName()
			if req.DisplayName != tc.want {
				t.Fatalf("got %q want %q", req.DisplayName, tc.want)
			}
		})
	}
}
