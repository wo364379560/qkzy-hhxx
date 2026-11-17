package handler_webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/pkg/config"
)

func TestTencentCallbackAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Unix(1800000000, 0)
	token := "callback-test-only-key"
	stamp := strconv.FormatInt(now.Unix(), 10)
	sig := sha256.Sum256([]byte(token + stamp))
	signed := hex.EncodeToString(sig[:])
	for _, tc := range []struct {
		name, app, at, sign, body, command, key string
		want                                    int
	}{
		{"valid", "7", stamp, signed, `{"CallbackCommand":"Sns.CallbackFriendAdd"}`, "Sns.CallbackFriendAdd", token, 200},
		{"unsigned", "7", stamp, "", `{}`, "", token, 403},
		{"wrong application", "8", stamp, signed, `{}`, "", token, 403},
		{"wrong signature", "7", stamp, strings.Repeat("0", 64), `{}`, "", token, 403},
		{"old request", "7", strconv.FormatInt(now.Unix()-61, 10), signed, `{}`, "", token, 403},
		{"future request", "7", strconv.FormatInt(now.Unix()+61, 10), signed, `{}`, "", token, 403},
		{"unconfigured", "7", stamp, signed, `{}`, "", "", 503},
		{"invalid JSON", "7", stamp, signed, `{`, "", token, 400},
		{"command mismatch", "7", stamp, signed, `{"CallbackCommand":"Sns.CallbackFriendAdd"}`, "State.StateChange", token, 400},
		{"missing command", "7", stamp, signed, `{}`, "Sns.CallbackFriendAdd", token, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			c.Request = httptest.NewRequest("POST", "/?SdkAppid="+tc.app+"&RequestTime="+tc.at+"&Sign="+tc.sign+"&CallbackCommand="+tc.command, strings.NewReader(tc.body))
			valid := authenticateTencentCallbackWith(c, config.TencentIM{SDKAppID: 7, CallbackToken: tc.key}, now)
			if valid {
				c.Status(200)
				if c.GetString("owner_callback_event") == "" {
					t.Fatal("no stable canonical event key")
				}
			}
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d", response.Code, tc.want)
			}
		})
	}
}
