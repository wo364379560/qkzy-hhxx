package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"gorm.io/gorm"
)

type GroupMessagePinResult struct {
	GroupID               uint   `json:"group_id"`
	IMGroupID             string `json:"im_group_id"`
	MessageID             string `json:"message_id"`
	MessageSeq            int64  `json:"message_seq"`
	MessageSenderIMUserID string `json:"message_sender_im_user_id"`
	MessageSenderName     string `json:"message_sender_name"`
	MessageType           string `json:"message_type"`
	MessageAbstract       string `json:"message_abstract"`
	Status                string `json:"status"`
	PinnedAt              string `json:"pinned_at"`
}

type SaveGroupMessagePinReq struct {
	GroupID               string
	MessageID             string
	MessageSeq            int64
	MessageSenderIMUserID string
	MessageSenderName     string
	MessageType           string
	MessageAbstract       string
}

func GetGroupMessagePin(imUserID, groupID string) (*GroupMessagePinResult, error) {
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

	localGroup, err := repository.NewGroupRepository().FindByIMGroupID(groupID)
	if err != nil {
		return nil, err
	}

	pin, err := repository.NewGroupMessagePinRepository().FindActiveByGroupID(localGroup.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &GroupMessagePinResult{
		GroupID:               pin.GroupID,
		IMGroupID:             pin.IMGroupID,
		MessageID:             pin.MessageID,
		MessageSeq:            pin.MessageSeq,
		MessageSenderIMUserID: pin.MessageSenderIMUserID,
		MessageSenderName:     pin.MessageSenderName,
		MessageType:           pin.MessageType,
		MessageAbstract:       pin.MessageAbstract,
		Status:                pin.Status,
		PinnedAt:              pin.PinnedAt.Format(time.DateTime),
	}, nil
}

func SaveGroupMessagePin(imUserID string, req SaveGroupMessagePinReq) (*GroupMessagePinResult, error) {
	groupID := strings.TrimSpace(req.GroupID)
	if groupID == "" {
		return nil, errors.New("group_id不能为空")
	}
	if strings.TrimSpace(req.MessageID) == "" {
		return nil, errors.New("message_id不能为空")
	}

	group, err := SearchGroup(context.Background(), imUserID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("群组不存在")
	}

	role, err := tencentim.NewClient().GroupGetRole(groupID, imUserID)
	if err != nil {
		return nil, err
	}
	if !canManageGroupMessagePinRole(role) {
		return nil, errors.New("仅群主或管理员可置顶群消息")
	}

	localGroup, err := repository.NewGroupRepository().FindByIMGroupID(groupID)
	if err != nil {
		return nil, err
	}
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	pin := &model.GroupMessagePin{
		GroupID:               localGroup.ID,
		IMGroupID:             localGroup.IMGroupID,
		MessageID:             strings.TrimSpace(req.MessageID),
		MessageSeq:            req.MessageSeq,
		MessageSenderIMUserID: strings.TrimSpace(req.MessageSenderIMUserID),
		MessageSenderName:     strings.TrimSpace(req.MessageSenderName),
		MessageType:           strings.TrimSpace(req.MessageType),
		MessageAbstract:       strings.TrimSpace(req.MessageAbstract),
		OperatorUserID:        &user.ID,
		Status:                "active",
		PinnedAt:              time.Now(),
	}
	if err := repository.NewGroupMessagePinRepository().ReplaceActive(pin); err != nil {
		return nil, err
	}

	return &GroupMessagePinResult{
		GroupID:               pin.GroupID,
		IMGroupID:             pin.IMGroupID,
		MessageID:             pin.MessageID,
		MessageSeq:            pin.MessageSeq,
		MessageSenderIMUserID: pin.MessageSenderIMUserID,
		MessageSenderName:     pin.MessageSenderName,
		MessageType:           pin.MessageType,
		MessageAbstract:       pin.MessageAbstract,
		Status:                pin.Status,
		PinnedAt:              pin.PinnedAt.Format(time.DateTime),
	}, nil
}

func RemoveGroupMessagePin(imUserID, groupID string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("group_id不能为空")
	}

	group, err := SearchGroup(context.Background(), imUserID, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("群组不存在")
	}

	role, err := tencentim.NewClient().GroupGetRole(groupID, imUserID)
	if err != nil {
		return err
	}
	if !canManageGroupMessagePinRole(role) {
		return errors.New("仅群主或管理员可取消置顶群消息")
	}

	localGroup, err := repository.NewGroupRepository().FindByIMGroupID(groupID)
	if err != nil {
		return err
	}
	return repository.NewGroupMessagePinRepository().RemoveActiveByGroupID(localGroup.ID)
}

func canManageGroupMessagePin(imUserID string, members []tencentim.GroupMember) bool {
	for _, member := range members {
		if member.Account != imUserID {
			continue
		}
		return member.Role == "Owner" || member.Role == "Admin"
	}
	return false
}

func canManageGroupMessagePinRole(role string) bool {
	return role == "Owner" || role == "Admin"
}
