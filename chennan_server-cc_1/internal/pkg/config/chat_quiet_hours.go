package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

const (
	chatQuietHoursPolicySettingKey = "chat_quiet_hours_policy"
	defaultQuietHoursStart         = "00:00"
	defaultQuietHoursEnd           = "08:00"
	defaultQuietHoursTimeZone      = "Asia/Shanghai"
)

const (
	// ChatQuietHoursActionRegister restricts self-registration during quiet hours.
	ChatQuietHoursActionRegister = "register"
	// ChatQuietHoursActionSendMessage restricts IM message sending during quiet hours.
	ChatQuietHoursActionSendMessage = "send_message"
	// ChatQuietHoursActionCreateGroup restricts user group creation during quiet hours.
	ChatQuietHoursActionCreateGroup = "create_group"
	// ChatQuietHoursActionApplyJoinGroup restricts users from applying to groups during quiet hours.
	ChatQuietHoursActionApplyJoinGroup = "apply_join_group"
	// ChatQuietHoursActionInviteJoinGroup restricts inviting users into groups during quiet hours.
	ChatQuietHoursActionInviteJoinGroup = "invite_join_group"
	// ChatQuietHoursActionAddFriend restricts adding friends during quiet hours.
	ChatQuietHoursActionAddFriend = "add_friend"
)

// ChatQuietHoursPolicy defines the configured quiet-hours action restrictions.
type ChatQuietHoursPolicy struct {
	Enabled          bool   `json:"enabled"`
	TimeZone         string `json:"timezone"`
	Start            string `json:"start"`
	End              string `json:"end"`
	BlockRegister    bool   `json:"block_register"`
	BlockSendMessage bool   `json:"block_send_message"`
	BlockCreateGroup bool   `json:"block_create_group"`
	BlockAddFriend   bool   `json:"block_add_friend"`
}

// GetChatQuietHoursPolicy returns the global quiet-hours policy.
func GetChatQuietHoursPolicy(ctx context.Context, db *gorm.DB) (ChatQuietHoursPolicy, error) {
	if db == nil {
		return ChatQuietHoursPolicy{}, errors.New("mysql datasource not initialized")
	}

	policy := defaultChatQuietHoursPolicy()

	var setting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", chatQuietHoursPolicySettingKey).
		Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return policy, nil
		}
		return ChatQuietHoursPolicy{}, err
	}

	if err := json.Unmarshal([]byte(setting.Value), &policy); err != nil {
		return ChatQuietHoursPolicy{}, err
	}
	if err := normalizeChatQuietHoursPolicy(&policy); err != nil {
		return ChatQuietHoursPolicy{}, err
	}
	return policy, nil
}

// SetChatQuietHoursPolicyTx persists the global quiet-hours policy in the provided transaction or DB handle.
func SetChatQuietHoursPolicyTx(ctx context.Context, db *gorm.DB, policy ChatQuietHoursPolicy, updatedBy uint) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}
	if err := normalizeChatQuietHoursPolicy(&policy); err != nil {
		return err
	}

	raw, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	if len(raw) > 255 {
		return errors.New("静默时段配置过长，请缩短配置项")
	}

	return upsertSystemSettingTx(ctx, db, chatQuietHoursPolicySettingKey, string(raw), updatedBy)
}

func defaultChatQuietHoursPolicy() ChatQuietHoursPolicy {
	return ChatQuietHoursPolicy{
		Enabled:          false,
		TimeZone:         currentSystemTimeZone(),
		Start:            defaultQuietHoursStart,
		End:              defaultQuietHoursEnd,
		BlockRegister:    true,
		BlockSendMessage: true,
		BlockCreateGroup: true,
		BlockAddFriend:   true,
	}
}

// DefaultChatQuietHoursPolicy returns the default quiet-hours policy template.
func DefaultChatQuietHoursPolicy() ChatQuietHoursPolicy {
	return defaultChatQuietHoursPolicy()
}

func currentSystemTimeZone() string {
	timeZone := strings.TrimSpace(Cfg.Get().System.TimeZone)
	if timeZone == "" {
		return defaultQuietHoursTimeZone
	}
	return timeZone
}

