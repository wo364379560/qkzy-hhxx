package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/presence"
)

// TouchLastOnlineOnSuccess records the current user's last online time after a
// business endpoint succeeds. The actual database write is batched asynchronously.
func TouchLastOnlineOnSuccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 || len(c.Errors) > 0 {
			return
		}

		imUserID := c.GetString("im_user_id")
		if imUserID == "" {
			return
		}

		presence.EnqueueLastOnline(imUserID)
	}
}
