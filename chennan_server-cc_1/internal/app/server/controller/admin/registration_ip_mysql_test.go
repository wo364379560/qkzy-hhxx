package controller_admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	handler "github.com/ixayda/ix-Go/internal/app/server/handler/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// Opt-in via the same disposable-loopback helper as the account security suite.
// Real AuthAdmin, handlers/controllers, bcrypt and MySQL transactions execute;
// no production accounts, SMS, Tencent calls or browser sessions are touched.
func TestRegistrationIPAndAdminControlsMySQL(t *testing.T) {
	db := accountSecurityDB(t)
	if err := db.AutoMigrate(&model.AdminAccessAudit{}, &model.UserRefreshSession{}); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	const password = "fixturePasswordOnly123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	superID, managerID := uint(201), uint(202)
	rows := []model.AdminUser{
		{ID: superID, Username: "regsuper", Role: "super_admin"},
		{ID: managerID, Username: "regmanager", Role: "admin", ParentAdminID: &superID},
		{ID: 203, Username: "regassistant", Role: "assistant", ParentAdminID: &managerID},
	}
	tokens := map[uint]string{}
	for _, row := range rows {
		row.PasswordHash, row.Status, row.AuthVersion = string(hash), "active", 1
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		token, err := pkgjwt.Generate(row.ID, "super_admin", 1)
		if err != nil {
			t.Fatal(err)
		}
		// Deliberately overclaim the JWT role: AuthAdmin must use the database role.
		tokens[row.ID] = token
	}
	router := gin.New()
	router.Use(middleware.AdminAccessAudit(), middleware.AuthAdmin())
	router.GET("/api/v1/admin/security/registration-ips", handler.ListRegistrationIPs)
	router.POST("/api/v1/admin/security/registration-ips", handler.ChangeRegistrationIP)
	router.POST("/api/v1/admin/users/batch-global-search", handler.BatchSetUserGlobalSearch)
	router.POST("/api/v1/admin/users/:id/reset-password", handler.ResetUserPassword)
	call := func(actor uint, path string, payload any) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tokens[actor])
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:41000"
		req.Header.Set("X-Real-IP", "8.8.4.4")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	change := func(actor uint, ip, pass string, remove bool) *httptest.ResponseRecorder {
		return call(actor, "/api/v1/admin/security/registration-ips", map[string]any{"ip": ip, "current_password": pass, "note": "隔离验证", "remove": remove})
	}
	assertEntries := func(want int) {
		t.Helper()
		entries, err := repository.ListRegistrationIPs(context.Background(), db)
		if err != nil || len(entries) != want {
			t.Fatalf("entries=%d want=%d err=%v", len(entries), want, err)
		}
	}
	for _, actor := range []uint{202, 203} {
		if rec := change(actor, "8.8.8.8", password, false); rec.Code != 403 {
			t.Fatalf("non-super actor=%d status=%d", actor, rec.Code)
		}
	}
	if rec := change(superID, "8.8.8.8", "wrongPassword", false); rec.Code != 400 {
		t.Fatalf("wrong password status=%d", rec.Code)
	}
	assertEntries(0)
	if rec := change(superID, "8.8.8.8", password, false); rec.Code != 200 {
		t.Fatalf("allowlist add=%s", rec.Body.String())
	}
	assertEntries(1)
	var audits []model.AdminAccessAudit
	if err := db.Where("action_key = ? AND success = ?", "registration_ip.add", true).Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].AdminUserID == nil || *audits[0].AdminUserID != superID || audits[0].TargetUsername != "8.8.8.8" {
		t.Fatalf("successful mutation audit missing or duplicated: count=%d", len(audits))
	}
	raw, _ := json.Marshal(audits)
	if bytes.Contains(raw, []byte(password)) || bytes.Contains(raw, []byte("wrongPassword")) {
		t.Fatal("password leaked into audit")
	}

	// Only simulate an already-exhausted hourly counter. The proxy verification,
	// database allowlist lookup, route middleware and fail-closed behavior are real.
	register := gin.New()
	register.Use(middleware.PublicProtection(middleware.PublicProtectionOptions{Check: func(_ context.Context, subject string, rules []ratelimit.Rule) (ratelimit.Decision, error) {
		if strings.HasPrefix(subject, "ip:") {
			return ratelimit.Decision{Allowed: false, Rule: rules[0], Count: 51, Limit: 50}, ratelimit.ErrLimited
		}
		return ratelimit.Decision{Allowed: true}, nil
	}}))
	register.POST("/api/v1/chat/register", func(c *gin.Context) { c.Status(204) })
	probe := func(peer string) int {
		req := httptest.NewRequest("POST", "/api/v1/chat/register", strings.NewReader(`{}`))
		req.RemoteAddr = peer
		req.Header.Set("X-Real-IP", "8.8.8.8")
		rec := httptest.NewRecorder()
		register.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := probe("127.0.0.1:41000"); got != 204 {
		t.Fatalf("verified allowlisted client=%d", got)
	}
	if got := probe("1.1.1.1:41000"); got != 429 {
		t.Fatalf("unverified source bypassed quota: %d", got)
	}
	if rec := change(superID, "8.8.8.8", password, true); rec.Code != 200 {
		t.Fatalf("delete=%s", rec.Body.String())
	}
	assertEntries(0)
	if got := probe("127.0.0.1:41000"); got != 429 {
		t.Fatalf("deleted exemption remained cached: %d", got)
	}

	// Real MySQL fault injection proves the exemption cannot commit without audit.
	if err := db.Exec("CREATE TRIGGER fail_registration_audit BEFORE INSERT ON admin_access_audits FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='isolated audit failure'").Error; err != nil {
		t.Fatal(err)
	}
	rec := change(superID, "8.8.8.9", password, false)
	if err := db.Exec("DROP TRIGGER fail_registration_audit").Error; err != nil {
		t.Fatal(err)
	}
	if rec.Code != 400 {
		t.Fatalf("unaudited mutation accepted: %d", rec.Code)
	}
	assertEntries(0)

	// MySQL row locks must preserve every successful concurrent addition.
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			failures <- repository.ChangeRegistrationIP(context.Background(), db, superID, fmt.Sprintf("8.8.8.%d", 20+i), "parallel", false, model.AdminAccessAudit{})
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertEntries(12)

	// Existing UI action still reaches the authenticated global-search handler.
	user := model.User{ID: 901, BizID: 901, Phone: "fixture901", IMUserID: "fixture901", Nickname: "fixture", Status: "active", OwnerID: superID, AuthVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		rec := call(superID, "/api/v1/admin/users/batch-global-search", map[string]any{"userIDs": []uint{user.ID}, "enabled": enabled})
		if rec.Code != 200 {
			t.Fatalf("global search response=%s", rec.Body.String())
		}
		var stored model.User
		if err := db.First(&stored, user.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.GloballySearchable != enabled {
			t.Fatalf("global search did not change to %v: %s", enabled, rec.Body.String())
		}
	}

	// Default password comes only from protected server config, never response.
	const resetPassword = "configuredFixtureReset123"
	t.Setenv("ADMIN_USER_RESET_PASSWORD", resetPassword)
	if err := config.Setup(); err != nil {
		t.Fatal(err)
	}
	session := model.UserRefreshSession{UserID: user.ID, RefreshTokenHash: "fixture-refresh-hash", DeviceID: "fixture-device", LastUsedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	rec = call(superID, "/api/v1/admin/users/901/reset-password", map[string]any{"mode": "default", "must_change": true})
	if rec.Code != 200 || strings.Contains(rec.Body.String(), resetPassword) {
		t.Fatalf("reset response=%s", rec.Body.String())
	}
	var stored model.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(resetPassword)) != nil || !stored.MustChangePassword || stored.AuthVersion != 2 {
		t.Fatal("configured password/reset semantics changed")
	}
	if err := db.First(&session, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if session.RevokedAt == nil {
		t.Fatal("password reset failed to revoke refresh session")
	}
	t.Setenv("ADMIN_USER_RESET_PASSWORD", "")
	if err := config.Setup(); err != nil {
		t.Fatal(err)
	}
	if rec := call(superID, "/api/v1/admin/users/901/reset-password", map[string]any{"mode": "default"}); rec.Code != 400 {
		t.Fatal("unset default silently restored hardcoded password")
	}
}
