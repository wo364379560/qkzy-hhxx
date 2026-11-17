package handler_chat

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// GetChallengeDetail 获取挑战详情。
func GetChallengeDetail(c *gin.Context) {
	rawID := c.Param("id")
	challengeID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || challengeID == 0 {
		response.InvalidParams(c, "invalid challenge id")
		return
	}

	result, err := controller_chat.GetChallengeDetail(c.Request.Context(), c.GetString("im_user_id"), uint(challengeID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
