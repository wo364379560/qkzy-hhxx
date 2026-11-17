package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

const (
	ipAccessBlockCacheTTL = 15 * time.Second
	ipAccessBlockCacheMax = 10000
)

type ipAccessBlockCacheEntry struct {
	blocked   bool
	expiresAt time.Time
}

var ipAccessBlockCache = struct {
	sync.Mutex
	entries map[string]ipAccessBlockCacheEntry
}{entries: make(map[string]ipAccessBlockCacheEntry)}

// IPAccessBlock enforces application-layer IP blocks on unauthenticated high-risk endpoints.
func IPAccessBlock() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !shouldCheckIPAccessBlock(c.Request.Method, c.FullPath(), c.Request.URL.Path) {
			c.Next()
			return
		}

		ipAddress := clientip.FromContext(c)
		blocked, cached := getCachedIPAccessBlock(ipAddress, time.Now())
		if !cached {
			var err error
			blocked, err = isIPAccessBlocked(c.Request.Context(), ipAddress)
			if err == nil {
				cacheIPAccessBlock(ipAddress, blocked, time.Now())
			}
			if err != nil {
				c.Next()
				return
			}
		}
		if blocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "访问已被限制",
			})
			return
		}
		c.Next()
	}
}

func getCachedIPAccessBlock(ipAddress string, now time.Time) (bool, bool) {
	if ipAddress == "" {
		return false, false
	}
	ipAccessBlockCache.Lock()
	defer ipAccessBlockCache.Unlock()
	entry, exists := ipAccessBlockCache.entries[ipAddress]
	if !exists {
		return false, false
	}
	if !now.Before(entry.expiresAt) {
		delete(ipAccessBlockCache.entries, ipAddress)
		return false, false
	}
	return entry.blocked, true
}

func cacheIPAccessBlock(ipAddress string, blocked bool, now time.Time) {
	if ipAddress == "" {
		return
	}
	ipAccessBlockCache.Lock()
	defer ipAccessBlockCache.Unlock()
	if len(ipAccessBlockCache.entries) >= ipAccessBlockCacheMax {
		for cachedIP, entry := range ipAccessBlockCache.entries {
			if !now.Before(entry.expiresAt) {
				delete(ipAccessBlockCache.entries, cachedIP)
			}
		}
		if len(ipAccessBlockCache.entries) >= ipAccessBlockCacheMax {
			return
		}
	}
	ipAccessBlockCache.entries[ipAddress] = ipAccessBlockCacheEntry{
		blocked:   blocked,
		expiresAt: now.Add(ipAccessBlockCacheTTL),
	}
}

func shouldCheckIPAccessBlock(method, fullPath, requestPath string) bool {
	path := requestPath
	if path == "" {
		path = fullPath
	}
	policy, protected := PublicPolicyFor(method, path)
	if !protected {
		return false
	}
	switch policy.Action {
	case "login",
		"register",
		"sms_send",
		"password_reset",
		"admin_auth":
		return true
	default:
		return false
	}
}

func isIPAccessBlocked(ctx context.Context, ipAddress string) (bool, error) {
	if ipAddress == "" {
		return false, nil
	}
	var count int64
	err := datasource.Datasource.Mysql.GetClient().
		WithContext(ctx).
		Model(&model.IPAccessBlock{}).
		Where("ip_address = ? AND status = ?", ipAddress, "active").
		Count(&count).Error
	return count > 0, err
}
