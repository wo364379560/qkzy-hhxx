package handler_admin

import (
	"github.com/gin-gonic/gin"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/forwardqueue"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func ListForwardJobs(c *gin.Context) {
	actor, err := controller_admin.BuildActorContext(c.GetUint("admin_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	cursor := c.Query("cursor")
	if len(cursor) > 64 {
		response.BadRequest(c, "分页参数无效")
		return
	}
	var jobs []struct {
		ID        string
		RequestID string
		Sender    string
	}
	db := datasource.Datasource.Mysql.GetClient().WithContext(c.Request.Context())
	query := db.Table("forward_jobs AS jobs").Select("jobs.id,jobs.request_id,jobs.sender").Joins("JOIN users ON users.im_user_id = jobs.sender")
	if !controller_admin.IsSuperAdminRole(actor.Role) {
		query = query.Where("users.owner_id IN ?", actor.ScopeAdminIDs)
	}
	if cursor != "" {
		query = query.Where("jobs.id > ?", cursor)
	}
	if query.Order("jobs.id ASC").Limit(21).Find(&jobs).Error != nil {
		response.InternalError(c)
		return
	}
	next := ""
	if len(jobs) > 20 {
		next = jobs[19].ID
		jobs = jobs[:20]
	}
	rows := []gin.H{}
	for _, job := range jobs {
		status, err := (forwardqueue.Queue{DB: db}).Status(c.Request.Context(), job.Sender, job.RequestID)
		if err != nil {
			response.InternalError(c)
			return
		}
		rows = append(rows, gin.H{"sender": job.Sender, "progress": status})
	}
	response.OK(c, gin.H{"items": rows, "next_cursor": next, "settings": forwardqueue.CurrentSettings()})
}
