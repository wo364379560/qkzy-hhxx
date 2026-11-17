package handler_chat

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/app/server/forwardqueue"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func forwardQueue() forwardqueue.Queue {
	return forwardqueue.Queue{DB: datasource.Datasource.Mysql.GetClient()}
}
func forwardError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.NotFound(c, "转发任务不存在")
	case errors.Is(err, forwardqueue.ErrInvalid), errors.Is(err, forwardqueue.ErrConflict), errors.Is(err, forwardqueue.ErrChunk), errors.Is(err, forwardqueue.ErrNotReady):
		response.BadRequest(c, err.Error())
	default:
		response.InternalError(c)
	}
}
func forwardBody(c *gin.Context, body any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	if err := c.ShouldBindJSON(body); err != nil {
		response.BadRequest(c, "转发分段内容无效或过大")
		return false
	}
	return true
}
func forwardResult(c *gin.Context, result *forwardqueue.Snapshot, err error) {
	if err != nil {
		forwardError(c, err)
		return
	}
	response.OK(c, result)
}
func CreateForwardJob(c *gin.Context) {
	if !tencentim.NewClient().IsConfigured() {
		response.BadRequest(c, "腾讯 IM 尚未配置")
		return
	}
	var req forwardqueue.Request
	if !forwardBody(c, &req) {
		return
	}
	result, err := forwardQueue().Create(c.Request.Context(), c.GetString("im_user_id"), req, true)
	forwardResult(c, result, err)
}
func InitForwardJob(c *gin.Context) {
	if !tencentim.NewClient().IsConfigured() {
		response.BadRequest(c, "腾讯 IM 尚未配置")
		return
	}
	var req forwardqueue.Manifest
	if !forwardBody(c, &req) {
		return
	}
	result, err := forwardQueue().Init(c.Request.Context(), c.GetString("im_user_id"), req)
	forwardResult(c, result, err)
}
func UploadForwardTargets(c *gin.Context) {
	var req forwardqueue.TargetChunk
	if !forwardBody(c, &req) {
		return
	}
	result, err := forwardQueue().UploadTargets(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id"), req)
	forwardResult(c, result, err)
}
func UploadForwardMessages(c *gin.Context) {
	var req forwardqueue.MessageChunk
	if !forwardBody(c, &req) {
		return
	}
	result, err := forwardQueue().UploadMessages(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id"), req)
	forwardResult(c, result, err)
}
func CommitForwardJob(c *gin.Context) {
	result, err := forwardQueue().Commit(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id"))
	forwardResult(c, result, err)
}
func ForwardJobStatus(c *gin.Context) {
	result, err := forwardQueue().Status(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id"))
	forwardResult(c, result, err)
}
func GetForwardJob(c *gin.Context) {
	cursor, err := strconv.ParseUint(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil {
		response.BadRequest(c, "进度分页参数无效")
		return
	}
	result, err := forwardQueue().GetPage(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id"), cursor)
	forwardResult(c, result, err)
}
func CancelForwardJob(c *gin.Context) {
	if err := forwardQueue().Cancel(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id")); err != nil {
		forwardError(c, err)
		return
	}
	ForwardJobStatus(c)
}
func RetryForwardJob(c *gin.Context) {
	var req struct {
		RetryID string `json:"retry_id"`
	}
	if !forwardBody(c, &req) {
		return
	}
	result, err := forwardQueue().RetryFailed(c.Request.Context(), c.GetString("im_user_id"), c.Param("request_id"), req.RetryID)
	forwardResult(c, result, err)
}
func ForwardSettings(c *gin.Context) { response.OK(c, forwardqueue.CurrentSettings()) }

// The server enumerates every owned task; another window cannot hide a job by
// overwriting a single local recovery key.
func ListForwardJobs(c *gin.Context) {
	cursor := c.Query("cursor")
	if len(cursor) > 64 {
		response.BadRequest(c, "分页参数无效")
		return
	}
	var jobs []struct {
		ID        string
		RequestID string
	}
	db := datasource.Datasource.Mysql.GetClient().WithContext(c.Request.Context())
	query := db.Table("forward_jobs").Select("id, request_id").Where("sender = ?", c.GetString("im_user_id"))
	if cursor != "" {
		query = query.Where("id > ?", cursor)
	}
	if err := query.Order("id ASC").Limit(21).Find(&jobs).Error; err != nil {
		response.InternalError(c)
		return
	}
	next := ""
	if len(jobs) > 20 {
		next = jobs[19].ID
		jobs = jobs[:20]
	}
	results := []*forwardqueue.Snapshot{}
	for _, job := range jobs {
		snapshot, err := forwardQueue().Status(c.Request.Context(), c.GetString("im_user_id"), job.RequestID)
		if err != nil {
			forwardError(c, err)
			return
		}
		results = append(results, snapshot)
	}
	response.OK(c, gin.H{"items": results, "next_cursor": next})
}
