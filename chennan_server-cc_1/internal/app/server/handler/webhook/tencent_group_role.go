package handler_webhook

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/logger"
)

// Invoked only after TencentCallback verifies SDKAppID, signature and timestamp.
func handleGroupMemberRoleChanged(c *gin.Context) {
	var req struct {
		GroupID         string `json:"GroupId"`
		ModifiedAccount string `json:"Modified_Account"`
		OperatorAccount string `json:"Operator_Account"`
		Role            string `json:"Role"`
		TopicID         string `json:"TopicId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, callbackResp{ActionStatus: "FAIL", ErrorCode: 400, ErrorInfo: "invalid member change"})
		return
	}
	// Name-card changes and topic-specific roles do not change group authority.
	if strings.TrimSpace(req.Role) == "" || strings.TrimSpace(req.TopicID) != "" {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK"})
		return
	}
	groupID, userID := strings.TrimSpace(req.GroupID), strings.TrimSpace(req.ModifiedAccount)
	if groupID == "" || userID == "" || (req.Role != "Admin" && req.Role != "Member") {
		c.JSON(http.StatusBadRequest, callbackResp{ActionStatus: "FAIL", ErrorCode: 400, ErrorInfo: "invalid member role"})
		return
	}
	if err := controller_chat.SyncTencentGroupMemberRoles(groupID, []string{userID}, req.OperatorAccount); err != nil {
		logger.Warnf("sync tencent group member role failed: group=%s user=%s", groupID, userID)
		c.JSON(http.StatusInternalServerError, callbackResp{ActionStatus: "FAIL", ErrorCode: 500, ErrorInfo: "member role sync failed"})
		return
	}
	logger.Infof("synced tencent group member role: group=%s user=%s", groupID, userID)
	c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK"})
}
