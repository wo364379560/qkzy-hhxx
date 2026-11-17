package handler_chat

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// ListOfflineActivities 获取可报名线下活动列表。
func ListOfflineActivities(c *gin.Context) {
	result, err := controller_chat.ListOfflineActivities(c.GetString("im_user_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ToggleOfflineActivityRegistration 切换线下活动报名状态。
func ToggleOfflineActivityRegistration(c *gin.Context) {
	rawID := c.Param("id")
	activityID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || activityID == 0 {
		response.InvalidParams(c, "invalid activity id")
		return
	}

	result, err := controller_chat.ToggleOfflineActivityRegistration(c.GetString("im_user_id"), uint(activityID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
