package handler_webhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mysqlconfig "github.com/go-sql-driver/mysql"
	adminhandler "github.com/ixayda/ix-Go/internal/app/server/handler/admin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	applog "github.com/ixayda/ix-Go/pkg/logger"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOwnerCallbackMySQL(t *testing.T) {
	raw := os.Getenv("RUI_OWNER_MYSQL_DSN")
	if raw == "" {
		t.Skip("requires disposable loopback MySQL")
	}
	cfg, err := mysqlconfig.ParseDSN(raw)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !strings.HasPrefix(cfg.DBName, "rui_owner_test") {
		t.Fatal("refusing non-isolated database")
	}
	cfg.DBName = ""
	admin, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated MySQL unavailable")
	}
	name := "rui_owner_test_callback_" + repository.OwnershipID()[:8]
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	addr := strings.Split(cfg.Addr, ":")
	for key, value := range map[string]string{"MYSQL_ADDR": addr[0], "MYSQL_PORT": addr[1], "MYSQL_USER": cfg.User, "MYSQL_PASSWD": cfg.Passwd, "MYSQL_DATABASE": name, "REDIS_ENABLED": "false", "TENCENT_IM_SDK_APP_ID": "7", "TENCENT_IM_CALLBACK_TOKEN": "callback-fixture", "LOG_DIR": t.TempDir()} {
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
		c, _ := admin.DB()
		_ = c.Close()
	})
	if err := db.AutoMigrate(&model.AdminUser{}, &model.User{}, &model.InviteCode{}, &model.Group{}, &model.FriendOwnerOperation{}, &model.AdminChatQuietHoursPolicy{}, &model.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	if err := repository.EnsureOwnershipSchema(db); err != nil {
		t.Fatal(err)
	}
	for _, a := range []model.AdminUser{{ID: 100, Username: "super", Role: "super_admin", Status: "active"}, {ID: 110, Username: "A", Role: "admin", Status: "active"}, {ID: 120, Username: "B", Role: "admin", Status: "active"}} {
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []model.User{{ID: 201, BizID: 201, Phone: "pool", IMUserID: "pool", Nickname: "pool", Status: "active"}, {ID: 202, BizID: 202, Phone: "hidden-a", IMUserID: "hidden-a", Nickname: "hidden", OwnerID: 110, Status: "active", GloballySearchable: false}} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	post := func(command string, fields map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		fields["CallbackCommand"] = command
		body, _ := json.Marshal(fields)
		stamp := strconv.FormatInt(time.Now().Unix(), 10)
		sign := sha256.Sum256([]byte("callback-fixture" + stamp))
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", "/?SdkAppid=7&CallbackCommand="+command+"&RequestTime="+stamp+"&Sign="+hex.EncodeToString(sign[:]), bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		TencentCallback(c)
		return rec
	}
	response := func(from, to, action string) map[string]any {
		return map[string]any{"From_Account": from, "ResponseFriendItem": []map[string]string{{"To_Account": to, "ResponseAction": action}}}
	}
	checkResult := func(rec *httptest.ResponseRecorder, code int) {
		t.Helper()
		var result prevFriendAddResp
		if json.Unmarshal(rec.Body.Bytes(), &result) != nil || rec.Code != 200 || len(result.ResultItem) != 1 || result.ResultItem[0].ResultCode != code {
			t.Fatalf("unexpected callback result HTTP %d: %s", rec.Code, rec.Body.String())
		}
	}
	checkResult(post(cmdPrevFriendAdd, map[string]any{"From_Account": "hidden-a", "AddFriendItem": []map[string]string{{"To_Account": "pool"}}}), 0)
	checkResult(post(cmdPrevFriendResponse, response("pool", "hidden-a", "Response_Action_AgreeAndAdd")), 0)
	var pool model.User
	db.First(&pool, 201)
	if pool.OwnerID != 0 || pool.OwnerRevision != 0 {
		t.Fatal("pre callbacks assigned before Tencent confirmation")
	}
	r := &repository.OwnershipRepository{DB: db}
	if _, err := r.Transfer(context.Background(), 100, 201, 120); err != nil {
		t.Fatal(err)
	}
	checkResult(post(cmdPrevFriendResponse, response("pool", "hidden-a", "Response_Action_AgreeAndAdd")), errCodeCrossOwner)
	checkResult(post(cmdPrevFriendResponse, response("pool", "hidden-a", "Response_Action_Reject")), 0)
	callback := map[string]any{"ClientCmd": "FriendResponse", "PairList": []map[string]string{{"From_Account": "hidden-a", "To_Account": "pool", "Initiator_Account": "hidden-a"}}}
	for i := 0; i < 2; i++ {
		if rec := post(cmdFriendAdd, callback); rec.Code != 200 {
			t.Fatal("durable callback was not acknowledged")
		}
	}
	var count int64
	db.Model(&model.FriendOwnerOperation{}).Where("source='callback'").Count(&count)
	if count != 1 {
		t.Fatal("replayed callback was not idempotent")
	}
	db.First(&pool, 201)
	if pool.OwnerID != 120 || pool.OwnerRevision != 1 {
		t.Fatal("success callback synchronously overwrote a different space")
	}
	if err := db.Create(&model.Group{ID: 301, BizID: 301, IMGroupID: "independent-test", CreatedBy: 201, OwnerID: 110, OwnerBinding: "independent", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	if owned, err := repairGroupOwnerToOperator("independent-test", &pool); err != nil || owned {
		t.Fatal("preserved independent group was falsely reported as reassigned")
	}
	var job model.FriendOwnerOperation
	db.Where("source='callback'").First(&job)
	if job.State != "pending" {
		t.Fatal("callback claimed completion without bilateral proof")
	}
	if err := db.Model(&job).Updates(map[string]any{"state": "conflict", "reason_code": "cross_owner", "result_json": `{"state":"conflict","request_id":"prior"}`}).Error; err != nil {
		t.Fatal(err)
	}
	manual := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(manual)
	c.Set("admin_id", uint(100))
	c.Set("admin_role", "super_admin")
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(job.ID, 10)}}
	c.Request = httptest.NewRequest("POST", "/retry", nil)
	adminhandler.RetryOwnershipOperation(c)
	if manual.Code != 200 {
		t.Fatal("manual re-evaluation was not queued")
	}
	db.First(&job, job.ID)
	assignment, err := r.Assignment(context.Background(), job.ID)
	var history []struct {
		State string `json:"state"`
	}
	if err != nil || assignment.State != "pending" || job.ManualRetries != 1 || job.LastRetriedBy != 100 || json.Unmarshal([]byte(job.RetryHistory), &history) != nil || len(history) != 1 || history[0].State != "conflict" {
		t.Fatal("manual retry lost audit or returned the stale terminal result")
	}
	if err := db.Migrator().DropTable(&model.FriendOwnerOperation{}); err != nil {
		t.Fatal(err)
	}
	if rec := post(cmdFriendAdd, callback); rec.Code != 503 {
		t.Fatal("callback acknowledged a task that was not durably saved")
	}
}
