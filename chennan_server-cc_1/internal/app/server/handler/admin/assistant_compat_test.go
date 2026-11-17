package handler_admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
)

func TestCreateAssistantCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, payload, role, wantName string
		status                        int
	}{
		{"legacy form", `{"username":"1242","password":"abcdefgh"}`, "admin", "1242", 200},
		{"empty nickname", `{"username":"tester","password":"abcdefgh","display_name":""}`, "admin", "tester", 200},
		{"blank nickname", `{"username":"tester","password":"abcdefgh","display_name":"  "}`, "admin", "tester", 200},
		{"custom nickname", `{"username":"tester","password":"abcdefgh","display_name":" 测试助理 "}`, "admin", "测试助理", 200},
		{"trim account", `{"username":" tester ","password":"abcdefgh"}`, "admin", "tester", 200},
		{"super admin", `{"username":"tester","password":"abcdefgh"}`, "super_admin", "tester", 200},
		{"short username", `{"username":"ab","password":"abcdefgh"}`, "admin", "", 400},
		{"blank username", `{"username":"   ","password":"abcdefgh"}`, "admin", "", 400},
		{"long username", `{"username":"` + strings.Repeat("u", 33) + `","password":"abcdefgh"}`, "admin", "", 400},
		{"short password", `{"username":"tester","password":"short"}`, "admin", "", 400},
		{"empty password", `{"username":"tester","password":""}`, "admin", "", 400},
		{"unicode name boundary", `{"username":"tester","password":"abcdefgh","display_name":"` + strings.Repeat("睿", 64) + `"}`, "admin", strings.Repeat("睿", 64), 200},
		{"long nickname", `{"username":"tester","password":"abcdefgh","display_name":"` + strings.Repeat("睿", 65) + `"}`, "admin", "", 400},
		{"invalid JSON", `{"username":`, "admin", "", 400},
		{"wrong field type", `{"username":1234,"password":"abcdefgh"}`, "admin", "", 400},
		{"assistant denied", `{"username":"tester","password":"abcdefgh"}`, "assistant", "", 403},
		{"no role denied", `{"username":"tester","password":"abcdefgh"}`, "", "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(writer)
			c.Set("admin_role", tc.role)
			c.Set("admin_id", uint(7))
			c.Request = httptest.NewRequest(http.MethodPost, "/assistants", strings.NewReader(tc.payload))
			c.Request.Header.Set("Content-Type", "application/json")
			called := false
			createAssistant(c, func(req controller_admin.CreateAssistantReq) (*controller_admin.AssistantResult, error) {
				called = true
				if req.DisplayName != tc.wantName || req.Username != strings.TrimSpace(req.Username) || req.CreatedBy != 7 {
					t.Fatalf("incorrect normalized name, account or actor")
				}
				return &controller_admin.AssistantResult{ID: 42, Username: req.Username, DisplayName: req.DisplayName, Role: "assistant"}, nil
			})
			if writer.Code != tc.status {
				t.Fatalf("status = %d, expected %d: %s", writer.Code, tc.status, writer.Body.String())
			}
			if called != (tc.status == http.StatusOK) {
				t.Fatalf("invalid or unauthorized request reached account creation: %t", called)
			}
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    struct {
					DisplayName string `json:"displayName"`
				} `json:"data"`
			}
			if err := json.Unmarshal(writer.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.status == http.StatusOK && (body.Code != 0 || body.Data.DisplayName != tc.wantName) {
				t.Fatal("response did not preserve the confirmed nickname")
			}
			if strings.Contains(body.Message, "createAssistantReq") || strings.Contains(body.Message, "Field validation") || strings.Contains(body.Message, "abcdefgh") {
				t.Fatal("validation leaked internal structure or a password")
			}
		})
	}
}

func TestCreateAssistantBusinessErrorPreserved(t *testing.T) {
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	c.Set("admin_role", "admin")
	c.Request = httptest.NewRequest(http.MethodPost, "/assistants", strings.NewReader(`{"username":"tester","password":"abcdefgh"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	createAssistant(c, func(controller_admin.CreateAssistantReq) (*controller_admin.AssistantResult, error) {
		return nil, errors.New("用户名已存在")
	})
	if writer.Code != http.StatusBadRequest || !strings.Contains(writer.Body.String(), "用户名已存在") {
		t.Fatal("a business failure must not become success")
	}
}
