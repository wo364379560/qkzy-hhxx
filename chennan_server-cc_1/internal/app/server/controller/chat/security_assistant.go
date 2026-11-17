package controller_chat

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/pkg/logger"
)

// SecurityAssistantIMUserID 安全助手系统账号（腾讯云 IM UserID）。
const SecurityAssistantIMUserID = "security_assistant"

const securityAssistantNick = "安全助手"

var ensureSecurityAssistantOnce sync.Once

// EnsureSecurityAssistantAccount 确保安全助手 IM 账号已导入。
func EnsureSecurityAssistantAccount() {
	ensureSecurityAssistantOnce.Do(func() {
		client := tencentim.NewClient()
		if !client.IsConfigured() {
			logger.Warnf("security assistant: tencent IM not configured, skip account import")
			return
		}
		if err := client.AccountImport(SecurityAssistantIMUserID, securityAssistantNick, ""); err != nil {
			logger.Warnf("security assistant: account import failed: %v", err)
			return
		}
		logger.Infof("security assistant: account ready (%s)", SecurityAssistantIMUserID)
	})
}

func notifySecurityAssistantAsync(toIMUserID, text, pushTitle, pushDesc string) {
	toIMUserID = strings.TrimSpace(toIMUserID)
	text = strings.TrimSpace(text)
	if toIMUserID == "" || text == "" {
		return
	}
	go func() {
		EnsureSecurityAssistantAccount()
		client := tencentim.NewClient()
		if !client.IsConfigured() {
			return
		}
		pushDesc = truncateRunes(pushDesc, 64)
		push := tencentim.NewC2CTextOfflinePushInfo(
			pushTitle,
			pushDesc,
			"",
			securityAssistantNick,
			pushDesc,
		)
		if _, err := client.BatchSendTextMessageWithPush(
			SecurityAssistantIMUserID,
			[]string{toIMUserID},
			text,
			false,
			push,
		); err != nil {
			logger.Warnf("security assistant: send to %s failed: %v", toIMUserID, err)
		}
	}()
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}

func maskIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "未知"
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	if len(ip) > 8 {
		return ip[:len(ip)/2] + "…"
	}
	return ip
}

// NotifyLoginSuccessAlert 登录成功后发安全提醒（受「异地登录提醒」开关控制）。
func NotifyLoginSuccessAlert(user *model.User, meta SessionMeta) {
	if user == nil || strings.TrimSpace(user.IMUserID) == "" {
		return
	}
	normalized := normalizeSessionMeta(meta)
	if isWebDevice(normalized.Platform, normalized.DeviceName) {
		return
	}
	settings, err := repository.NewUserSecuritySettingsRepository().GetOrDefault(user.ID)
	if err != nil || !settings.RemoteLoginAlert {
		return
	}

	deviceName := deviceDisplayName(normalized.DeviceName, normalized.Platform)
	platform := normalized.Platform
	if platform == "" {
		platform = "未知"
	}
	when := time.Now().Format("2006-01-02 15:04")
	text := fmt.Sprintf(
		"【安全提醒】检测到账号登录\n设备：%s\n平台：%s\nIP：%s\n时间：%s\n若非本人操作，请尽快修改密码，并在「登录设备管理」中移除可疑设备。",
		deviceName,
		platform,
		maskIP(normalized.IP),
		when,
	)
	notifySecurityAssistantAsync(user.IMUserID, text, securityAssistantNick, "检测到账号登录，请确认是否本人操作")
}

// NotifyLoginChallengePending 新设备登录待确认时通知已登录设备。
func NotifyLoginChallengePending(user *model.User, challenge *model.UserLoginChallenge) {
	if user == nil || challenge == nil || strings.TrimSpace(user.IMUserID) == "" {
		return
	}
	if isWebDevice(challenge.Platform, challenge.DeviceName) {
		return
	}
	deviceName := deviceDisplayName(challenge.DeviceName, challenge.Platform)
	platform := challenge.Platform
	if platform == "" {
		platform = "未知"
	}
	expires := challenge.ExpiresAt.Format("15:04")
	text := fmt.Sprintf(
		"【安全助手】新设备请求登录\n设备：%s\n平台：%s\nIP：%s\n确认码：%s\n有效期至：%s\n\n请打开「我的 → 登录设备管理 → 待确认登录」进行允许或拒绝。\n请核对确认码后再操作。",
		deviceName,
		platform,
		maskIP(challenge.IP),
		challenge.ShortCode,
		expires,
	)
	notifySecurityAssistantAsync(
		user.IMUserID,
		text,
		securityAssistantNick,
		fmt.Sprintf("新设备请求登录，确认码 %s", challenge.ShortCode),
	)
}

// NotifyNewDeviceApproved 新设备通过授权后提醒账号。
func NotifyNewDeviceApproved(user *model.User, meta SessionMeta) {
	if user == nil || strings.TrimSpace(user.IMUserID) == "" {
		return
	}
	normalized := normalizeSessionMeta(meta)
	if isWebDevice(normalized.Platform, normalized.DeviceName) {
		return
	}
	settings, err := repository.NewUserSecuritySettingsRepository().GetOrDefault(user.ID)
	if err != nil || !settings.RemoteLoginAlert {
		return
	}
	deviceName := deviceDisplayName(normalized.DeviceName, normalized.Platform)
	text := fmt.Sprintf(
		"【安全提醒】新设备已通过验证并登录\n设备：%s\nIP：%s\n时间：%s\n如非本人操作，请立即修改密码。",
		deviceName,
		maskIP(normalized.IP),
		time.Now().Format("2006-01-02 15:04"),
	)
	notifySecurityAssistantAsync(user.IMUserID, text, securityAssistantNick, "新设备已通过验证并登录")
}
