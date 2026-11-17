package handler_admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type updateCommunityPostReq struct {
	PostType string `json:"postType"`
	Title    string `json:"title"`
	Content  string `json:"content" binding:"required"`
}

type updateCommunityUserRestrictionReq struct {
	CommunityPostDisabled    *bool `json:"communityPostDisabled"`
	CommunityCommentDisabled *bool `json:"communityCommentDisabled"`
}

func ListAdminCommunityPosts(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	actor, err := controller_admin.BuildActorContext(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}
	page, pageSize := parsePagination(c)

	var ownerID uint
	if controller_admin.IsAssistantRole(role) {
		ownerID = adminID
	} else if oid := c.Query("ownerID"); oid != "" {
		if id, parseErr := strconv.ParseUint(oid, 10, 64); parseErr == nil {
			if controller_admin.IsSuperAdminRole(role) || controller_admin.CanManageOwnerID(actor, uint(id)) {
				ownerID = uint(id)
			}
		}
	}

	var userID uint
	if uid := c.Query("userID"); uid != "" {
		if id, parseErr := strconv.ParseUint(uid, 10, 64); parseErr == nil {
			userID = uint(id)
		}
	}

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	result, err := controller_admin.ListAdminCommunityPosts(controller_admin.ListAdminCommunityPostsReq{
		Page:          page,
		PageSize:      pageSize,
		ScopeOwnerIDs: scopeOwnerIDs,
		OwnerID:       ownerID,
		Keyword:       c.Query("keyword"),
		PostType:      c.Query("postType"),
		UserID:        userID,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func GetAdminCommunityComments(c *gin.Context) {
	postID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.GetAdminCommunityComments(adminID, isAdmin, postID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func UpdateAdminCommunityPost(c *gin.Context) {
	postID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req updateCommunityPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err := controller_admin.UpdateAdminCommunityPost(adminID, isAdmin, postID, controller_admin.UpdateAdminCommunityPostReq{
		PostType: req.PostType,
		Title:    req.Title,
		Content:  req.Content,
	}); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "community_post", &postID, "", req.Title)
	middleware.SetAdminAccessAction(c, "community_post.update", "修改内容")
	response.OK(c, nil)
}

func DeleteAdminCommunityPost(c *gin.Context) {
	postID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err := controller_admin.DeleteAdminCommunityPost(adminID, isAdmin, postID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "community_post", &postID, "", "省钱社区内容")
	middleware.SetAdminAccessAction(c, "community_post.delete", "删除内容")
	response.OK(c, nil)
}

func DeleteAdminCommunityComment(c *gin.Context) {
	commentID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err := controller_admin.DeleteAdminCommunityComment(adminID, isAdmin, commentID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "community_comment", &commentID, "", "省钱社区评论")
	middleware.SetAdminAccessAction(c, "community_comment.delete", "删除评论")
	response.OK(c, nil)
}

func UpdateCommunityUserRestriction(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req updateCommunityUserRestrictionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	if err := controller_admin.UpdateCommunityUserRestriction(adminID, isAdmin, userID, controller_admin.UpdateCommunityUserRestrictionReq{
		CommunityPostDisabled:    req.CommunityPostDisabled,
		CommunityCommentDisabled: req.CommunityCommentDisabled,
	}); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &userID, "", "省钱社区权限用户")
	middleware.SetAdminAccessAction(c, "community_user.update_restriction", "修改用户省钱社区权限")
	response.OK(c, nil)
}
