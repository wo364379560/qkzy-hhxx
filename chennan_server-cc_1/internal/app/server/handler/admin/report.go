package handler_admin

import (
	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// ListReports 获取举报列表。
func ListReports(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	page, pageSize := parsePagination(c)
	actor, err := controller_admin.BuildActorContext(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}

	var ownerID uint
	if controller_admin.IsAssistantRole(role) {
		ownerID = adminID
	}

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	result, err := controller_admin.ListReports(controller_admin.ListReportsReq{
		Page:          page,
		PageSize:      pageSize,
		ScopeOwnerIDs: scopeOwnerIDs,
		OwnerID:       ownerID,
		Status:        c.Query("status"),
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ReviewReport 标记举报为已处理。
func ReviewReport(c *gin.Context) {
	reportID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	if err = controller_admin.ReviewReport(adminID, isAdmin, reportID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "report", &reportID, "", "举报单")
	middleware.SetAdminAccessAction(c, "report.review", "处理举报")
	middleware.SetAdminAccessChangeSet(c, `{"status":{"before":"pending","after":"reviewed"}}`)
	response.OK(c, nil)
}

// DismissReport 忽略举报。
func DismissReport(c *gin.Context) {
	reportID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	if err = controller_admin.DismissReport(adminID, isAdmin, reportID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "report", &reportID, "", "举报单")
	middleware.SetAdminAccessAction(c, "report.dismiss", "忽略举报")
	middleware.SetAdminAccessChangeSet(c, `{"status":{"before":"pending","after":"dismissed"}}`)
	response.OK(c, nil)
}
