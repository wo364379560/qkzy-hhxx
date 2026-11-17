package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type addGroupMembersReq struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1"`
}

func FilterGroupMembers(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	var req addGroupMembersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.FilterGroupMembers(imUserID, groupID, req.UserIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func AddGroupMembers(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	var req addGroupMembersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.AddGroupMembers(imUserID, groupID, req.UserIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func RemoveGroupMember(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	userID := c.Param("user_id")
	if err := controller_chat.RemoveGroupMember(imUserID, groupID, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func MuteGroupMember(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	userID := c.Param("user_id")
	if err := controller_chat.SetGroupMemberMute(imUserID, groupID, userID, true); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func UnmuteGroupMember(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	userID := c.Param("user_id")
	if err := controller_chat.SetGroupMemberMute(imUserID, groupID, userID, false); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
