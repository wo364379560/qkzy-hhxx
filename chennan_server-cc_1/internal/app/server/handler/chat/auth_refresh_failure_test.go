package handler_chat

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRefreshFailureResponseIsCompatibleAndDoesNotLeakInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err       error
		status    int
		code      string
		retryable bool
	}{
		{controller_chat.AuthErr("invalid session"), 401, "SESSION_INVALID", false},
		{errors.New("database credentials/token detail"), 503, "SESSION_TEMPORARY", true},
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		writeRefreshFailure(c, tc.err, "server-generated-id", time.Now())
		var body struct {
			Code int                            `json:"code"`
			Data controller_chat.RefreshFailure `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tc.status || body.Code != tc.status || body.Data.ErrorCode != tc.code || body.Data.Retryable != tc.retryable {
			t.Fatalf("response=%s status=%d", rec.Body.String(), rec.Code)
		}
		if strings.Contains(rec.Body.String(), "credentials") {
			t.Fatal("internal error leaked")
		}
		if tc.retryable && rec.Header().Get("Retry-After") == "" {
			t.Fatal("temporary failure must supply retry hint")
		}
	}
}
