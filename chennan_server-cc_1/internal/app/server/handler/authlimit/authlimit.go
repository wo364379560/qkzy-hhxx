package authlimit

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

const retryAfterSeconds = "60"

func AbortIfAuthWorkBusy(c *gin.Context, err error) bool {
	if !errors.Is(err, authwork.ErrBusy) {
		return false
	}
	ratelimit.LogOverload("bcrypt", "cpu_gate_full", clientip.FromContext(c))
	handlers.MarkAccessLogSampled(c)
	c.Header("Retry-After", "1")
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, response.Response{
		Code:    http.StatusServiceUnavailable,
		Message: "认证服务繁忙，请稍后重试",
	})
	return true
}

// Enforce applies all public-auth rate-limit dimensions before the request can
// reach database lookups, external verification, or password hashing.
func Enforce(c *gin.Context, action string, subjects []ratelimit.SubjectRules) bool {
	decision, err := ratelimit.CheckSubjects(c.Request.Context(), subjects)
	return enforceDecision(c, action, decision, err)
}

func enforceDecision(c *gin.Context, action string, decision ratelimit.Decision, err error) bool {
	if err == nil && decision.Allowed {
		return true
	}
	if err != nil && !errors.Is(err, ratelimit.ErrLimited) {
		_ = c.Error(err)
		ratelimit.LogOverload(action, "rate_limit_check_failed", clientip.FromContext(c))
		handlers.MarkAccessLogSampled(c)
		c.Header("Retry-After", "1")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, response.Response{
			Code:    http.StatusServiceUnavailable,
			Message: "服务繁忙，请稍后重试",
		})
		return false
	}

	ratelimit.LogLimited(action, "multiple", clientip.FromContext(c), decision)
	handlers.MarkAccessLogSampled(c)
	c.Header("Retry-After", retryAfterSeconds)
	c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Response{
		Code:    http.StatusTooManyRequests,
		Message: ratelimit.DefaultMessage,
	})
	return false
}
