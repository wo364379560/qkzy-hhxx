package handlers

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/base/response"
	"github.com/ixayda/ix-Go/pkg/logger"

	"go.uber.org/zap"
)

const sampledAccessLogContextKey = "sampled_access_log"

// MarkAccessLogSampled tells the global access logger that this rejection was
// handled by a dedicated sampled security logger, whether emitted or suppressed.
func MarkAccessLogSampled(ctx *gin.Context) {
	if ctx != nil {
		ctx.Set(sampledAccessLogContextKey, true)
	}
}

// AccessLogSampled reports whether a rejection path already emitted (or was
// intentionally suppressed by) its dedicated sampled security logger.
func AccessLogSampled(ctx *gin.Context) bool {
	return ctx != nil && ctx.GetBool(sampledAccessLogContextKey)
}

func SetHandlers(engine *gin.Engine) {
	appConfig := config.Cfg.Get().App
	if err := engine.SetTrustedProxies(clientip.TrustedProxyCIDRs(appConfig.TrustedProxyCIDRs)); err != nil {
		panic(err)
	}

	engine.TrustedPlatform = ""
	engine.ForwardedByClientIP = true
	engine.RemoteIPHeaders = append([]string(nil), appConfig.RealIPHeaderOrder...)

	// registry gin middleware
	engine.Use(ginCORS(), ginLogger(), ginRecovery())
}

func ginLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		ctx.Next()

		// Stop timer
		cost := time.Since(start)

		// log process
		responStatus := ctx.Writer.Status()
		if AccessLogSampled(ctx) {
			return
		}
		if shouldSkipAccessLog(
			ctx.Request.Method,
			ctx.Request.URL.Path,
			responStatus,
		) {
			return
		}

		logFields := []zap.Field{
			zap.String("ip", clientip.FromContext(ctx)),
			zap.Int("status", responStatus),
			zap.String("method", ctx.Request.Method),
			zap.String("query", truncateLogValue(ctx.Request.URL.Path, 256)),
			zap.String("proto", ctx.Request.Proto),
			zap.String("user-agent", truncateLogValue(ctx.Request.UserAgent(), 256)),
			zap.String("time", cost.String()),
			zap.String("errors", ctx.Errors.ByType(gin.ErrorTypePrivate).String()),
		}

		if responStatus >= 400 && responStatus <= 499 {
			logger.GinWarn(fmt.Sprintf("%s Access [%d]", ctx.Request.Proto, responStatus), logFields...)
		} else if responStatus >= 500 && responStatus <= 599 {
			logger.GinError(fmt.Sprintf("%s Access [%d]", ctx.Request.Proto, responStatus), logFields...)
		} else {
			logger.GinInfo(fmt.Sprintf("%s Access [%d]", ctx.Request.Proto, responStatus), logFields...)
		}
	}
}

func shouldSkipAccessLog(method, path string, status int) bool {
	// Rejected requests are skipped only through MarkAccessLogSampled. Headers
	// such as Retry-After are response semantics and do not prove that another
	// logger recorded the rejection.
	if status >= 400 {
		return false
	}

	if method == http.MethodOptions {
		return true
	}

	if method == http.MethodGet {
		return true
	}

	switch method {
	case http.MethodPost:
		switch path {
		case "/api/v1/chat/auth/refresh",
			"/api/v1/chat/users/last-online":
			return true
		}
	}

	return false
}

func truncateLogValue(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}

func ginRecovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					var se *os.SyscallError
					if errors.As(ne, &se) {
						seStr := strings.ToLower(se.Error())
						if strings.Contains(seStr, "broken pipe") ||
							strings.Contains(seStr, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				httpRequest, _ := httputil.DumpRequest(ctx.Request, false)

				if brokenPipe {
					// If the connection is dead, we can't write a status to it.
					logger.GinError("HTTP broken pipe", zap.Any("error", err), zap.String("request", string(httpRequest)))
					response.InternalError(ctx)
					ctx.Error(err.(error)) //nolint: errcheck
					ctx.Abort()
					return
				} else {
					logger.GinError("Panic recovery", zap.Any("error", err), zap.String("request", string(httpRequest)), zap.Stack("stacktrace"))
					response.InternalError(ctx)
					ctx.Abort()
				}
			}
		}()
		ctx.Next()
	}
}
