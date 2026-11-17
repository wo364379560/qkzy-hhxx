package handler_admin

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func setInviteCodeAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

// ListInviteCodes 获取邀请码列表。
// 管理员：查看所有邀请码
// 助理：只能查看自己的邀请码
func ListInviteCodes(c *gin.Context) {
	page, pageSize := parsePagination(c)
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	actor, err := controller_admin.BuildActorContext(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}

	var ownerID uint
	if controller_admin.IsAssistantRole(role) {
		ownerID = adminID // 助理只能查看自己的
	} else if oid := c.Query("ownerID"); oid != "" {
		if id, parseErr := strconv.ParseUint(oid, 10, 64); parseErr == nil {
			if controller_admin.IsSuperAdminRole(role) || controller_admin.CanManageOwnerID(actor, uint(id)) {
				ownerID = uint(id)
			}
		}
	}

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	result, err := controller_admin.ListInviteCodes(controller_admin.ListInviteCodesReq{
		Page:          page,
		PageSize:      pageSize,
		ScopeOwnerIDs: scopeOwnerIDs,
		OwnerID:       ownerID,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type createInviteCodeReq struct {
	MaxUses   int    `json:"maxUses"`
	ExpiredAt string `json:"expiredAt"` // RFC3339 or empty
}

// CreateInviteCode 创建单个邀请码。
func CreateInviteCode(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "助理无权管理邀请码")
		return
	}
	var req createInviteCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")

	var expiredAt *time.Time
	if req.ExpiredAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiredAt)
		if err != nil {
			response.InvalidParams(c, "到期时间格式无效，请使用 RFC3339")
			return
		}
		expiredAt = &t
	}

	result, err := controller_admin.CreateInviteCode(controller_admin.CreateInviteCodeReq{
		OwnerID:   adminID,
		MaxUses:   req.MaxUses,
		ExpiredAt: expiredAt,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	middleware.SetAdminAccessTarget(c, "invite_code", &result.ID, result.Code, result.Code)
	middleware.SetAdminAccessAction(c, "invite_code.create", "创建邀请码")
	setInviteCodeAuditChangeSet(c, map[string]any{
		"code":     map[string]any{"before": "", "after": result.Code},
		"max_uses": map[string]any{"before": 0, "after": result.MaxUses},
		"status":   map[string]any{"before": "", "after": result.Status},
	})
	response.OK(c, result)
}

type batchCreateInviteCodesReq struct {
	Count     int    `json:"count"     binding:"required,min=1,max=200"`
	MaxUses   int    `json:"maxUses"`
	ExpiredAt string `json:"expiredAt"`
	OwnerID   uint   `json:"ownerID"` // 0 = 归属当前管理员
}

// BatchCreateInviteCodes 批量创建邀请码。
func BatchCreateInviteCodes(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "助理无权管理邀请码")
		return
	}
	var req batchCreateInviteCodesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var expiredAt *time.Time
	if req.ExpiredAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiredAt)
		if err != nil {
			response.InvalidParams(c, "到期时间格式无效，请使用 RFC3339")
			return
		}
		expiredAt = &t
	}

	results, err := controller_admin.BatchCreateInviteCodes(adminID, isAdmin, req.OwnerID, req.Count, req.MaxUses, expiredAt)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "invite_code", nil, "", "邀请码批量创建")
	middleware.SetAdminAccessAction(c, "invite_code.batch_create", "批量创建邀请码")
	setInviteCodeAuditChangeSet(c, map[string]any{
		"count":    map[string]any{"before": 0, "after": req.Count},
		"max_uses": map[string]any{"before": 0, "after": req.MaxUses},
	})
	response.OK(c, results)
}

type updateInviteCodeReq struct {
	MaxUses int    `json:"maxUses"`
	Status  string `json:"status"`
	Code    string `json:"code"` // 可选：手动指定码字符串
}

// UpdateInviteCode 更新邀请码设置（maxUses、启用状态，可选码字符串）。
func UpdateInviteCode(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "助理无权管理邀请码")
		return
	}
	codeID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req updateInviteCodeReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.UpdateInviteCode(adminID, isAdmin, controller_admin.UpdateInviteCodeReq{
		CodeID:  codeID,
		MaxUses: req.MaxUses,
		Status:  req.Status,
		Code:    req.Code,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "invite_code", &codeID, req.Code, req.Code)
	middleware.SetAdminAccessAction(c, "invite_code.update", "更新邀请码")
	setInviteCodeAuditChangeSet(c, map[string]any{
		"max_uses": map[string]any{"before": "已更新前值", "after": result.MaxUses},
		"status":   map[string]any{"before": "已更新前值", "after": result.Status},
		"code":     map[string]any{"before": "已更新前值", "after": result.Code},
	})
	response.OK(c, result)
}

// RegenerateInviteCode 重新生成邀请码字符串（保留其他设置）。
func RegenerateInviteCode(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "助理无权管理邀请码")
		return
	}
	codeID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.RegenerateInviteCode(adminID, isAdmin, codeID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "invite_code", &codeID, result.Code, result.Code)
	middleware.SetAdminAccessAction(c, "invite_code.regenerate", "重置邀请码")
	setInviteCodeAuditChangeSet(c, map[string]any{
		"code": map[string]any{"before": "旧邀请码", "after": result.Code},
	})
	response.OK(c, result)
}

// DeleteInviteCode 删除邀请码。
func DeleteInviteCode(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "助理无权管理邀请码")
		return
	}
	codeID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err = controller_admin.DeleteInviteCode(adminID, isAdmin, codeID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "invite_code", &codeID, "", "邀请码")
	middleware.SetAdminAccessAction(c, "invite_code.delete", "删除邀请码")
	setInviteCodeAuditChangeSet(c, map[string]any{
		"status": map[string]any{"before": "active", "after": "deleted"},
	})
	response.OK(c, nil)
}
