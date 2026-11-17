package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/groupslowcache"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

const (
	minGroupSlowModeSeconds = 5
	maxGroupSlowModeSeconds = 3600
)

type GroupSlowModeResult struct {
	GroupID         string `json:"group_id"`
	SlowModeEnabled bool   `json:"slow_mode_enabled"`
	SlowModeSeconds int    `json:"slow_mode_seconds"`
	CanSetSlowMode  bool   `json:"can_set_slow_mode"`
}

func GetGroupSlowMode(imUserID, groupID string) (*GroupSlowModeResult, error) {
	group, canSet, err := loadGroupSlowModeContext(imUserID, groupID)
	if err != nil {
		return nil, err
	}
	return buildGroupSlowModeResult(group, canSet), nil
}

func SaveGroupSlowMode(imUserID, groupID string, enabled bool, seconds int) (*GroupSlowModeResult, error) {
	group, canSet, err := loadGroupSlowModeContext(imUserID, groupID)
	if err != nil {
		return nil, err
	}
	if !canSet {
		return nil, errors.New("仅群主可以设置发言频率限制")
	}

	if !enabled {
		seconds = 0
	} else {
		if seconds < minGroupSlowModeSeconds || seconds > maxGroupSlowModeSeconds {
			return nil, errors.New("发言间隔需设置在5秒到3600秒之间")
		}
	}

	version := time.Now().UnixNano()
	if err := repository.NewGroupRepository().UpdateSlowMode(group.ID, enabled, seconds, version); err != nil {
		return nil, err
	}
	group.SlowModeEnabled = enabled
	group.SlowModeSeconds = seconds
	group.SlowModeVersion = version
	groupslowcache.Invalidate(context.Background(), group.IMGroupID)
	return buildGroupSlowModeResult(group, true), nil
}

func loadGroupSlowModeContext(imUserID, groupID string) (*model.Group, bool, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, false, errors.New("group_id不能为空")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return nil, false, err
	}
	group, err := repository.NewGroupRepository().FindActiveByIMGroupID(groupID)
	if err != nil {
		return nil, false, errors.New("群组不存在")
	}
	permission, err := canManageRegularMembersFromDB(groupID, imUserID)
	if err != nil {
		return nil, false, err
	}
	return group, permission != nil && permission.IsOwner, nil
}

func buildGroupSlowModeResult(group *model.Group, canSet bool) *GroupSlowModeResult {
	seconds := group.SlowModeSeconds
	if !group.SlowModeEnabled {
		seconds = 0
	}
	return &GroupSlowModeResult{
		GroupID:         group.IMGroupID,
		SlowModeEnabled: group.SlowModeEnabled,
		SlowModeSeconds: seconds,
		CanSetSlowMode:  canSet,
	}
}
