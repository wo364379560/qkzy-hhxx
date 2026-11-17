package handler_chat

import (
	"errors"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// GetProfile 获取当前用户资料。
func GetProfile(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	uid, ok := imUserID.(string)
	if !ok || uid == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	result, err := controller_chat.GetProfile(c.Request.Context(), uid)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type updateProfileReq struct {
	WananID       *string `json:"wanan_id"`
	Nickname      *string `json:"nickname"`
	AvatarURL     *string `json:"avatar_url"`
	SelfSignature *string `json:"self_signature"`
	Gender        *int8   `json:"gender"`
	Birthday      *int    `json:"birthday"` // YYYYMMDD
	AllowType     *int8   `json:"allow_type"`
}

// UpdateProfile 更新当前用户资料（增量，nil 字段不更新）。
func UpdateProfile(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	uid, ok := imUserID.(string)
	if !ok || uid == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.UpdateProfile(c.Request.Context(), uid, controller_chat.UpdateProfileReq{
		WananID:       req.WananID,
		Nickname:      req.Nickname,
		AvatarURL:     req.AvatarURL,
		SelfSignature: req.SelfSignature,
		Gender:        req.Gender,
		Birthday:      req.Birthday,
		AllowType:     req.AllowType,
	})
	if err != nil {
		var syncError *controller_chat.ProfileSyncError
		if errors.As(err, &syncError) {
			c.JSON(502, response.Response{Code: 502, Message: syncError.Error(), Data: gin.H{"stage": "im_profile", "business_saved": true, "avatar_url": syncError.AvatarURL, "retryable": true}})
			return
		}
		var pe controller_chat.ProfileErr
		if errors.As(err, &pe) {
			response.BadRequest(c, pe.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}
