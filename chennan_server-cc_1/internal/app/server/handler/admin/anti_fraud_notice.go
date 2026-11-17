package handler_admin

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type setAntiFraudNoticeReq struct {
	Enabled         bool   `json:"enabled"`
	Content         string `json:"content"`
	CountdownSec    int    `json:"countdown_sec"`
	CurrentPassword string `json:"current_password" binding:"required"`
}

type setRealnameVerifyReq struct {
	Enabled         bool   `json:"enabled"`
	CurrentPassword string `json:"current_password" binding:"required"`
}

// GetAntiFraudNoticeSettings 获取防诈公告配置。
func GetAntiFraudNoticeSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	result, err := controller_admin.GetAntiFraudNoticeSettings()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// SetAntiFraudNoticeSettings 设置防诈公告配置。
func SetAntiFraudNoticeSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setAntiFraudNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	before, _ := controller_admin.GetAntiFraudNoticeSettings()
	if err := controller_admin.SetAntiFraudNoticeSettings(adminID, req.CurrentPassword, controller_admin.AntiFraudNoticeSettingsResult{
		Enabled:      req.Enabled,
		Content:      req.Content,
		CountdownSec: req.CountdownSec,
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "anti_fraud_notice", "防诈公告")
	middleware.SetAdminAccessAction(c, "config.update_anti_fraud_notice", "修改防诈公告")
	middleware.SetAdminAccessActionDetail(c, "修改防诈公告配置")
	if payload, err := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before.Enabled,
			"after":  req.Enabled,
		},
		"countdown_sec": map[string]any{
			"before": before.CountdownSec,
			"after":  req.CountdownSec,
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}

// GetRealnameVerifySettings 获取全局实名认证开关。
func GetRealnameVerifySettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	enabled, err := controller_admin.GetRealnameVerifyEnabled()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, map[string]bool{"enabled": enabled})
}

// SetRealnameVerifySettings 设置全局实名认证开关。
func SetRealnameVerifySettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setRealnameVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	before, _ := controller_admin.GetRealnameVerifyEnabled()
	if err := controller_admin.SetRealnameVerifyEnabled(adminID, req.CurrentPassword, req.Enabled); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "realname_verify", "实名认证")
	middleware.SetAdminAccessAction(c, "config.update_realname_verify", "修改实名认证开关")
	middleware.SetAdminAccessActionDetail(c, "修改全局实名认证开关")
	if payload, err := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before,
			"after":  req.Enabled,
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}

// ResetUserRealname 重置用户实名认证状态。
func ResetUserRealname(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	if err := controller_admin.ResetUserRealname(c.GetUint("admin_id"), uint(id)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := uint(id)
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "")
	middleware.SetAdminAccessAction(c, "user.reset_realname", "重置用户实名")
	middleware.SetAdminAccessActionDetail(c, "重置用户实名认证状态")
	response.OK(c, nil)
}

type verifyUserRealnameReq struct {
	RealName string `json:"real_name" binding:"required"`
	IDNumber string `json:"id_number" binding:"required"`
}

type setUserRealnameStatusReq struct {
	Verified bool `json:"verified"`
}

// SetUserRealnameStatus 简化实名开关（已实名 / 未实名）。
func SetUserRealnameStatus(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	var req setUserRealnameStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_admin.SetUserRealnameVerified(c.GetUint("admin_id"), uint(id), req.Verified); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := uint(id)
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "")
	if req.Verified {
		middleware.SetAdminAccessAction(c, "user.set_realname", "标记用户已实名")
	} else {
		middleware.SetAdminAccessAction(c, "user.set_realname", "标记用户未实名")
	}
	response.OK(c, nil)
}

// VerifyUserRealname 超管代用户提交阿里云二要素核验。
func VerifyUserRealname(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	var req verifyUserRealnameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_admin.VerifyUserRealname(c.GetUint("admin_id"), uint(id), req.RealName, req.IDNumber); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := uint(id)
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "")
	middleware.SetAdminAccessAction(c, "user.verify_realname", "代核验用户实名")
	middleware.SetAdminAccessActionDetail(c, "调用阿里云二要素核验并写入实名结果")
	response.OK(c, nil)
}
