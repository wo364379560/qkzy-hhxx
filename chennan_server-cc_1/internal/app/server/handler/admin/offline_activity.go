package handler_admin

import (
	"encoding/json"
	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func setOfflineActivityAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

type offlineActivityReq struct {
	Title       string `json:"title" binding:"required,max=64"`
	Subtitle    string `json:"subtitle" binding:"max=120"`
	Description string `json:"description" binding:"max=1000"`
	City        string `json:"city" binding:"required,max=64"`
	Venue       string `json:"venue" binding:"required,max=255"`
	StartAt     string `json:"startAt" binding:"required"`
	EndAt       string `json:"endAt" binding:"required"`
	Quota       int    `json:"quota"`
	Status      string `json:"status"`
}

// ListOfflineActivities 获取线下活动列表。
func ListOfflineActivities(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	page, pageSize := parsePagination(c)
	status := c.DefaultQuery("status", "")

	result, err := controller_admin.ListOfflineActivities(controller_admin.ListOfflineActivitiesReq{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// CreateOfflineActivity 创建线下活动。
func CreateOfflineActivity(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}

	var req offlineActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_admin.CreateOfflineActivity(controller_admin.CreateOfflineActivityReq{
		Title:       req.Title,
		Subtitle:    req.Subtitle,
		Description: req.Description,
		City:        req.City,
		Venue:       req.Venue,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		Quota:       req.Quota,
		Status:      req.Status,
		CreatedBy:   c.GetUint("admin_id"),
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "offline_activity", &result.ID, "", result.Title)
	middleware.SetAdminAccessAction(c, "offline_activity.create", "创建活动")
	setOfflineActivityAuditChangeSet(c, map[string]any{
		"title":  map[string]any{"before": "", "after": result.Title},
		"city":   map[string]any{"before": "", "after": result.City},
		"venue":  map[string]any{"before": "", "after": result.Venue},
		"status": map[string]any{"before": "", "after": result.Status},
	})
	response.OK(c, result)
}

// UpdateOfflineActivity 更新线下活动。
func UpdateOfflineActivity(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	activityID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req offlineActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_admin.UpdateOfflineActivity(activityID, controller_admin.UpdateOfflineActivityReq{
		Title:       req.Title,
		Subtitle:    req.Subtitle,
		Description: req.Description,
		City:        req.City,
		Venue:       req.Venue,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		Quota:       req.Quota,
		Status:      req.Status,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "offline_activity", &activityID, "", result.Title)
	middleware.SetAdminAccessAction(c, "offline_activity.update", "更新活动")
	setOfflineActivityAuditChangeSet(c, map[string]any{
		"title":  map[string]any{"before": "已更新前值", "after": result.Title},
		"city":   map[string]any{"before": "已更新前值", "after": result.City},
		"venue":  map[string]any{"before": "已更新前值", "after": result.Venue},
		"status": map[string]any{"before": "已更新前值", "after": result.Status},
	})
	response.OK(c, result)
}

// CloseOfflineActivity 下线活动。
func CloseOfflineActivity(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	activityID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	if err := controller_admin.CloseOfflineActivity(activityID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "offline_activity", &activityID, "", "线下活动")
	middleware.SetAdminAccessAction(c, "offline_activity.close", "关闭活动")
	middleware.SetAdminAccessChangeSet(c, `{"status":{"before":"active","after":"closed"}}`)
	response.OK(c, nil)
}

// ListOfflineActivityRegistrations 获取活动报名名单。
func ListOfflineActivityRegistrations(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	activityID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	result, err := controller_admin.ListOfflineActivityRegistrations(activityID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
