package handler_chat

import (
	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/pkg/base/response"
)

// Ping 提供轻量探活接口，供客户端域名池切换前探测使用。
// GET /api/v1/chat/ping
func Ping(c *gin.Context) {
	response.OK(c, gin.H{
		"status": "ok",
	})
}
