package handler_webhook

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/pkg/config"
)

// Official IM callback authentication: Sign=sha256(Token+RequestTime), within
// one minute. The SDK application and query/body command must also agree.
func validateTencentCallback(app int, token, appValue, requestTime, sign string, now time.Time) bool {
	id, err := strconv.Atoi(appValue)
	if err != nil || id != app || app <= 0 || token == "" {
		return false
	}
	stamp, err := strconv.ParseInt(requestTime, 10, 64)
	if err != nil || stamp < now.Unix()-60 || stamp > now.Unix()+60 {
		return false
	}
	supplied, err := hex.DecodeString(sign)
	if err != nil || len(supplied) != 32 {
		return false
	}
	expected := sha256.Sum256([]byte(token + requestTime))
	return subtle.ConstantTimeCompare(supplied, expected[:]) == 1
}

func authenticateTencentCallback(c *gin.Context) bool {
	cfg := config.Cfg.Get().TencentIM
	return authenticateTencentCallbackWith(c, cfg, time.Now())
}

func authenticateTencentCallbackWith(c *gin.Context, cfg config.TencentIM, now time.Time) bool {
	if cfg.CallbackToken == "" {
		c.JSON(http.StatusServiceUnavailable, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "callback authentication not configured"})
		return false
	}
	if !validateTencentCallback(cfg.SDKAppID, cfg.CallbackToken, c.Query("SdkAppid"), c.Query("RequestTime"), c.Query("Sign"), now) {
		c.JSON(http.StatusForbidden, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid callback authentication"})
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		c.JSON(400, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid callback body"})
		return false
	}
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		c.JSON(400, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid callback body"})
		return false
	}
	command, _ := object["CallbackCommand"].(string)
	query := strings.TrimSpace(c.Query("CallbackCommand"))
	if command == "" || query != "" && query != command {
		c.JSON(400, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "callback command mismatch"})
		return false
	}
	canonical, _ := json.Marshal(object)
	sum := sha256.Sum256(canonical)
	c.Set("owner_callback_event", hex.EncodeToString(sum[:]))
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return true
}
