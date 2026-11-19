package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/idgs"
	"gorm.io/gorm"
)

// SearchGroupResult 表示同归属群组检索结果。
type SearchGroupResult struct {
	GroupID   string `json:"group_id"`
	GroupName string `json:"group_name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	GroupType string `json:"group_type"`
}

// SearchGroup 在调用者同归属下按群组 ID 精确检索群组。
func SearchGroup(ctx context.Context, imUserID, groupID string) (*SearchGroupResult, error) {
	userRepo := repository.NewUserRepositoryWithDB(datasource.Datasource.Mysql.GetClient().WithContext(ctx))
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, nil
	}

	group, err := syncOwnedGroupSnapshotContext(ctx, caller.OwnerID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}

	return &SearchGroupResult{
		GroupID:   group.IMGroupID,
		GroupName: group.Name,
		GroupType: group.GroupType,
	}, nil
}

// SearchGroups 在调用者同归属下按群组名称或 GroupID 模糊搜索群组。
// 用于 Flutter App 端的加群搜索功能。
func SearchGroups(ctx context.Context, imUserID, keyword string) ([]SearchGroupResult, error) {
	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []SearchGroupResult{}, nil
	}

	groupRepo := repository.NewGroupRepository()
	groups, err := groupRepo.SearchByOwner(caller.OwnerID, keyword)
	if err != nil {
		return nil, err
	}

	results := make([]SearchGroupResult, 0, len(groups))
	for _, g := range groups {
		results = append(results, SearchGroupResult{
			GroupID:   g.IMGroupID,
			GroupName: g.Name,
			GroupType: g.GroupType,
		})
	}
	return results, nil
}

func syncOwnedGroupSnapshot(expectedOwnerID uint, groupID string) (*model.Group, error) {
	return syncOwnedGroupSnapshotContext(context.Background(), expectedOwnerID, groupID)
}

func syncOwnedGroupSnapshotContext(ctx context.Context, expectedOwnerID uint, groupID string) (*model.Group, error) {
	groupRepo := repository.NewGroupRepositoryWithDB(datasource.Datasource.Mysql.GetClient().WithContext(ctx))

	localGroup, err := groupRepo.FindByIMGroupID(groupID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		localGroup = nil
	}

	if localGroup != nil && localGroup.Status != "active" {
		return nil, nil
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		if localGroup != nil && localGroup.OwnerID == expectedOwnerID {
			return localGroup, nil
		}
		return nil, nil
	}

	infoMap, err := imClient.GroupGetInfoContext(ctx, []string{groupID})
	if err != nil {
		if localGroup != nil && localGroup.OwnerID == expectedOwnerID {
			return localGroup, nil
		}
		return nil, err
	}

	info, ok := infoMap[groupID]
	if !ok {
		return nil, nil
	}

	ownerAccount := strings.TrimSpace(info.OwnerAccount)
	if ownerAccount == "" {
		return nil, nil
	}

	userRepo := repository.NewUserRepositoryWithDB(datasource.Datasource.Mysql.GetClient().WithContext(ctx))
	ownerUser, err := userRepo.FindByIMUserID(ownerAccount)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	groupName := strings.TrimSpace(info.Name)
	if groupName == "" {
		if localGroup != nil && strings.TrimSpace(localGroup.Name) != "" {
			groupName = localGroup.Name
		} else {
			groupName = groupID
		}
	}

	groupType := strings.TrimSpace(info.Type)
	if groupType == "" {
		if localGroup != nil && strings.TrimSpace(localGroup.GroupType) != "" {
			groupType = localGroup.GroupType
		} else {
			groupType = "Work"
		}
	}

	if localGroup == nil {
		if ownerUser.OwnerID != expectedOwnerID {
			return nil, nil
		}

		group := &model.Group{
			BizID:     idgs.Global.GroupGenerate().Int64(),
			IMGroupID: groupID,
			Name:      groupName,
			GroupType: groupType,
			OwnerID:   ownerUser.OwnerID,
			CreatedBy: ownerUser.ID,
			Status:    "active",
			CreatedAt: time.Now(),
		}
		if err := groupRepo.Create(group); err != nil {
			existingGroup, findErr := groupRepo.FindByIMGroupID(groupID)
			if findErr != nil {
				return nil, err
			}
			if existingGroup.Status != "active" || existingGroup.OwnerID != expectedOwnerID {
				return nil, nil
			}
			return existingGroup, nil
		}
		return group, nil
	}

	changed := false
	if localGroup.Name != groupName {
		localGroup.Name = groupName
		changed = true
	}
	if localGroup.GroupType != groupType {
		localGroup.GroupType = groupType
		changed = true
	}
	if localGroup.OwnerID != ownerUser.OwnerID {
		localGroup.OwnerID = ownerUser.OwnerID
		changed = true
	}
	if localGroup.CreatedBy != ownerUser.ID {
		localGroup.CreatedBy = ownerUser.ID
		changed = true
	}
	if changed {
		if err := groupRepo.SaveMirror(context.Background(), localGroup); err != nil {
			return nil, err
		}
	}

	if localGroup.OwnerID != expectedOwnerID {
		return nil, nil
	}
	return localGroup, nil
}
