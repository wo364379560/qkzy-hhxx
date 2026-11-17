package handler_chat

import (
	"errors"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type getUserBriefsReq struct {
	IMUserIDs []string `json:"im_user_ids" binding:"required,min=1,max=50"`
}

type getUserTagsReq struct {
	IMUserIDs []string `json:"im_user_ids" binding:"required,min=1,max=200"`
}

type getUserLastOnlineReq struct {
	IMUserIDs []string `json:"im_user_ids" binding:"required,min=1,max=100"`
}

// SearchUsers 在同归属下搜索用户。
// GET /api/v1/chat/users/search?q=<keyword>
func SearchUsers(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	keyword := c.Query("q")
	if keyword == "" {
		response.InvalidParams(c, "q is required")
		return
	}

	results, err := controller_chat.SearchUsers(c.Request.Context(), imUserID, keyword)
	if err != nil {
		var business controller_chat.UserBindOwnerErr
		if errors.As(err, &business) {
			response.BadRequest(c, business.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"users": results})
}

// SearchUsersExact 在同归属下按手机号、万安号或 IM UserID 精确搜索用户。
// GET /api/v1/chat/users/search-exact?q=<keyword>
func SearchUsersExact(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	keyword := c.Query("q")
	if keyword == "" {
		response.InvalidParams(c, "q is required")
		return
	}

	results, err := controller_chat.SearchUsersExact(c.Request.Context(), imUserID, keyword)
	if err != nil {
		var business controller_chat.UserBindOwnerErr
		if errors.As(err, &business) {
			response.BadRequest(c, business.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"users": results})
}

// GetUserBriefs 按 IMUserID 批量查询用户资料（手机号、弥络蓝序号、昵称）。
// POST /api/v1/chat/users/briefs
func GetUserBriefs(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req getUserBriefsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	results, err := controller_chat.GetUserBriefs(c.Request.Context(), imUserID, req.IMUserIDs)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"users": results})
}

// BindFriendOwnerFromMe 把当前登录用户的归属绑定给未归属好友。
// POST /api/v1/chat/users/:im_user_id/bind-owner-from-me
func BindFriendOwnerFromMe(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	targetIMUserID := c.Param("im_user_id")
	if targetIMUserID == "" {
		response.InvalidParams(c, "im_user_id is required")
		return
	}

	var req struct {
		RequestID string `json:"request_id" binding:"omitempty,max=128"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.InvalidParams(c, "请求标识无效")
			return
		}
	}
	ctx, requestID := repository.OwnershipRequestTrace(c.Request.Context())
	c.Header("X-Request-ID", requestID)
	result, err := controller_chat.BindFriendOwnerFromMe(ctx, imUserID, targetIMUserID, req.RequestID)
	if err != nil {
		var businessErr controller_chat.UserBindOwnerErr
		if errors.As(err, &businessErr) {
			response.BadRequest(c, businessErr.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, result)
}

// GetUserTags 在同归属下按 IMUserID 批量查询用户标签。
// POST /api/v1/chat/users/tags
func GetUserTags(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req getUserTagsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	results, err := controller_chat.GetUserTags(c.Request.Context(), imUserID, req.IMUserIDs)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"user_tags": results})
}

// GetUserLastOnline 在同归属下按 IMUserID 批量查询用户最后在线时间。
// POST /api/v1/chat/users/last-online
func GetUserLastOnline(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req getUserLastOnlineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	results, err := controller_chat.GetUserLastOnline(c.Request.Context(), imUserID, req.IMUserIDs)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"user_last_online": results})
}

// GetUserLoginIP 查询单聊对象最近有效登录 IP。
// GET /api/v1/chat/users/login-ip?im_user_id=<target>
func GetUserLoginIP(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	targetIMUserID := c.Query("im_user_id")
	if targetIMUserID == "" {
		response.InvalidParams(c, "im_user_id is required")
		return
	}

	result, err := controller_chat.GetUserLoginIP(c.Request.Context(), imUserID, targetIMUserID)
	if err != nil {
		if errors.Is(err, controller_chat.ErrUserLoginIPForbidden) {
			response.Forbidden(c, "无权查看该用户登录 IP")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, result)
}
