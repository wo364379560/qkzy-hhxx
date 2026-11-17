package handler_admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func ListOwnershipOperations(c *gin.Context) {
	if c.GetString("admin_role") != "super_admin" {
		response.Forbidden(c, "仅超级管理员可查看归属处理记录")
		return
	}
	page, size := parsePagination(c)
	q := datasource.Datasource.Mysql.GetClient().WithContext(c.Request.Context()).Model(&model.FriendOwnerOperation{})
	if state := c.Query("state"); state != "" {
		q = q.Where("state = ?", state)
	}
	if c.Query("review") == "true" {
		q = q.Where("needs_review = ?", true)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.InternalError(c)
		return
	}
	var records []model.FriendOwnerOperation
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&records).Error; err != nil {
		response.InternalError(c)
		return
	}
	rows := make([]gin.H, 0, len(records))
	for _, r := range records {
		rows = append(rows, gin.H{"id": r.ID, "operation_id": r.OperationID, "source": r.Source, "caller": r.Caller, "target": r.Target, "state": r.State, "reason_code": r.ReasonCode, "attempts": r.Attempts, "manual_retries": r.ManualRetries, "last_retried_by": r.LastRetriedBy, "last_retried_at": r.LastRetriedAt, "retry_history": r.RetryHistory, "available_at": r.AvailableAt, "needs_review": r.NeedsReview, "effects_pending": r.EffectsPending, "before": r.BeforeJSON, "after": r.AfterJSON, "created_at": r.CreatedAt})
	}
	response.OK(c, gin.H{"list": rows, "total": total})
}

func RetryOwnershipOperation(c *gin.Context) {
	if c.GetString("admin_role") != "super_admin" {
		response.Forbidden(c, "仅超级管理员可重试归属同步")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.InvalidParams(c, "记录编号无效")
		return
	}
	ctx, requestID := repository.OwnershipRequestTrace(c.Request.Context())
	c.Header("X-Request-ID", requestID)
	err = repository.NewOwnershipRepository().RetryAsAdmin(ctx, id, c.GetUint("admin_id"))
	if err != nil {
		response.BadRequest(c, "当前记录不可重试，可能正在处理或权限已变化")
		return
	}
	response.OK(c, gin.H{"queued": true})
}
