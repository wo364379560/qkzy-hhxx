package handler_admin

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func setGroupAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

// ListGroups 获取群组列表。
func ListGroups(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	actor, err := controller_admin.BuildActorContext(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}
	page, pageSize := parsePagination(c)

	var ownerID uint
	if controller_admin.IsAssistantRole(role) {
		ownerID = adminID
	} else if oid := c.Query("ownerID"); oid != "" {
		if id, err := strconv.ParseUint(oid, 10, 64); err == nil {
			if controller_admin.IsSuperAdminRole(role) || controller_admin.CanManageOwnerID(actor, uint(id)) {
				ownerID = uint(id)
			}
		}
	}

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	result, err := controller_admin.ListGroups(controller_admin.ListGroupsReq{
		Page:          page,
		PageSize:      pageSize,
		ScopeOwnerIDs: scopeOwnerIDs,
		OwnerID:       ownerID,
		Keyword:       c.Query("keyword"),
		Status:        c.Query("status"),
		GroupType:     c.Query("groupType"),
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type createGroupReq struct {
	Name          string `json:"name"          binding:"required,max=64"`
	GroupType     string `json:"groupType"     binding:"omitempty,oneof=Work Public Meeting AVChatRoom Community"`
	MemberIDs     []uint `json:"memberIDs"`
	OwnerMemberID uint   `json:"ownerMemberID"` // 指定群主（本地用户 ID），0 = 不指定
	OwnerID       uint   `json:"ownerID"`       // 管理员建群时指定归属人，0 = 使用创建者自身
}

// CreateGroup 创建群组。
func CreateGroup(c *gin.Context) {
	var req createGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	// 确定群组归属：管理员可指定，助理固定为自身
	ownerID := adminID
	if isAdmin && req.OwnerID > 0 {
		ownerID = req.OwnerID
	}

	result, err := controller_admin.CreateGroup(controller_admin.CreateGroupReq{
		Name:          req.Name,
		GroupType:     req.GroupType,
		OwnerID:       ownerID,
		CreatedBy:     adminID,
		MemberIDs:     req.MemberIDs,
		OwnerMemberID: req.OwnerMemberID,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", nil, "", req.Name)
	middleware.SetAdminAccessAction(c, "group.create", "创建群组")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("创建群组；name=%s；owner_id=%d", req.Name, ownerID))
	response.OK(c, result)
}

// BatchCreateGroups 批量创建群组（stub，待后续实现）。
func BatchCreateGroups(c *gin.Context) {
	response.BadRequest(c, "批量创建群组暂未开放")
}

type groupMembersReq struct {
	MemberIDs []uint `json:"memberIDs" binding:"required,min=1"`
}

type batchGroupIDsReq struct {
	GroupIDs []uint `json:"groupIDs" binding:"required,min=1,max=100"`
}

type saveGroupNoticeReq struct {
	Content string `json:"content"`
}

// AddGroupMembers 添加群成员。
func AddGroupMembers(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req groupMembersReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err = controller_admin.AddGroupMembers(adminID, isAdmin, groupID, req.MemberIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", &groupID, "", "")
	middleware.SetAdminAccessAction(c, "group.add_member", "添加群成员")
	count, summary := summarizeUintIDs(req.MemberIDs)
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("添加群成员；group_id=%d；count=%d；user_ids=%s", groupID, count, summary),
	)
	setGroupAuditChangeSet(c, map[string]any{
		"member_count_delta": map[string]any{
			"before": 0,
			"after":  count,
		},
	})
	response.OK(c, nil)
}

// RemoveGroupMembers 移除群成员。
func RemoveGroupMembers(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req groupMembersReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err = controller_admin.RemoveGroupMembers(adminID, isAdmin, groupID, req.MemberIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", &groupID, "", "")
	middleware.SetAdminAccessAction(c, "group.remove_member", "移除群成员")
	count, summary := summarizeUintIDs(req.MemberIDs)
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("移除群成员；group_id=%d；count=%d；user_ids=%s", groupID, count, summary),
	)
	setGroupAuditChangeSet(c, map[string]any{
		"member_count_delta": map[string]any{
			"before": 0,
			"after":  -count,
		},
	})
	response.OK(c, nil)
}

// RepairGroup 将孤立群组补同步到腾讯云 IM。
func RepairGroup(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err = controller_admin.RepairGroup(adminID, isAdmin, groupID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// MuteGroup 开启/关闭全员禁言。
func MuteGroup(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	type req struct {
		Mute bool `json:"mute"`
	}
	var r req
	if err = c.ShouldBindJSON(&r); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err = controller_admin.MuteGroup(adminID, isAdmin, groupID, r.Mute); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", &groupID, "", "")
	if r.Mute {
		middleware.SetAdminAccessAction(c, "group.mute", "设置群静默")
	} else {
		middleware.SetAdminAccessAction(c, "group.unmute", "取消群静默")
	}
	setGroupAuditChangeSet(c, map[string]any{
		"mute_all_members": map[string]any{
			"before": !r.Mute,
			"after":  r.Mute,
		},
	})
	response.OK(c, nil)
}

// BatchMuteGroups 批量开启全员禁言。
func BatchMuteGroups(c *gin.Context) {
	var req batchGroupIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.BatchMuteGroups(adminID, isAdmin, req.GroupIDs, true)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", nil, "", "")
	middleware.SetAdminAccessAction(c, "group.batch_mute", "批量全员禁言")
	setGroupAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  len(req.GroupIDs),
		},
	})
	response.OK(c, result)
}

// BatchUnmuteGroups 批量取消全员禁言。
func BatchUnmuteGroups(c *gin.Context) {
	var req batchGroupIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.BatchMuteGroups(adminID, isAdmin, req.GroupIDs, false)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", nil, "", "")
	middleware.SetAdminAccessAction(c, "group.batch_unmute", "批量取消全员禁言")
	setGroupAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  len(req.GroupIDs),
		},
	})
	response.OK(c, result)
}

// GetGroupMembers 获取群成员列表。
func GetGroupMembers(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.GetGroupMembers(adminID, isAdmin, groupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetGroup 获取群组详情。
func GetGroup(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")

	result, err := controller_admin.GetGroup(groupID, adminID, role)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetGroupNotice 获取群公告扩展内容。
func GetGroupNotice(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	result, err := controller_admin.GetAdminGroupNotice(groupID, adminID)
	if err != nil {
		if role == "" {
			response.InternalError(c)
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// SaveGroupNotice 保存群公告扩展内容。
func SaveGroupNotice(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req saveGroupNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	result, err := controller_admin.SaveAdminGroupNotice(groupID, adminID, req.Content)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// DeleteGroup 解散群组。
func DeleteGroup(c *gin.Context) {
	groupID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err = controller_admin.DismissGroup(adminID, isAdmin, groupID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "group", &groupID, "", "")
	middleware.SetAdminAccessAction(c, "group.dismiss", "解散群组")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("解散群组；group_id=%d", groupID))
	setGroupAuditChangeSet(c, map[string]any{
		"status": map[string]any{
			"before": "active",
			"after":  "dismissed",
		},
	})
	response.OK(c, nil)
}

// SyncGroupsPreview 从腾讯 IM 拉取群组数据预览。
func SyncGroupsPreview(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.SyncGroupsPreview(adminID, isAdmin)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

type syncConfirmReq struct {
	Selections []controller_admin.SyncSelection `json:"selections" binding:"required,min=1"`
}

// SyncGroupsConfirm 确认同步选中的群组。
func SyncGroupsConfirm(c *gin.Context) {
	var req syncConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.SyncGroupsConfirm(adminID, isAdmin, req.Selections)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
