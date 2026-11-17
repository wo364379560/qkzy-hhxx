package controller_admin

import (
	"errors"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

func resolveManagedOwnerID(actorID uint, isAdmin bool, requestedOwnerID uint) (uint, error) {
	actor, err := BuildActorContext(actorID)
	if err != nil {
		return 0, err
	}

	if !isAdmin || IsAssistantRole(actor.Role) {
		if requestedOwnerID > 0 && requestedOwnerID != actor.AdminID {
			return 0, errors.New("助理不能指定其他归属人")
		}
		return actor.AdminID, nil
	}

	ownerID := requestedOwnerID
	if ownerID == 0 {
		ownerID = actor.AdminID
	}
	if err := ensureOwnerExists(ownerID); err != nil {
		return 0, err
	}
	if !CanManageOwnerID(actor, ownerID) {
		return 0, errors.New("无权指定该归属人")
	}
	return ownerID, nil
}

func ensureOwnerExists(ownerID uint) error {
	repo := repository.NewAdminUserRepository()
	if _, err := repo.FindByID(ownerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("归属人不存在")
		}
		return err
	}
	return nil
}

func findManagedUser(actorID uint, isAdmin bool, userID uint) (*model.User, error) {
	repo := repository.NewUserRepository()
	user, err := repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageUser(actor, user) {
		return nil, errors.New("无权操作此用户")
	}
	return user, nil
}

func findManagedUserByIMUserID(actorID uint, isAdmin bool, imUserID string) (*model.User, error) {
	repo := repository.NewUserRepository()
	user, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageUser(actor, user) {
		return nil, errors.New("无权操作此用户")
	}
	return user, nil
}

func findManagedGroupByIMGroupID(actorID uint, isAdmin bool, imGroupID string) (*model.Group, error) {
	repo := repository.NewGroupRepository()
	group, err := repo.FindByIMGroupID(imGroupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("群组不存在")
		}
		return nil, err
	}
	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageOwnerID(actor, group.OwnerID) {
		return nil, errors.New("无权操作此群组")
	}
	return group, nil
}

func findManagedGroup(actorID uint, groupID uint) (*model.Group, error) {
	repo := repository.NewGroupRepository()
	group, err := repo.FindByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("群组不存在")
		}
		return nil, err
	}
	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageOwnerID(actor, group.OwnerID) {
		return nil, errors.New("无权操作此群组")
	}
	return group, nil
}

func findManagedInviteCode(actorID uint, isAdmin bool, codeID uint) (*model.InviteCode, error) {
	repo := repository.NewInviteCodeRepository()
	code, err := repo.FindByID(codeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("邀请码不存在")
		}
		return nil, err
	}
	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageOwnerID(actor, code.OwnerID) {
		return nil, errors.New("无权操作此邀请码")
	}
	return code, nil
}

func findManagedReport(actorID uint, isAdmin bool, reportID uint) (*model.Report, error) {
	repo := repository.NewReportRepository()
	report, err := repo.FindByID(reportID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("举报不存在")
		}
		return nil, err
	}
	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageOwnerID(actor, report.OwnerID) {
		return nil, errors.New("无权操作此举报")
	}
	return report, nil
}

func findManagedCommunityPost(actorID uint, isAdmin bool, postID uint) (*model.CommunityPost, error) {
	post, err := repository.NewCommunityRepository().FindPostByIDAnyStatus(postID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("内容不存在")
		}
		return nil, err
	}

	user, err := repository.NewUserRepository().FindByID(post.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("内容作者不存在")
		}
		return nil, err
	}

	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageOwnerID(actor, user.OwnerID) {
		return nil, errors.New("无权操作此帖子")
	}
	return post, nil
}

func findManagedCommunityComment(actorID uint, isAdmin bool, commentID uint) (*model.CommunityPostComment, error) {
	comment, err := repository.NewCommunityRepository().FindCommentByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("评论不存在")
		}
		return nil, err
	}

	user, err := repository.NewUserRepository().FindByID(comment.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("评论作者不存在")
		}
		return nil, err
	}

	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageOwnerID(actor, user.OwnerID) {
		return nil, errors.New("无权操作此评论")
	}
	return comment, nil
}

func findManagedAssistant(actorID uint, assistantID uint) (*model.AdminUser, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(assistantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("助理不存在")
		}
		return nil, err
	}
	if admin.Role != RoleAssistant {
		return nil, errors.New("该账号不是助理")
	}

	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !CanManageAdminUser(actor, admin) {
		return nil, errors.New("无权操作此助理")
	}
	return admin, nil
}

func findManagedManager(actorID uint, managerID uint) (*model.AdminUser, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(managerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("管理员不存在")
		}
		return nil, err
	}
	if admin.Role != RoleAdmin {
		return nil, errors.New("该账号不是管理员")
	}

	actor, ctxErr := BuildActorContext(actorID)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if !IsSuperAdminRole(actor.Role) {
		return nil, errors.New("仅超级管理员可操作管理员账号")
	}
	return admin, nil
}
