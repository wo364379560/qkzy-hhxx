package tencentim

import (
	"os"
	"strings"
)

func forwardPushSetting(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

// Defaults are the Rui WEB's existing values, never the reference application's.
// https://cloud.tencent.com/document/product/269/128209
func NewForwardOfflinePushInfo(title, desc, ext, sender string) *OfflinePushInfo {
	return &OfflinePushInfo{Title: title, Desc: desc, Ext: ext, PushFlag: 0, AndroidInfo: &OfflinePushAndroidInfo{
		XiaoMiChannelID:     forwardPushSetting("FORWARD_XIAOMI_CHANNEL_ID", "157799"),
		XiaoMiTemplateID:    forwardPushSetting("FORWARD_XIAOMI_TEMPLATE_ID", "M12762"),
		XiaoMiTemplateParam: map[string]string{"keywords1": sanitizeXiaoMiTemplateValue(sender, title), "keywords2": sanitizeXiaoMiTemplateValue(desc, "消息")},
		OPPOChannelID:       forwardPushSetting("FORWARD_OPPO_CHANNEL_ID", "push_oplus_category_service"), OPPOCategory: "IM",
		OPPOPrivateMsgTemplateID:   forwardPushSetting("FORWARD_OPPO_TEMPLATE_ID", "6a8408344fae550145d04b8f"),
		OPPOPrivateTitleParameters: map[string]string{"senderName": sender}, OPPOPrivateContentParameters: map[string]string{"text": desc},
	}}
}
