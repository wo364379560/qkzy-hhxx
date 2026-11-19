package handler_admin

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func ListRegistrationIPs(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	entries, err := repository.ListRegistrationIPs(c.Request.Context(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"items": entries})
}

func ChangeRegistrationIP(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	var req struct {
		IP              string `json:"ip" binding:"required,max=64"`
		Note            string `json:"note" binding:"max=200"`
		CurrentPassword string `json:"current_password" binding:"required,max=128"`
		Remove          bool   `json:"remove"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请填写精确公网 IP、备注及当前管理员密码")
		return
	}
	ip, err := repository.NormalizeRegistrationIP(req.IP)
	if err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	action, summary := "registration_ip.add", "添加注册 IP 放行"
	if req.Remove {
		action, summary = "registration_ip.remove", "删除注册 IP 放行"
	}
	middleware.SetAdminAccessAction(c, action, summary)
	middleware.SetAdminAccessTarget(c, "registration_ip", nil, ip, ip)
	if err := controller_admin.ChangeRegistrationIP(c.Request.Context(), c.GetUint("admin_id"), req.CurrentPassword, ip, req.Note, req.Remove, model.AdminAccessAudit{AdminUsername: c.GetString("admin_username"), AdminRole: c.GetString("admin_role"), IPAddress: clientip.FromContext(c)}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		middleware.SetAdminAccessFailureReason(c, "注册 IP 放行修改失败")
		// Don't echo storage errors or credentials to the browser/audit payload.
		response.BadRequest(c, "修改失败，请确认当前管理员密码、备注长度及服务状态")
		return
	}
	middleware.MarkAdminAccessAuditCommitted(c)
	changes, _ := json.Marshal(gin.H{"ip": ip, "removed": req.Remove, "note": req.Note})
	middleware.SetAdminAccessChangeSet(c, string(changes))
	response.OK(c, nil)
}
