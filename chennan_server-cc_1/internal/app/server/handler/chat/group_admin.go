package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type addGroupAdminReq struct {
	UserID string `json:"user_id" binding:"required"`
}

func GetGroupAdmins(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	result, err := controller_chat.ListGroupAdmins(imUserID, groupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func AddGroupAdmin(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	var req addGroupAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.AddGroupAdmin(imUserID, groupID, req.UserID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func RemoveGroupAdmin(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	userID := c.Param("user_id")
	if err := controller_chat.RemoveGroupAdmin(imUserID, groupID, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
