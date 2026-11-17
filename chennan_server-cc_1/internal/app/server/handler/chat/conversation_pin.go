package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type pinConversationReq struct {
	ConversationID string `json:"conversationID" binding:"required"`
	PinOrder       int    `json:"pinOrder"`
}

type reorderConversationPinsReq struct {
	ConversationIDs []string `json:"conversationIDs" binding:"required,min=1"`
}

func ListConversationCustomPins(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	uid, ok := imUserID.(string)
	if !ok || uid == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	result, err := controller_chat.ListConversationCustomPins(uid)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func PinConversationCustom(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	uid, ok := imUserID.(string)
	if !ok || uid == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req pinConversationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_chat.PinConversationCustom(uid, req.ConversationID, req.PinOrder); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func UnpinConversationCustom(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	uid, ok := imUserID.(string)
	if !ok || uid == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req pinConversationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_chat.UnpinConversationCustom(uid, req.ConversationID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func ReorderConversationCustomPins(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	uid, ok := imUserID.(string)
	if !ok || uid == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req reorderConversationPinsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_chat.ReorderConversationCustomPins(uid, req.ConversationIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
