package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
)

func UserRateLimit(action string, rules []ratelimit.Rule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enforceSubjectRateLimit(c, action, c.GetString("im_user_id"), rules) {
			return
		}
		c.Next()
	}
}

func AdminRateLimit(action string, rules []ratelimit.Rule) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetUint("admin_id")
		subject := "admin:" + strconv.FormatUint(uint64(adminID), 10)
		if !enforceSubjectRateLimit(c, action, subject, rules) {
			return
		}
		c.Next()
	}
}

func enforceSubjectRateLimit(c *gin.Context, action, subject string, rules []ratelimit.Rule) bool {
	decision, err := ratelimit.Check(c.Request.Context(), subject, rules)
	if err == nil && decision.Allowed {
		return true
	}
	if err != nil && !errors.Is(err, ratelimit.ErrLimited) {
		ratelimit.LogOverload(action, "rate_limit_check_failed", clientip.FromContext(c))
		handlers.MarkAccessLogSampled(c)
		c.Header("Retry-After", "1")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "服务繁忙，请稍后再试",
		})
		return false
	}
	ratelimit.LogLimited(action, subject, clientip.FromContext(c), decision)
	handlers.MarkAccessLogSampled(c)
	c.Header("Retry-After", "60")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code":    http.StatusTooManyRequests,
		"message": ratelimit.DefaultMessage,
	})
	return false
}
