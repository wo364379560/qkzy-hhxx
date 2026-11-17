package handler_chat

import (
	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type createChallengeReq struct {
	Name            string   `json:"name" binding:"required,max=64"`
	Description     string   `json:"description"`
	LinkedHabitID   uint     `json:"linked_habit_id" binding:"required"`
	DurationDays    int      `json:"duration_days"`
	MemberIMUserIDs []string `json:"member_im_user_ids"`
}

// CreateChallenge 创建挑战并自动建群。
func CreateChallenge(c *gin.Context) {
	var req createChallengeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.CreateChallenge(c.Request.Context(), c.GetString("im_user_id"), controller_chat.CreateChallengeReq{
		Name:            req.Name,
		Description:     req.Description,
		LinkedHabitID:   req.LinkedHabitID,
		DurationDays:    req.DurationDays,
		MemberIMUserIDs: req.MemberIMUserIDs,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
