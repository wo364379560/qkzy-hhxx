package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/pkg/config"
)

func ginCORS() gin.HandlerFunc {
	return corsMiddleware(config.Cfg.Get().App.CORSAllowedOrigins)
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowedOriginSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		allowedOriginSet[origin] = struct{}{}
	}

	_, allowAll := allowedOriginSet["*"]

	return func(ctx *gin.Context) {
		origin := strings.TrimSpace(ctx.GetHeader("Origin"))
		if origin == "" {
			ctx.Next()
			return
		}

		if allowAll {
			ctx.Header("Access-Control-Allow-Origin", "*")
		} else {
			if !corsOriginAllowed(origin, allowedOriginSet) {
				if ctx.Request.Method == http.MethodOptions {
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				ctx.Next()
				return
			}
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Origin, Accept, X-Requested-With")
		ctx.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		ctx.Header("Access-Control-Max-Age", "86400")

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

func corsOriginAllowed(origin string, allowedOriginSet map[string]struct{}) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || len(allowedOriginSet) == 0 {
		return false
	}
	if _, ok := allowedOriginSet[origin]; ok {
		return true
	}
	for pattern := range allowedOriginSet {
		if corsOriginMatchesPattern(origin, pattern) {
			return true
		}
	}
	return false
}

func corsOriginMatchesPattern(origin, pattern string) bool {
	origin = strings.TrimSpace(origin)
	pattern = strings.TrimSpace(pattern)
	if origin == "" || pattern == "" || !strings.Contains(pattern, "*") {
		return false
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	patternURL, err := url.Parse(pattern)
	if err != nil {
		return false
	}
	if originURL.Scheme == "" || originURL.Host == "" || originURL.User != nil ||
		originURL.Opaque != "" || originURL.RawQuery != "" || originURL.Fragment != "" {
		return false
	}
	if originURL.Path != "" && originURL.Path != "/" {
		return false
	}
	if originURL.Scheme != patternURL.Scheme {
		return false
	}
	if originURL.Port() != patternURL.Port() {
		return false
	}

	originHost := strings.ToLower(originURL.Hostname())
	patternHost := strings.ToLower(patternURL.Hostname())
	if !strings.HasPrefix(patternHost, "*.") {
		return false
	}
	suffix := patternHost[1:]
	if !strings.HasSuffix(originHost, suffix) {
		return false
	}
	prefix := strings.TrimSuffix(originHost, suffix)
	return prefix != "" && !strings.Contains(prefix, ".")
}
