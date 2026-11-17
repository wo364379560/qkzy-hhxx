package handler_webhook

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mysqlconfig "github.com/go-sql-driver/mysql"
	chat "github.com/ixayda/ix-Go/internal/app/server/handler/chat"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	applog "github.com/ixayda/ix-Go/pkg/logger"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests use a disposable loopback database and intercept every Tencent
// request in memory. No real group or production user can be modified.
func groupAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	raw := os.Getenv("RUI_GROUP_ADMIN_MYSQL_DSN")
	if raw == "" {
		t.Skip("requires disposable loopback MySQL")
	}
	cfg, err := mysqlconfig.ParseDSN(raw)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !strings.HasPrefix(cfg.DBName, "rui_group_admin_test") {
		t.Fatal("refusing non-isolated database")
	}
	cfg.DBName = ""
	admin, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated MySQL unavailable")
	}
	name := "rui_group_admin_test_" + repository.OwnershipID()[:12]
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	addr := strings.Split(cfg.Addr, ":")
	for key, value := range map[string]string{
		"MYSQL_ADDR": addr[0], "MYSQL_PORT": addr[1], "MYSQL_USER": cfg.User,
		"MYSQL_PASSWD": cfg.Passwd, "MYSQL_DATABASE": name, "REDIS_ENABLED": "false",
		"TENCENT_IM_SDK_APP_ID": "7", "TENCENT_IM_APP_KEY": "group-test-only-key",
		"TENCENT_IM_ADMIN_USER": "group-test-admin", "TENCENT_IM_CALLBACK_TOKEN": "group-test-callback",
		"LOG_DIR": t.TempDir(),
	} {
		t.Setenv(key, value)
	}
	if err := config.Setup(); err != nil {
		t.Fatal(err)
	}
	applog.Init()
	datasource.Datasource.Init()
	db := datasource.Datasource.Mysql.GetClient()
	t.Cleanup(func() {
		conn, _ := db.DB()
		_ = conn.Close()
		_ = admin.Exec("DROP DATABASE " + name).Error
		conn, _ = admin.DB()
		_ = conn.Close()
	})
	if err := db.AutoMigrate(&model.User{}, &model.Group{}, &model.GroupBusinessAdmin{}); err != nil {
		t.Fatal(err)
	}
	return db
}

type groupRoleTransport func(*http.Request) (*http.Response, error)

