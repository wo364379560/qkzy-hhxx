package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type submitReportReq struct {
	TargetIMID  string `json:"target_id"   binding:"required"`
	TargetType  string `json:"target_type" binding:"required,oneof=user group community_post community_comment"`
	TargetName  string `json:"target_name"`
	Category    string `json:"category"    binding:"required,oneof=spam abuse fraud other"`
	Description string `json:"description"`
}

// SubmitReport 接收 App 端举报请求并写入数据库。
func SubmitReport(c *gin.Context) {
	imUserID, _ := c.Get("im_user_id")
	reporterIMID, ok := imUserID.(string)
	if !ok || reporterIMID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req submitReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	userRepo := repository.NewUserRepository()

	// 取举报人信息
	reporter, err := userRepo.FindByIMUserID(reporterIMID)
	if err != nil {
		response.InternalError(c)
		return
	}

	createReq := controller_admin.CreateReportReq{
		ReporterID:   reporter.ID,
		ReporterName: reporter.Nickname,
		TargetIMID:   req.TargetIMID,
		TargetType:   req.TargetType,
		TargetName:   req.TargetName,
		Category:     req.Category,
		Description:  req.Description,
		OwnerID:      reporter.OwnerID, // 举报记录归属于举报人的助理
	}

	// 被举报方是用户时，补充其 DB ID
	if req.TargetType == "user" {
		target, err := userRepo.FindByIMUserID(req.TargetIMID)
		if err == nil {
			createReq.TargetID = target.ID
		}
	}

	if err := controller_admin.CreateReport(createReq); err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, nil)
}
