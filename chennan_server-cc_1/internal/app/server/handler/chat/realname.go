package handler_chat

import (
	"errors"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/pkg/cloudauth"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// GetAntiFraudNotice 获取防诈公告（需登录）。
func GetAntiFraudNotice(c *gin.Context) {
	result, err := controller_chat.GetAntiFraudNotice()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type verifyRealnameReq struct {
	RealName string `json:"real_name" binding:"required"`
	IDNumber string `json:"id_number" binding:"required"`
}

// GetRealnameStatus 获取当前用户实名状态。
func GetRealnameStatus(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	result, err := controller_chat.GetRealnameStatus(imUserID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type peerRealnameReq struct {
	IMUserIDs []string `json:"im_user_ids"`
}

// GetPeerRealnameStatus 批量查询对方是否已实名。
func GetPeerRealnameStatus(c *gin.Context) {
	if c.GetString("im_user_id") == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req peerRealnameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	items, err := controller_chat.GetPeerRealnameStatus(req.IMUserIDs)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"items": items})
}

// VerifyRealname 提交实名认证二要素核验。
func VerifyRealname(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req verifyRealnameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	err := controller_chat.VerifyRealname(imUserID, controller_chat.VerifyRealnameReq{
		RealName: req.RealName,
		IDNumber: req.IDNumber,
	})
	if err != nil {
		if errors.Is(err, cloudauth.ErrNotConfigured) {
			response.BadRequest(c, err.Error())
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
