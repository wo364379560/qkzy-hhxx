package handler_admin

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// ListUserOperationLogs 获取移动端用户操作日志列表。
func ListUserOperationLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	role := c.GetString("admin_role")

	if !controller_admin.IsSuperAdminRole(role) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	ipAddress := strings.TrimSpace(c.Query("ipAddress"))
	if ipAddress != "" && net.ParseIP(ipAddress) == nil {
		response.BadRequest(c, "请输入合法 IP")
		return
	}

	startTime, err := parseOptionalAdminQueryTime(c.Query("startTime"), false)
	if err != nil {
		response.BadRequest(c, "开始时间格式不正确")
		return
	}
	endTime, err := parseOptionalAdminQueryTime(c.Query("endTime"), true)
	if err != nil {
		response.BadRequest(c, "结束时间格式不正确")
		return
	}

	result, err := controller_admin.ListUserOperationLogs(controller_admin.ListUserOperationLogsReq{
		Page:          page,
		PageSize:      pageSize,
		Keyword:       c.Query("keyword"),
		OperationType: c.Query("operationType"),
		Status:        c.Query("status"),
		Phone:         c.Query("phone"),
		IMUserID:      c.Query("imUserId"),
		OwnerID:       parseOptionalUintQuery(c, "ownerId"),
		IPAddress:     ipAddress,
		StartTime:     startTime,
		EndTime:       endTime,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

func parseOptionalAdminQueryTime(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, time.Local)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			parsed = parsed.Add(24*time.Hour - time.Nanosecond)
		}
		return &parsed, nil
	}
	return nil, errors.New("invalid time format")
}

func parseOptionalUintQuery(c *gin.Context, name string) uint {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}
