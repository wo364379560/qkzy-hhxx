package controller_admin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mysqlconfig "github.com/go-sql-driver/mysql"
	admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	applog "github.com/ixayda/ix-Go/pkg/logger"
	otp "github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicit opt-in only. Every run creates and drops its own loopback schema.
// No production account, external authenticator, or cloud service is used.
func accountSecurityDB(t *testing.T) *gorm.DB {
	t.Helper()
	raw := os.Getenv("RUI_ADMIN_SECURITY_MYSQL_DSN")
	if raw == "" {
		t.Skip("requires disposable loopback MySQL")
	}
	cfg, err := mysqlconfig.ParseDSN(raw)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !strings.HasPrefix(cfg.DBName, "rui_admin_security_test") {
		t.Fatal("refusing non-isolated database")
	}
	cfg.DBName = ""
	control, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated MySQL unavailable")
	}
	name := "rui_admin_security_test_" + repository.OwnershipID()[:12]
	if err := control.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = control.Exec("DROP DATABASE " + name).Error
		conn, _ := control.DB()
		_ = conn.Close()
	})
	addr := strings.Split(cfg.Addr, ":")
	for key, value := range map[string]string{
		"MYSQL_ADDR": addr[0], "MYSQL_PORT": addr[1], "MYSQL_USER": cfg.User,
		"MYSQL_PASSWD": cfg.Passwd, "MYSQL_DATABASE": name, "REDIS_ENABLED": "false",
		"TURNSTILE_ENABLED": "false", "TELEGRAM_ALERT_ENABLED": "false",
		"JWT_SECRET":          "isolated-admin-security-test-only",
		"TOTP_ENCRYPTION_KEY": strings.Repeat("12", 32), "LOG_DIR": t.TempDir(),
	} {
		t.Setenv(key, value)
	}
	if err := config.Setup(); err != nil {
		t.Fatal(err)
	}
	applog.Init()
	datasource.Datasource.Init()
	db := datasource.Datasource.Mysql.GetClient()
	t.Cleanup(func() { conn, _ := db.DB(); _ = conn.Close() })
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminTOTPBackupCode{}, &model.AdminLoginAudit{},
		&model.AdminUserIPWhitelist{}, &model.AdminChatQuietHoursPolicy{}, &model.SystemSetting{},
		&model.LoginRiskControl{}, &model.User{}, &model.Group{}, &model.InviteCode{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAdminAccountSecurityMySQL(t *testing.T) {
	db := accountSecurityDB(t)
	ctx := context.Background()
	const password = "testPassword123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	superID, managerID := uint(1), uint(2)
	rows := []model.AdminUser{
		{ID: 1, Username: "super1", Role: "super_admin"},
		{ID: 2, Username: "manager2", Role: "admin", ParentAdminID: &superID},
		{ID: 3, Username: "manager3", Role: "admin", ParentAdminID: &superID},
		{ID: 4, Username: "一路向南", Role: "assistant", ParentAdminID: &managerID},
		{ID: 5, Username: "assistant5", Role: "assistant", ParentAdminID: &managerID},
	}
	for _, row := range rows {
		row.PasswordHash, row.DisplayName, row.Status = string(hash), "中文昵称", "active"
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Set2FAEnabledTx(ctx, db, true, superID); err != nil {
		t.Fatal(err)
	}
	login := func(name string) *admin.LoginResult {
		t.Helper()
		result, err := admin.Login(ctx, name, password, "", "127.0.0.1", "isolated-test")
		if err != nil {
			t.Fatalf("login %s: %v", name, err)
		}
		return result
	}
	code := func(secret string) string {
		t.Helper()
		value, err := otp.GenerateCode(secret, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	load := func(id uint) model.AdminUser {
		t.Helper()
		var row model.AdminUser
		if err := db.First(&row, id).Error; err != nil {
			t.Fatal(err)
		}
		return row
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", middleware.AuthAdmin(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	assertSession := func(token string, want int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("session status %d want %d", rec.Code, want)
		}
	}

	t.Run("validation_and_scope", func(t *testing.T) {
		for _, name := range []string{"中文账号", "user_name", "user-name", "user name", "ａｂｃ１２３", "ab", strings.Repeat("a", 33)} {
			if _, err := admin.CreateAssistant(admin.CreateAssistantReq{CreatedBy: 1, Username: name, DisplayName: "助理", Password: password}); err == nil {
				t.Fatalf("assistant accepted invalid login name %q", name)
			}
			if _, err := admin.CreateManager(admin.CreateManagerReq{CreatedBy: 1, Username: name, DisplayName: "管理员", Password: password}); err == nil {
				t.Fatalf("manager accepted invalid login name %q", name)
			}
			if _, err := admin.UpdateAssistantProfile(admin.UpdateAssistantProfileReq{ActorID: 1, AssistantID: 4, Username: &name, DisplayName: "助理"}); err == nil {
				t.Fatalf("assistant rename accepted %q", name)
			}
			if _, err := admin.UpdateManagerProfile(admin.UpdateManagerProfileReq{ActorID: 1, ManagerID: 2, Username: &name, DisplayName: "管理员"}); err == nil {
				t.Fatalf("manager rename accepted %q", name)
			}
		}
		created, err := admin.CreateAssistant(admin.CreateAssistantReq{CreatedBy: 2, Username: "  helper6  ", DisplayName: "  中文助理  ", Password: password})
		if err != nil || created.Username != "helper6" || created.DisplayName != "中文助理" {
			t.Fatalf("create assistant: %v", err)
		}
		managed, err := admin.CreateManager(admin.CreateManagerReq{CreatedBy: 1, Username: "  manager7  ", DisplayName: "中文管理员", Password: password})
		if err != nil || managed.Username != "manager7" || managed.DisplayName != "中文管理员" {
			t.Fatalf("create manager: %v", err)
		}
		name := "helper4"
		if _, err := admin.UpdateAssistantProfile(admin.UpdateAssistantProfileReq{ActorID: 3, AssistantID: 4, Username: &name, DisplayName: "越权"}); err == nil {
			t.Fatal("cross-owner rename allowed")
		}
		if err := admin.ResetAssistantTOTP(3, 4); err == nil {
			t.Fatal("cross-owner reset allowed")
		}
		if err := admin.DeleteAssistant(3, 4); err == nil {
			t.Fatal("cross-owner deletion allowed")
		}
		if err := admin.ResetManagerTOTP(2, 3); err == nil {
			t.Fatal("manager resetting another manager allowed")
		}
		name = "manager3"
		if _, err := admin.UpdateAssistantProfile(admin.UpdateAssistantProfileReq{ActorID: 1, AssistantID: 4, Username: &name, DisplayName: "重名"}); err == nil || !strings.Contains(err.Error(), "已存在") {
			t.Fatalf("duplicate name not rejected: %v", err)
		}
		// Older nickname-only clients remain supported, including legacy names.
		if _, err := admin.UpdateAssistantProfile(admin.UpdateAssistantProfileReq{ActorID: 2, AssistantID: 4, DisplayName: "新的中文昵称"}); err != nil {
			t.Fatal(err)
		}
		if load(4).Username != "一路向南" {
			t.Fatal("nickname-only update renamed legacy account")
		}
	})

	// Local TOTP also works for a historical Unicode login name. The new name
	// restriction is a deliberate account policy, not proof of the live failure.
	initial := login("一路向南")
	if !initial.RequireTOTPSetup || initial.TOTPSecret == "" {
		t.Fatal("global 2FA did not require enrollment")
	}
	bound, err := admin.BindTOTP(initial.TempToken, code(initial.TOTPSecret))
	if err != nil || len(bound.BackupCodes) != 8 {
		t.Fatalf("initial enrollment: %v", err)
	}
	assertSession(bound.Token, http.StatusNoContent)
	if _, err := admin.BindTOTP(initial.TempToken, code(initial.TOTPSecret)); err == nil {
		t.Fatal("enrollment replay allowed")
	}

	t.Run("rename_preserves_ownership", func(t *testing.T) {
		for _, row := range []any{
			&model.User{ID: 101, BizID: 101, IMUserID: "test-user", Phone: "test-phone", OwnerID: 4},
			&model.Group{ID: 101, BizID: 101, IMGroupID: "test-group", OwnerID: 4, CreatedBy: 101},
			&model.InviteCode{ID: 101, Code: "test-invite", OwnerID: 4},
		} {
			if err := db.Create(row).Error; err != nil {
				t.Fatal(err)
			}
		}
		before := load(4)
		name := "  helper4  "
		updated, err := admin.UpdateAssistantProfile(admin.UpdateAssistantProfileReq{ActorID: 2, AssistantID: 4, Username: &name, DisplayName: "一路向南"})
		if err != nil || updated.Username != "helper4" || updated.DisplayName != "一路向南" {
			t.Fatalf("rename: %v", err)
		}
		after := load(4)
		if after.ID != before.ID || *after.ParentAdminID != *before.ParentAdminID || after.PasswordHash != before.PasswordHash || after.TOTPSecret != before.TOTPSecret || after.AuthVersion != before.AuthVersion+1 {
			t.Fatal("rename unexpectedly changed identity, ownership, password or TOTP")
		}
		for _, table := range []string{"users", "groups", "invite_codes"} {
			var count int64
			if err := db.Table(table).Where("id = 101 AND owner_id = 4").Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("ownership changed in %s: %v", table, err)
			}
		}
		assertSession(bound.Token, http.StatusUnauthorized)
		for i := 0; i < 2; i++ {
			if _, err := admin.UpdateAssistantProfile(admin.UpdateAssistantProfileReq{ActorID: 2, AssistantID: 4, Username: &name, DisplayName: "一路向南"}); err != nil {
				t.Fatalf("unchanged profile should succeed: %v", err)
			}
		}
		if load(4).AuthVersion != after.AuthVersion {
			t.Fatal("unchanged profile revoked sessions")
		}
	})

	t.Run("reset_revokes_sessions_and_challenges_then_rebinds", func(t *testing.T) {
		pending := login("helper4")
		if !pending.RequireTOTPVerify {
			t.Fatal("renamed account lost valid enrollment")
		}
		verified, err := admin.VerifyTOTP(pending.TempToken, code(initial.TOTPSecret), "")
		if err != nil {
			t.Fatal(err)
		}
		assertSession(verified.Token, http.StatusNoContent)
		if err := admin.ResetAssistantTOTP(1, 4); err != nil {
			t.Fatal(err)
		}
		row := load(4)
		var count int64
		if err := db.Model(&model.AdminTOTPBackupCode{}).Where("admin_id = 4").Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if row.TOTPEnabled || row.TOTPSecret != "" || row.TOTPBoundAt != nil || count != 0 {
			t.Fatal("reset retained enrollment or backup codes")
		}
		assertSession(verified.Token, http.StatusUnauthorized)
		if _, err := admin.VerifyTOTP(pending.TempToken, "", bound.BackupCodes[0]); err == nil {
			t.Fatal("old verification challenge survived reset")
		}
		stale := login("helper4")
		if !stale.RequireTOTPSetup {
			t.Fatal("next login did not require enrollment")
		}
		if err := admin.ResetAssistantTOTP(1, 4); err != nil {
			t.Fatal(err)
		}
		if _, err := admin.BindTOTP(stale.TempToken, code(stale.TOTPSecret)); err == nil {
			t.Fatal("old setup challenge undid reset")
		}
		fresh := login("helper4")
		if !fresh.RequireTOTPSetup || fresh.TOTPSecret == stale.TOTPSecret {
			t.Fatal("fresh enrollment was not generated")
		}
		if _, err := admin.BindTOTP(fresh.TempToken, "bad-code"); err == nil {
			t.Fatal("invalid TOTP accepted")
		}
		rebound, err := admin.BindTOTP(fresh.TempToken, code(fresh.TOTPSecret))
		if err != nil || len(rebound.BackupCodes) != 8 {
			t.Fatalf("fresh enrollment: %v", err)
		}
		assertSession(rebound.Token, http.StatusNoContent)
	})

	t.Run("manager_reset_and_rename", func(t *testing.T) {
		pending := login("manager2")
		if err := admin.ResetManagerTOTP(1, 2); err != nil {
			t.Fatal(err)
		}
		if _, err := admin.BindTOTP(pending.TempToken, code(pending.TOTPSecret)); err == nil {
			t.Fatal("manager setup survived reset")
		}
		name := "newmanager2"
		if _, err := admin.UpdateManagerProfile(admin.UpdateManagerProfileReq{ActorID: 1, ManagerID: 2, Username: &name, DisplayName: "中文管理员"}); err != nil {
			t.Fatal(err)
		}
		if *load(4).ParentAdminID != 2 {
			t.Fatal("manager rename changed assistant parent")
		}
		fresh := login(name)
		if _, err := admin.BindTOTP(fresh.TempToken, code(fresh.TOTPSecret)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("deletion_keeps_dependency_checks", func(t *testing.T) {
		if err := admin.DeleteAssistant(1, 4); err == nil || !strings.Contains(err.Error(), "关联") {
			t.Fatalf("dependent assistant deletion not blocked: %v", err)
		}
		if err := admin.DeleteManager(1, 2); err == nil || !strings.Contains(err.Error(), "助理") {
			t.Fatalf("manager with assistants deletion not blocked: %v", err)
		}
		if err := admin.DeleteAssistant(1, 5); err != nil {
			t.Fatal(err)
		}
		var row model.AdminUser
		if err := db.First(&row, 5).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatal("empty assistant was not deleted")
		}
	})
}
