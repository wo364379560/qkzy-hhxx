package handler_webhook

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAfterSendAcknowledgesWithoutQuotaStorage(t *testing.T) {
	for _, command := range []string{cmdC2CAfterSendMsg, cmdGroupAfterSendMsg} {
		for n := 0; n < 10; n++ {
			body := `{"CallbackCommand":"` + command + `","From_Account":"unverified","ErrorCode":0}`
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhook/tencent", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			handleAfterSendMsg(c)
			var result beforeSendMsgResp
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || result.ActionStatus != "OK" || result.ErrorCode != 0 {
				t.Fatalf("callback %s #%d: %+v", command, n, result)
			}
		}
	}
}