func normalizeChatQuietHoursPolicy(policy *ChatQuietHoursPolicy) error {
	if policy == nil {
		return errors.New("静默时段配置不能为空")
	}

	policy.TimeZone = strings.TrimSpace(policy.TimeZone)
	if policy.TimeZone == "" {
		policy.TimeZone = currentSystemTimeZone()
	}
	if _, err := time.LoadLocation(policy.TimeZone); err != nil {
		return errors.New("时区配置无效")
	}

	policy.Start = strings.TrimSpace(policy.Start)
	if policy.Start == "" {
		policy.Start = defaultQuietHoursStart
	}
	policy.End = strings.TrimSpace(policy.End)
	if policy.End == "" {
		policy.End = defaultQuietHoursEnd
	}

	if _, err := time.Parse("15:04", policy.Start); err != nil {
		return errors.New("开始时间格式无效，请使用 HH:MM")
	}
	if _, err := time.Parse("15:04", policy.End); err != nil {
		return errors.New("结束时间格式无效，请使用 HH:MM")
	}
	if policy.Start == policy.End {
		return errors.New("开始时间和结束时间不能相同")
	}

	return nil
}

// NormalizeChatQuietHoursPolicy validates and fills default values for a quiet-hours policy.
func NormalizeChatQuietHoursPolicy(policy *ChatQuietHoursPolicy) error {
	return normalizeChatQuietHoursPolicy(policy)
}

// IsChatQuietHoursActionBlockedNow reports whether the specified action is blocked at the given time.
func IsChatQuietHoursActionBlockedNow(policy ChatQuietHoursPolicy, action string, now time.Time) (bool, error) {
	if !policy.Enabled {
		return false, nil
	}

	enabled := false
	switch action {
	case ChatQuietHoursActionRegister:
		enabled = policy.BlockRegister
	case ChatQuietHoursActionSendMessage:
		enabled = policy.BlockSendMessage
	case ChatQuietHoursActionCreateGroup:
		enabled = policy.BlockCreateGroup
	case ChatQuietHoursActionApplyJoinGroup:
		enabled = policy.BlockCreateGroup
	case ChatQuietHoursActionInviteJoinGroup:
		enabled = policy.BlockCreateGroup
	case ChatQuietHoursActionAddFriend:
		enabled = policy.BlockAddFriend
	default:
		return false, nil
	}
	if !enabled {
		return false, nil
	}

	loc, err := time.LoadLocation(policy.TimeZone)
	if err != nil {
		return false, err
	}
	localNow := now.In(loc)

	startMinute, err := parseClockMinute(policy.Start)
	if err != nil {
		return false, err
	}
	endMinute, err := parseClockMinute(policy.End)
	if err != nil {
		return false, err
	}
	return isWithinDailyBlockedWindow(localNow, startMinute, endMinute), nil
}

func parseClockMinute(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func isWithinDailyBlockedWindow(now time.Time, startMinute, endMinute int) bool {
	currentMinute := now.Hour()*60 + now.Minute()

	if startMinute == endMinute {
		return true
	}
	if startMinute < endMinute {
		return currentMinute >= startMinute && currentMinute < endMinute
	}
	return currentMinute >= startMinute || currentMinute < endMinute
}

// ChatQuietHoursBlockedMessage returns a user-facing Chinese message for the blocked action.
func ChatQuietHoursBlockedMessage(action string, policy ChatQuietHoursPolicy) string {
	window := fmt.Sprintf("%s-%s", policy.Start, policy.End)

	switch action {
	case ChatQuietHoursActionRegister:
		return fmt.Sprintf("当前处于系统静默时段（%s），暂不开放新用户注册，请稍后再试", window)
	case ChatQuietHoursActionSendMessage:
		return fmt.Sprintf("当前处于静默时段（%s），无法发送消息，请于%s之后重试", window, policy.End)
	case ChatQuietHoursActionCreateGroup:
		return fmt.Sprintf("当前处于系统静默时段（%s），暂时禁止创建群组，请稍后再试", window)
	case ChatQuietHoursActionApplyJoinGroup:
		return fmt.Sprintf("当前处于系统静默时段（%s），暂时禁止申请加入群组，请稍后再试", window)
	case ChatQuietHoursActionInviteJoinGroup:
		return fmt.Sprintf("当前处于系统静默时段（%s），暂时禁止邀请成员入群，请稍后再试", window)
	case ChatQuietHoursActionAddFriend:
		return fmt.Sprintf("当前处于系统静默时段（%s），暂时禁止添加好友，请稍后再试", window)
	default:
		return fmt.Sprintf("当前处于系统静默时段（%s），请稍后再试", window)
	}
}
