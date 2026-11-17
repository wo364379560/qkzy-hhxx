package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type revokeGroupMessageReq struct {
	GroupID string `json:"group_id" binding:"required"`
	MsgSeq  uint64 `json:"msg_seq" binding:"required"`
}

// RevokeGroupMessage POST /messages/group/revoke
func RevokeGroupMessage(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req revokeGroupMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_chat.RevokeGroupMessage(imUserID, req.GroupID, req.MsgSeq); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
