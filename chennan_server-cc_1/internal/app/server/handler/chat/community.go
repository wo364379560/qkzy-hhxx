package handler_chat

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type createCommunityPostReq struct {
	PostType          string `json:"post_type" binding:"required"`
	Title             string `json:"title"`
	Content           string `json:"content" binding:"required"`
	LinkedHabitID     uint   `json:"linked_habit_id"`
	LinkedChallengeID uint   `json:"linked_challenge_id"`
}

type createCommunityCommentReq struct {
	Content string `json:"content" binding:"required"`
}

// ListCommunityPosts 获取社区帖子流。
func ListCommunityPosts(c *gin.Context) {
	feed := c.DefaultQuery("feed", "recommended")
	result, err := controller_chat.ListCommunityPosts(c.Request.Context(), c.GetString("im_user_id"), feed)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// CreateCommunityPost 创建社区帖子。
func CreateCommunityPost(c *gin.Context) {
	var req createCommunityPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.CreateCommunityPost(c.Request.Context(), c.GetString("im_user_id"), controller_chat.CreateCommunityPostReq{
		PostType:          req.PostType,
		Title:             req.Title,
		Content:           req.Content,
		LinkedHabitID:     req.LinkedHabitID,
		LinkedChallengeID: req.LinkedChallengeID,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetCommunityPostDetail 获取帖子详情。
func GetCommunityPostDetail(c *gin.Context) {
	rawID := c.Param("id")
	postID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || postID == 0 {
		response.InvalidParams(c, "invalid post id")
		return
	}

	result, err := controller_chat.GetCommunityPostDetail(c.Request.Context(), c.GetString("im_user_id"), uint(postID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// CreateCommunityComment 创建帖子评论。
func CreateCommunityComment(c *gin.Context) {
	rawID := c.Param("id")
	postID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || postID == 0 {
		response.InvalidParams(c, "invalid post id")
		return
	}

	var req createCommunityCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.CreateCommunityComment(c.Request.Context(), c.GetString("im_user_id"), uint(postID), req.Content)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// ToggleCommunityPostLike 切换帖子点赞。
func ToggleCommunityPostLike(c *gin.Context) {
	rawID := c.Param("id")
	postID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || postID == 0 {
		response.InvalidParams(c, "invalid post id")
		return
	}

	result, err := controller_chat.ToggleCommunityPostLike(c.Request.Context(), c.GetString("im_user_id"), uint(postID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// DeleteCommunityPost 删除社区帖子。
func DeleteCommunityPost(c *gin.Context) {
	rawID := c.Param("id")
	postID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || postID == 0 {
		response.InvalidParams(c, "invalid post id")
		return
	}

	if err := controller_chat.DeleteCommunityPost(c.Request.Context(), c.GetString("im_user_id"), uint(postID)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, nil)
}
