package handler_chat

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type createHabitReq struct {
	Name         string `json:"name" binding:"required,max=32"`
	Icon         string `json:"icon"`
	GoalType     string `json:"goal_type" binding:"required"`
	GoalTarget   string `json:"goal_target"`
	ReminderTime string `json:"reminder_time"`
}

type checkinHabitReq struct {
	Value string `json:"value"`
}

// GetHabitDashboard 获取当前用户的打卡首页数据。
func GetHabitDashboard(c *gin.Context) {
	result, err := controller_chat.GetHabitDashboard(c.Request.Context(), c.GetString("im_user_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// CreateHabit 创建当前用户习惯。
func CreateHabit(c *gin.Context) {
	var req createHabitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.CreateHabit(c.Request.Context(), c.GetString("im_user_id"), controller_chat.CreateHabitReq{
		Name:         req.Name,
		Icon:         req.Icon,
		GoalType:     req.GoalType,
		GoalTarget:   req.GoalTarget,
		ReminderTime: req.ReminderTime,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// CheckinHabit 完成或更新今日打卡。
func CheckinHabit(c *gin.Context) {
	rawHabitID := c.Param("id")
	habitID, err := strconv.ParseUint(rawHabitID, 10, 64)
	if err != nil || habitID == 0 {
		response.InvalidParams(c, "invalid habit id")
		return
	}

	var req checkinHabitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.CheckinHabit(c.Request.Context(), c.GetString("im_user_id"), uint(habitID), controller_chat.CheckinHabitReq{
		Value: req.Value,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
