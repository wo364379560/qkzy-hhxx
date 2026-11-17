package tencentim

import (
	"strings"
	"unicode"
)

const (
	xiaoMiPushChannelID      = "157799"
	xiaoMiChatTemplateID     = "M10291"
	xiaoMiTemplateValueLimit = 128
)

// NewC2CTextOfflinePushInfo 生成单聊文字/系统通知的离线推送参数。
// M10291: 标题 【好友聊天】{$keywords1$} / 内容 {$keywords2$}: {$keywords3$}
// keywords1=发送者，keywords2=发送者，keywords3=消息摘要。
func NewC2CTextOfflinePushInfo(title, desc, ext, senderName, summary string) *OfflinePushInfo {
	resolvedSender := sanitizeXiaoMiTemplateValue(senderName, title)
	resolvedSummary := sanitizeXiaoMiTemplateValue(summary, desc)
	return newXiaoMiOfflinePushInfo(
		title,
		desc,
		ext,
		map[string]string{
			"keywords1": resolvedSender,
			"keywords2": resolvedSender,
			"keywords3": resolvedSummary,
		},
	)
}

// NewGroupMessageOfflinePushInfo 生成群消息的离线推送参数。
// 与 Flutter / 控制台一致，统一使用 M10291 三参数：
// keywords1=群名，keywords2=发送者，keywords3=消息摘要。
func NewGroupMessageOfflinePushInfo(title, desc, ext, senderName, groupName, summary string) *OfflinePushInfo {
	resolvedSender := sanitizeXiaoMiTemplateValue(senderName, "成员")
	resolvedGroup := sanitizeXiaoMiTemplateValue(groupName, title)
	resolvedSummary := sanitizeXiaoMiTemplateValue(summary, desc)
	return newXiaoMiOfflinePushInfo(
		title,
		desc,
		ext,
		map[string]string{
			"keywords1": resolvedGroup,
			"keywords2": resolvedSender,
			"keywords3": resolvedSummary,
		},
	)
}

func newXiaoMiOfflinePushInfo(
	title, desc, ext string,
	templateParam map[string]string,
) *OfflinePushInfo {
	return &OfflinePushInfo{
		PushFlag: 0,
		Title:    strings.TrimSpace(title),
		Desc:     strings.TrimSpace(desc),
		Ext:      strings.TrimSpace(ext),
		AndroidInfo: &OfflinePushAndroidInfo{
			XiaoMiChannelID:     xiaoMiPushChannelID,
			XiaoMiTemplateID:    xiaoMiChatTemplateID,
			XiaoMiTemplateParam: templateParam,
		},
	}
}

// sanitizeXiaoMiTemplateValue 遵守小米模板变量约束：非空、最长 128 个字符，
// 并移除 emoji、Unicode Other Symbol (So)、控制/格式与未分配字符。
// 仅用于 XiaoMiTemplateParam，不改写共用 Title/Desc。
func sanitizeXiaoMiTemplateValue(value, fallback string) string {
	result := normalizeXiaoMiTemplateValue(value)
	if result == "" {
		result = normalizeXiaoMiTemplateValue(fallback)
	}
	if result == "" {
		result = "消息"
	}
	runes := []rune(result)
	if len(runes) > xiaoMiTemplateValueLimit {
		result = string(runes[:xiaoMiTemplateValueLimit])
	}
	return result
}

func normalizeXiaoMiTemplateValue(value string) string {
	var builder strings.Builder
	lastWasSpace := true
	for _, r := range strings.TrimSpace(value) {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				builder.WriteByte(' ')
				lastWasSpace = true
			}
			continue
		}
		if isXiaoMiDisallowedRune(r) {
			continue
		}
		builder.WriteRune(r)
		lastWasSpace = false
	}
	return strings.TrimSpace(builder.String())
}

func isXiaoMiDisallowedRune(r rune) bool {
	if !unicode.IsGraphic(r) || unicode.Is(unicode.So, r) {
		return true
	}
	// 补充过滤 emoji 修饰符与组合符；它们不一定属于 So。
	return (r >= 0x1F000 && r <= 0x1FFFF) ||
		r == 0x200D ||
		r == 0x20E3 ||
		r == 0xFE0E ||
		r == 0xFE0F
}
