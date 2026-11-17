package tencentim

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"
)

func TestNewC2CTextOfflinePushInfoXiaoMiTemplate(t *testing.T) {
	push := NewC2CTextOfflinePushInfo(
		"安全助手",
		"检测到账号登录",
		`{"entity":{"conversationID":"c2c_security_assistant"}}`,
		"安全助手🛡️",
		"检测到\n账号登录",
	)

	if push.AndroidInfo == nil {
		t.Fatal("AndroidInfo must not be nil")
	}
	if push.AndroidInfo.XiaoMiChannelID != "157799" {
		t.Fatalf("unexpected channel ID: %q", push.AndroidInfo.XiaoMiChannelID)
	}
	if push.AndroidInfo.XiaoMiTemplateID != "M10291" {
		t.Fatalf("unexpected template ID: %q", push.AndroidInfo.XiaoMiTemplateID)
	}
	want := map[string]string{
		"keywords1": "安全助手",
		"keywords2": "安全助手",
		"keywords3": "检测到 账号登录",
	}
	for key, expected := range want {
		if actual := push.AndroidInfo.XiaoMiTemplateParam[key]; actual != expected {
			t.Errorf("%s = %q, want %q", key, actual, expected)
		}
	}
	if push.Title != "安全助手" || push.Desc != "检测到账号登录" {
		t.Fatalf("Title/Desc must stay unsanitized for other vendors: %#v", push)
	}

	encoded, err := json.Marshal(push)
	if err != nil {
		t.Fatalf("marshal push: %v", err)
	}
	if !strings.Contains(string(encoded), `"XiaoMiTemplateId":"M10291"`) ||
		!strings.Contains(string(encoded), `"XiaoMiTemplateParam":{`) {
		t.Fatalf("REST field names are incorrect: %s", encoded)
	}
}

func TestNewGroupMessageOfflinePushInfoUsesM10291(t *testing.T) {
	longSummary := strings.Repeat("长", 140) + "©😀"
	push := NewGroupMessageOfflinePushInfo(
		"测试群",
		longSummary,
		"{}",
		"小明😀",
		"测试群©",
		longSummary,
	)

	android := push.AndroidInfo
	if android == nil || android.XiaoMiTemplateID != "M10291" {
		t.Fatalf("unexpected AndroidInfo: %#v", android)
	}
	if got := android.XiaoMiTemplateParam["keywords1"]; got != "测试群" {
		t.Fatalf("keywords1 = %q, want group name", got)
	}
	if got := android.XiaoMiTemplateParam["keywords2"]; got != "小明" {
		t.Fatalf("keywords2 = %q, want sender", got)
	}
	if _, ok := android.XiaoMiTemplateParam["keywords4"]; ok {
		t.Fatal("M10291 must not include keywords4")
	}
	for key, value := range android.XiaoMiTemplateParam {
		if value == "" {
			t.Fatalf("%s must not be empty", key)
		}
		if count := len([]rune(value)); count > xiaoMiTemplateValueLimit {
			t.Fatalf("%s has %d runes", key, count)
		}
		for _, r := range value {
			if unicode.Is(unicode.So, r) {
				t.Fatalf("%s contains Unicode So rune %U", key, r)
			}
		}
	}
	if got := len([]rune(android.XiaoMiTemplateParam["keywords3"])); got != 128 {
		t.Fatalf("keywords3 rune count = %d, want 128", got)
	}
	if push.Title != "测试群" {
		t.Fatalf("Title should remain raw: %q", push.Title)
	}
}
