package controller_chat

import (
	"context"
	"errors"
	"strings"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

func GetGroupNotice(imUserID, groupID string) (*controller_admin.GroupNoticeResult, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("group_id不能为空")
	}

	group, err := SearchGroup(context.Background(), imUserID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("群组不存在")
	}

	groupRepo := repository.NewGroupRepository()
	localGroup, err := groupRepo.FindByIMGroupID(groupID)
	if err != nil {
		return nil, err
	}

	notice, err := repository.NewGroupNoticeRepository().FindByGroupID(localGroup.ID)
	if err == nil {
		return &controller_admin.GroupNoticeResult{
			GroupID:  localGroup.ID,
			Content:  notice.Content,
			Pinned:   notice.Pinned,
			Summary:  notice.Summary,
			Source:   "backend",
			HasValue: strings.TrimSpace(notice.Content) != "",
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &controller_admin.GroupNoticeResult{
		GroupID:  localGroup.ID,
		Content:  "",
		Summary:  "",
		Source:   "fallback",
		HasValue: false,
	}, nil
}

func SaveGroupNotice(imUserID, groupID, content string, pinned ...*bool) (*controller_admin.GroupNoticeResult, error) {
	groupID = strings.TrimSpace(groupID)
	content = strings.TrimSpace(content)
	if groupID == "" {
		return nil, errors.New("group_id不能为空")
	}
	if controller_admin.Utf8ByteLength(content) > 5000 {
		return nil, errors.New("群公告长度不能超过5000字节")
	}

	group, err := SearchGroup(context.Background(), imUserID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("群组不存在")
	}

	groupRepo := repository.NewGroupRepository()
	localGroup, err := groupRepo.FindByIMGroupID(groupID)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	// Work groups retain their existing member-editable announcement behavior.
	if localGroup.GroupType == "Work" && (len(pinned) == 0 || pinned[0] == nil) {
		if err := ensureGroupMember(imUserID, groupID); err != nil {
			return nil, err
		}
	} else if err := ensureGroupOwnerOrAdmin(imUserID, groupID); err != nil {
		return nil, err
	}
	summary := controller_admin.BuildNoticeSummary(content, 240)
	notice := &model.GroupNotice{
		GroupID:         localGroup.ID,
		Content:         content,
		Summary:         summary,
		UpdatedByUserID: &user.ID,
	}
	if err := repository.NewGroupNoticeRepository().Upsert(notice, pinned...); err != nil {
		return nil, err
	}

	return &controller_admin.GroupNoticeResult{
		GroupID:  localGroup.ID,
		Content:  content,
		Pinned:   notice.Pinned,
		Summary:  summary,
		Source:   "backend",
		HasValue: content != "",
	}, nil
}

func PinGroupNotice(imUserID, groupID string, pinned bool) (*controller_admin.GroupNoticeResult, error) {
	if err := ensureGroupOwnerOrAdmin(imUserID, groupID); err != nil {
		return nil, err
	}
	group, err := repository.NewGroupRepository().FindByIMGroupID(groupID)
	if err != nil {
		return nil, err
	}
	repo := repository.NewGroupNoticeRepository()
	if _, err := repo.FindByGroupID(group.ID); err != nil {
		return nil, errors.New("请先保存群公告")
	}
	if err := repo.SetPinned(group.ID, pinned); err != nil {
		return nil, err
	}
	return GetGroupNotice(imUserID, groupID)
}
