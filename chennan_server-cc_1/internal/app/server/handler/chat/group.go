package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type saveGroupNoticeReq struct {
	Pinned  *bool  `json:"pinned"`
	GroupID string `json:"group_id" binding:"required"`
	Content string `json:"content"`
}

type saveGroupSlowModeReq struct {
	Enabled bool `json:"enabled"`
	Seconds int  `json:"seconds"`
}

// SearchGroup 在同归属下搜索群组。
// 支持两种模式：
//   - 精确匹配：GET /api/v1/chat/groups/search?group_id=<groupID>
//   - 模糊搜索：GET /api/v1/chat/groups/search?q=<keyword>
func SearchGroup(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Query("group_id")
	keyword := c.Query("q")

	// 优先精确匹配（保持兼容）
	if groupID != "" {
		result, err := controller_chat.SearchGroup(c.Request.Context(), imUserID, groupID)
		if err != nil {
			response.InternalError(c)
			return
		}
		response.OK(c, gin.H{"group": result})
		return
	}

	// 模糊搜索
	if keyword != "" {
		results, err := controller_chat.SearchGroups(c.Request.Context(), imUserID, keyword)
		if err != nil {
			response.InternalError(c)
			return
		}
		response.OK(c, gin.H{"groups": results})
		return
	}

	response.InvalidParams(c, "group_id or q is required")
}

func GetGroupNotice(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Query("group_id")
	result, err := controller_chat.GetGroupNotice(imUserID, groupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func SaveGroupNotice(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req saveGroupNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.SaveGroupNotice(imUserID, req.GroupID, req.Content, req.Pinned)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func GetGroupSlowMode(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	result, err := controller_chat.GetGroupSlowMode(imUserID, groupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func SaveGroupSlowMode(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	var req saveGroupSlowModeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.SaveGroupSlowMode(imUserID, groupID, req.Enabled, req.Seconds)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func PinGroupNotice(c *gin.Context) {
	var req struct {
		GroupID string `json:"group_id" binding:"required"`
		Pinned  *bool  `json:"pinned" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.PinGroupNotice(c.GetString("im_user_id"), req.GroupID, *req.Pinned)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
