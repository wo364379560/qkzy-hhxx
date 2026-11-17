package handler_admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type governanceReasonReq struct {
	Reason string `json:"reason"`
}

type blockIPReq struct {
	IPAddress string `json:"ip_address" binding:"required"`
	Reason    string `json:"reason"`
}

func requireSuperAdmin(c *gin.Context) bool {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		middleware.SetAdminAccessFailureReason(c, "仅超级管理员可访问运营治理中心")
		response.Forbidden(c, "仅超级管理员可访问运营治理中心")
		return false
	}
	return true
}

// GetGovernanceOverview returns operation governance overview.
func GetGovernanceOverview(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	result, err := controller_admin.GetGovernanceOverview()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// GetSecurityRiskSummary returns server-security focused risk counters.
func GetSecurityRiskSummary(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	result, err := controller_admin.GetSecurityRiskSummary()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// GetCleanupStats returns cleanup risk control counters.
func GetCleanupStats(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	result, err := controller_admin.GetCleanupStats()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ListCleanupCandidates returns cleanup candidates.
func ListCleanupCandidates(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	page, pageSize := parsePagination(c)
	result, err := controller_admin.ListCleanupCandidates(
		page,
		pageSize,
		c.Query("keyword"),
		c.Query("currentLoginIP"),
	)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// AddCleanupWhitelist adds one candidate to cleanup whitelist.
func AddCleanupWhitelist(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	userID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.AddCleanupWhitelist(c.GetUint("admin_id"), userID, req.Reason); err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "清理白名单")
	middleware.SetAdminAccessAction(c, "cleanup.whitelist", "加入清理白名单")
	middleware.SetAdminAccessActionDetail(c, "加入清理白名单，自动清理将跳过该用户")
	response.OK(c, nil)
}

// BanCleanupCandidate bans one cleanup candidate.
func BanCleanupCandidate(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	userID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.BanCleanupUser(c.Request.Context(), c.GetUint("admin_id"), userID, req.Reason); err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "清理风控封禁")
	middleware.SetAdminAccessAction(c, "cleanup.manual_ban", "清理风控手动封禁")
	middleware.SetAdminAccessActionDetail(c, "可恢复封禁：本地 banned，腾讯 IM 封禁并踢下线")
	response.OK(c, nil)
}

// RestoreCleanupUser restores a cleanup-banned user.
func RestoreCleanupUser(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	userID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.RestoreCleanupUser(c.Request.Context(), c.GetUint("admin_id"), userID, req.Reason); err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "清理风控恢复")
	middleware.SetAdminAccessAction(c, "cleanup.restore", "清理风控恢复用户")
	middleware.SetAdminAccessActionDetail(c, "恢复本地账号为 active，并解除腾讯 IM 封禁")
	response.OK(c, nil)
}

// ListLoginRisks returns login risk rows.
func ListLoginRisks(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	page, pageSize := parsePagination(c)
	result, err := controller_admin.ListLoginRisks(controller_admin.ListLoginRisksReq{
		Page:     page,
		PageSize: pageSize,
		Scope:    c.Query("scope"),
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Source:   c.Query("source"),
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ResolveLoginRisk resolves one login risk.
func ResolveLoginRisk(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	riskID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.ResolveLoginRisk(c.GetUint("admin_id"), riskID, req.Reason); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "login_risk", &riskID, "", "登录风控")
	middleware.SetAdminAccessAction(c, "login_risk.resolve", "解除登录风控")
	response.OK(c, nil)
}

// ListIPAccessBlocks returns IP block rows.
func ListIPAccessBlocks(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	page, pageSize := parsePagination(c)
	result, err := controller_admin.ListIPAccessBlocks(page, pageSize, c.Query("status"), c.Query("ipAddress"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// BlockIP creates one application-layer IP block.
func BlockIP(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req blockIPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_admin.BlockIP(c.GetUint("admin_id"), req.IPAddress, req.Reason); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "ip_access_block", nil, req.IPAddress, req.IPAddress)
	middleware.SetAdminAccessAction(c, "security.ip_block", "封禁 IP")
	middleware.SetAdminAccessActionDetail(c, "应用层封禁 IP，不修改防火墙")
	response.OK(c, nil)
}

// UnblockIP removes one application-layer IP block.
func UnblockIP(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	blockID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.UnblockIP(c.GetUint("admin_id"), blockID, req.Reason); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "ip_access_block", &blockID, "", "IP 封禁")
	middleware.SetAdminAccessAction(c, "security.ip_unblock", "解封 IP")
	response.OK(c, nil)
}

// ListQuietHoursGroupRisks returns quiet-hours group risk rows.
func ListQuietHoursGroupRisks(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	page, pageSize := parsePagination(c)
	var ownerID uint
	if raw := c.Query("ownerID"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.InvalidParams(c, "ownerID 无效")
			return
		}
		ownerID = uint(id)
	}
	result, err := controller_admin.ListQuietHoursGroupRisks(page, pageSize, c.Query("keyword"), c.Query("status"), ownerID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ResolveQuietHoursGroupRisk clears one quiet-hours group risk mark.
func ResolveQuietHoursGroupRisk(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	riskID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.ResolveQuietHoursGroupRisk(c.GetUint("admin_id"), riskID, req.Reason); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "quiet_hours_group_risk", &riskID, "", "宵禁违规群")
	middleware.SetAdminAccessAction(c, "security.quiet_group_resolve", "解除违规群风险标记")
	response.OK(c, nil)
}

// SyncQuietHoursGroupRisk refreshes local group mirror for one risk row.
func SyncQuietHoursGroupRisk(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	riskID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := controller_admin.SyncQuietHoursGroupRisk(c.Request.Context(), riskID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "quiet_hours_group_risk", &riskID, "", "宵禁违规群")
	middleware.SetAdminAccessAction(c, "security.quiet_group_sync", "补同步违规群资料")
	response.OK(c, nil)
}

// DismissQuietHoursGroupRisk manually dismisses one risky group.
func DismissQuietHoursGroupRisk(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	riskID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req governanceReasonReq
	_ = c.ShouldBindJSON(&req)
	if err := controller_admin.DismissQuietHoursGroupRisk(c.GetUint("admin_id"), riskID, req.Reason); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "quiet_hours_group_risk", &riskID, "", "宵禁违规群")
	middleware.SetAdminAccessAction(c, "security.quiet_group_dismiss", "手动解散违规群")
	middleware.SetAdminAccessActionDetail(c, "超级管理员二次确认后手动解散单个违规群")
	response.OK(c, nil)
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		response.InvalidParams(c, name+" 无效")
		return 0, false
	}
	return uint(id), true
}