func (f groupRoleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGroupAdminSyncMySQL(t *testing.T) {
	db := groupAdminTestDB(t)
	gin.SetMode(gin.TestMode)
	const groupID = "role-sync-test"
	for i, id := range []string{"owner", "admin", "member", "peer", "business", "foreign"} {
		ownerID := uint(101)
		if id == "foreign" {
			ownerID = 102
		}
		if err := db.Create(&model.User{ID: uint(i + 1), BizID: int64(i + 1), Phone: id, IMUserID: id, Nickname: id, OwnerID: ownerID, Status: "active"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.Group{BizID: 1, IMGroupID: groupID, Name: "Test", GroupType: "Public", OwnerID: 101, CreatedBy: 1, Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	var roles map[string]string
	var deletes, roleReads int
	var rolesFail, ignoreModify bool
	var modifyError string
	reply := func(body map[string]any) (*http.Response, error) {
		body["ActionStatus"], body["ErrorCode"] = "OK", 0
		if msg, ok := body["ErrorInfo"]; ok {
			body["ActionStatus"], body["ErrorCode"] = "FAIL", 10007
			body["ErrorInfo"] = msg
		}
		data, _ := json.Marshal(body)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(data))}, nil
	}
	oldTransport := http.DefaultTransport
	http.DefaultTransport = groupRoleTransport(func(r *http.Request) (*http.Response, error) {
		var req struct {
			UserAccount []string `json:"User_Account"`
			Account     string   `json:"Member_Account"`
			Role        string   `json:"Role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
		switch r.URL.Path {
		case "/v4/group_open_http_svc/get_group_info":
			return reply(map[string]any{"GroupInfo": []map[string]any{{"GroupId": groupID, "Name": "Test", "Type": "Public", "Owner_Account": "owner"}}})
		case "/v4/group_open_http_svc/get_role_in_group":
			roleReads++
			if rolesFail {
				return nil, errors.New("fixture unavailable")
			}
			items := []map[string]any{}
			for _, id := range req.UserAccount {
				if role, ok := roles[id]; ok {
					items = append(items, map[string]any{"Member_Account": id, "Role": role})
				}
			}
			return reply(map[string]any{"UserIdList": items})
		case "/v4/group_open_http_svc/get_group_member_info":
			items := []map[string]any{}
			for id, role := range roles {
				if role != "NotMember" {
					items = append(items, map[string]any{"Member_Account": id, "Role": role})
				}
			}
			return reply(map[string]any{"MemberList": items, "MemberNum": len(items)})
		case "/v4/group_open_http_svc/modify_group_member_info":
			if modifyError != "" {
				return reply(map[string]any{"ErrorInfo": modifyError})
			}
			if !ignoreModify {
				roles[req.Account] = req.Role
			}
			return reply(map[string]any{})
		case "/v4/group_open_http_svc/delete_group_member":
			deletes++
			return reply(map[string]any{})
		default:
			return nil, fmt.Errorf("unexpected Tencent command: %s", r.URL.Path)
		}
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	repo := repository.NewGroupBusinessAdminRepository()
	reset := func(t *testing.T) {
		t.Helper()
		if err := db.Where("group_id = ?", groupID).Delete(&model.GroupBusinessAdmin{}).Error; err != nil {
			t.Fatal(err)
		}
		roles = map[string]string{"owner": "Owner", "admin": "Admin", "peer": "Admin", "member": "Member", "business": "Member", "foreign": "Admin"}
		deletes, roleReads, rolesFail, ignoreModify, modifyError = 0, 0, false, false, ""
		// Reproduce the old shortcut: an owner is already synced, but the new
		// Tencent Admin has no corresponding local administrator record.
		if _, err := repo.UpsertActiveRole(groupID, "owner", "owner", repository.GroupAdminSourceTencentRoleSync, repository.GroupRoleOwner); err != nil {
			t.Fatal(err)
		}
	}
	grant := func(t *testing.T, userID, role string) {
		t.Helper()
		if _, err := repo.UpsertActiveRole(groupID, userID, "owner", "test", role); err != nil {
			t.Fatal(err)
		}
	}
	assertRole := func(t *testing.T, userID, role string) {
		t.Helper()
		row, err := repo.FindActive(groupID, userID)
		if role == "" {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("unexpected active grant for %s: row=%+v err=%v", userID, row, err)
			}
			return
		}
		if err != nil || row.Role != role {
			t.Fatalf("role for %s: row=%+v err=%v want=%s", userID, row, err, role)
		}
	}
	request := func(method, path, caller, body string) *httptest.ResponseRecorder {
		router := gin.New()
		router.Use(func(c *gin.Context) { c.Set("im_user_id", caller) }) // AuthChat's verified identity
		router.DELETE("/api/v1/chat/groups/:group_id/members/:user_id", chat.RemoveGroupMember)
		router.GET("/api/v1/chat/groups/:group_id/admins", chat.GetGroupAdmins)
		router.POST("/api/v1/chat/groups/:group_id/admins", chat.AddGroupAdmin)
		router.DELETE("/api/v1/chat/groups/:group_id/admins/:user_id", chat.RemoveGroupAdmin)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		return rec
	}
	for _, tc := range []struct {
		name, caller, target, callerRole, localCallerRole, localTargetRole, want string
	}{
		{"native_admin_missing_local", "admin", "member", "Admin", "", "", ""},
		{"ordinary_member_denied", "member", "business", "Member", "", "", "没有群管理权限"},
		{"revoked_native_admin_denied", "admin", "member", "Member", repository.GroupRoleTencentAdmin, "", "没有群管理权限"},
		{"departed_business_admin_denied", "business", "member", "NotMember", repository.GroupRoleBusinessAdmin, "", "没有群管理权限"},
		{"business_admin_allowed", "business", "member", "Member", repository.GroupRoleBusinessAdmin, "", ""},
		{"cross_owner_denied", "foreign", "member", "Admin", "", "", "群组不存在"},
		{"admin_cannot_kick_owner", "admin", "owner", "Admin", "", "", "不能操作群主"},
		{"admin_cannot_kick_unmirrored_admin", "admin", "peer", "Admin", "", "", "不能操作群管理员"},
		{"admin_cannot_kick_business_admin", "admin", "business", "Admin", "", repository.GroupRoleBusinessAdmin, "不能操作业务管理员"},
		{"business_cannot_kick_native_admin", "business", "admin", "Member", repository.GroupRoleBusinessAdmin, "", "不能操作群管理员"},
		{"owner_can_kick_admin", "owner", "admin", "Owner", "", "", ""},
		{"owner_can_kick_business_admin", "owner", "business", "Owner", "", repository.GroupRoleBusinessAdmin, ""},
		{"cannot_kick_self", "admin", "admin", "Admin", "", "", "不能移除自己"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset(t)
			roles[tc.caller] = tc.callerRole
			if tc.localCallerRole != "" {
				grant(t, tc.caller, tc.localCallerRole)
			}
			if tc.localTargetRole != "" {
				grant(t, tc.target, tc.localTargetRole)
			}
			rec := request("DELETE", "/api/v1/chat/groups/"+groupID+"/members/"+tc.target, tc.caller, "")
			if tc.want == "" {
				if rec.Code != 200 || deletes != 1 {
					t.Fatalf("authorized removal failed: status=%d deletes=%d body=%s", rec.Code, deletes, rec.Body)
				}
			} else if rec.Code != 400 || deletes != 0 || !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("unauthorized operation: status=%d deletes=%d body=%s", rec.Code, deletes, rec.Body)
			}
			if tc.name == "native_admin_missing_local" {
				assertRole(t, "admin", repository.GroupRoleTencentAdmin)
			}
			if tc.name == "revoked_native_admin_denied" || tc.name == "departed_business_admin_denied" {
				assertRole(t, tc.caller, "")
			}
		})
	}
	for _, failure := range []string{"missing_role", "provider_failure", "non_member_target", "database_failure"} {
		t.Run(failure, func(t *testing.T) {
			reset(t)
			grant(t, "admin", repository.GroupRoleTencentAdmin)
			switch failure {
			case "missing_role":
				delete(roles, "admin")
			case "provider_failure":
				rolesFail = true
			case "non_member_target":
				roles["member"] = "NotMember"
			case "database_failure":
				if err := db.Exec("CREATE TRIGGER reject_role_sync BEFORE INSERT ON group_business_admins FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected_role_sync_failure'").Error; err != nil {
					t.Fatal(err)
				}
				defer db.Exec("DROP TRIGGER reject_role_sync")
			}
			rec := request("DELETE", "/api/v1/chat/groups/"+groupID+"/members/member", "admin", "")
			if rec.Code == 200 || deletes != 0 {
				t.Fatalf("failure authorized removal: %d %s", rec.Code, rec.Body)
			}
		})
	}
	t.Run("admin_list_refreshes_even_with_synced_owner", func(t *testing.T) {
		reset(t)
		grant(t, "business", repository.GroupRoleBusinessAdmin)
		rec := request("GET", "/api/v1/chat/groups/"+groupID+"/admins", "admin", "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"user_id":"admin"`) || !strings.Contains(rec.Body.String(), `"role_source":"business"`) {
			t.Fatalf("stale admin list: %s", rec.Body)
		}
		assertRole(t, "admin", repository.GroupRoleTencentAdmin)
		roles["admin"] = "Member"
		rec = request("GET", "/api/v1/chat/groups/"+groupID+"/admins", "member", "")
		if rec.Code != 200 || strings.Contains(rec.Body.String(), `"user_id":"admin"`) {
			t.Fatalf("revoked admin retained: %s", rec.Body)
		}
		assertRole(t, "admin", "")
		assertRole(t, "business", repository.GroupRoleBusinessAdmin)
	})
	for _, mode := range []string{"confirmed", "ignored_modify", "permission_error", "quota_overflow"} {
		t.Run("set_admin_"+mode, func(t *testing.T) {
			reset(t)
			switch mode {
			case "ignored_modify":
				ignoreModify = true
			case "permission_error":
				modifyError = "permission denied"
			case "quota_overflow":
				modifyError = "number of admin reaches the limit"
			}
			rec := request("POST", "/api/v1/chat/groups/"+groupID+"/admins", "owner", `{"user_id":"member"}`)
			if mode == "confirmed" || mode == "quota_overflow" {
				if rec.Code != 200 {
					t.Fatalf("setting admin failed: %s", rec.Body)
				}
				want := repository.GroupRoleTencentAdmin
				if mode == "quota_overflow" {
					want = repository.GroupRoleBusinessAdmin
				}
				assertRole(t, "member", want)
				modifyError = ""
				rec = request("DELETE", "/api/v1/chat/groups/"+groupID+"/admins/member", "owner", "")
				if rec.Code != 200 {
					t.Fatalf("revoking admin failed: %s", rec.Body)
				}
				assertRole(t, "member", "")
			} else {
				if rec.Code == 200 {
					t.Fatalf("unconfirmed grant accepted: %s", rec.Body)
				}
				assertRole(t, "member", "")
			}
		})
	}
	callback := func(role, sign string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"CallbackCommand": cmdAfterMemberFieldChanged, "GroupId": groupID, "Modified_Account": "admin", "Operator_Account": "owner", "Role": role})
		stamp := strconv.FormatInt(time.Now().Unix(), 10)
		if sign == "valid" {
			sum := sha256.Sum256([]byte("group-test-callback" + stamp))
			sign = hex.EncodeToString(sum[:])
		}
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", "/api/v1/webhook/tencent?SdkAppid=7&CallbackCommand="+cmdAfterMemberFieldChanged+"&RequestTime="+stamp+"&Sign="+sign, bytes.NewReader(body))
		TencentCallback(c)
		return rec
	}
	t.Run("signed_role_callbacks_refresh_and_do_not_replay_stale_payloads", func(t *testing.T) {
		reset(t)
		for i := 0; i < 2; i++ {
			if rec := callback("Admin", "valid"); rec.Code != 200 {
				t.Fatalf("callback failed: %s", rec.Body)
			}
			assertRole(t, "admin", repository.GroupRoleTencentAdmin)
		}
		roles["admin"] = "Member"
		for _, payloadRole := range []string{"Member", "Admin", "Admin"} {
			if rec := callback(payloadRole, "valid"); rec.Code != 200 {
				t.Fatalf("callback failed: %s", rec.Body)
			}
			assertRole(t, "admin", "")
		}
	})
	t.Run("callback_authentication_and_errors", func(t *testing.T) {
		reset(t)
		if rec := callback("Admin", ""); rec.Code != 403 || roleReads != 0 {
			t.Fatalf("unsigned callback accepted: %d", rec.Code)
		}
		if rec := callback("", "valid"); rec.Code != 200 || roleReads != 0 {
			t.Fatalf("namecard-only callback queried roles: %d", rec.Code)
		}
		rolesFail = true
		if rec := callback("Admin", "valid"); rec.Code != 500 {
			t.Fatalf("sync failure acknowledged as success: %d", rec.Code)
		}
		assertRole(t, "admin", "")
	})
	t.Run("concurrent_api_and_callback_upserts_are_idempotent", func(t *testing.T) {
		reset(t)
		var wg sync.WaitGroup
		errs := make(chan error, 12)
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := repo.UpsertActiveRole(groupID, "admin", "owner", "tencent", repository.GroupRoleTencentAdmin)
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		var count int64
		db.Model(&model.GroupBusinessAdmin{}).Where("group_id = ? AND user_id = ?", groupID, "admin").Count(&count)
		if count != 1 {
			t.Fatalf("duplicate grant rows: %d", count)
		}
		assertRole(t, "admin", repository.GroupRoleTencentAdmin)
	})
	t.Run("full_sync_preserves_business_grants_and_retires_departed_members", func(t *testing.T) {
		reset(t)
		grant(t, "business", repository.GroupRoleBusinessAdmin)
		grant(t, "admin", repository.GroupRoleTencentAdmin)
		grant(t, "gone", repository.GroupRoleBusinessAdmin)
		if err := repo.SyncTencentRoles(groupID, map[string]string{"owner": "Owner", "business": "Member", "admin": "Member"}, "owner", true); err != nil {
			t.Fatal(err)
		}
		assertRole(t, "owner", repository.GroupRoleOwner)
		assertRole(t, "business", repository.GroupRoleBusinessAdmin)
		assertRole(t, "admin", "")
		assertRole(t, "gone", "")
	})
	t.Run("role_sync_rolls_back_all_changes_on_write_failure", func(t *testing.T) {
		reset(t)
		if err := db.Exec("CREATE TRIGGER reject_second_role BEFORE INSERT ON group_business_admins FOR EACH ROW BEGIN IF NEW.user_id = 'z-fail' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected_second_role_failure'; END IF; END").Error; err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TRIGGER reject_second_role")
		err := repo.SyncTencentRoles(groupID, map[string]string{"admin": "Admin", "z-fail": "Admin"}, "owner", false)
		if err == nil {
			t.Fatal("failed transaction reported success")
		}
		assertRole(t, "admin", "")
		assertRole(t, "owner", repository.GroupRoleOwner)
	})
}
