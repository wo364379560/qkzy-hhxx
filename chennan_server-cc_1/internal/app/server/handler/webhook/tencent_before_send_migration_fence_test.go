package handler_webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBeforeSendRejectsWhileSourceMigrationFenceIsActive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	marker := filepath.Join(t.TempDir(), "fence.json")
	t.Setenv("CHENGNAN_SOURCE_MIGRATION_FENCE_FILE", marker)
	if err := os.WriteFile(marker, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	body := `{"CallbackCommand":"C2C.CallbackBeforeSendMsg","From_Account":"probe","To_Account":"probe-target","MsgBody":[]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/tencent", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request

	handleBeforeSendMsg(context)
	if response.Code != http.StatusOK {
		t.Fatalf("callback returned %d", response.Code)
	}
	var payload beforeSendMsgResp
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ActionStatus != "OK" || payload.ErrorCode != sourceMigrationRejectCodeSendMessage {
		t.Fatalf("unexpected callback response: %+v", payload)
	}
}
