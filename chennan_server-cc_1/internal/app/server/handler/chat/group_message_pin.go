package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type saveGroupMessagePinReq struct {
	GroupID               string `json:"group_id" binding:"required"`
	MessageID             string `json:"message_id" binding:"required"`
	MessageSeq            int64  `json:"message_seq"`
	MessageSenderIMUserID string `json:"message_sender_im_user_id"`
	MessageSenderName     string `json:"message_sender_name"`
	MessageType           string `json:"message_type"`
	MessageAbstract       string `json:"message_abstract"`
}

type removeGroupMessagePinReq struct {
	GroupID string `json:"group_id" binding:"required"`
}

func GetGroupMessagePin(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Query("group_id")
	result, err := controller_chat.GetGroupMessagePin(imUserID, groupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func SaveGroupMessagePin(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req saveGroupMessagePinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.SaveGroupMessagePin(imUserID, controller_chat.SaveGroupMessagePinReq{
		GroupID:               req.GroupID,
		MessageID:             req.MessageID,
		MessageSeq:            req.MessageSeq,
		MessageSenderIMUserID: req.MessageSenderIMUserID,
		MessageSenderName:     req.MessageSenderName,
		MessageType:           req.MessageType,
		MessageAbstract:       req.MessageAbstract,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func RemoveGroupMessagePin(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req removeGroupMessagePinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_chat.RemoveGroupMessagePin(imUserID, req.GroupID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
