package keywordfilter

import (
	"encoding/json"
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

func TestReplaceTextByKeywords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		rules []filterRule
		want  string
	}{
		{
			name: "replace single keyword",
			text: "2",
			rules: []filterRule{
				{keyword: "2", replacement: "**"},
			},
			want: "**",
		},
		{
			name: "replace longer keyword first",
			text: "12 2 312",
			rules: []filterRule{
				{keyword: "12", replacement: "##"},
				{keyword: "2", replacement: "**"},
			},
			want: "## ** 3##",
		},
		{
			name: "no keyword matched",
			text: "hello",
			rules: []filterRule{
				{keyword: "world", replacement: "--"},
			},
			want: "hello",
		},
		{
			name: "replace chinese keyword in sentence",
			text: "这是一条包含屏蔽词的中文句子",
			rules: []filterRule{
				{keyword: "屏蔽词", replacement: "**"},
			},
			want: "这是一条包含**的中文句子",
		},
		{
			name: "replace chinese keyword only",
			text: "敏感词",
			rules: []filterRule{
				{keyword: "敏感词", normalized: NormalizeKeyword("敏感词"), replacement: "[已屏蔽]"},
			},
			want: "[已屏蔽]",
		},
		{
			name: "replace chinese keyword with spaces between runes",
			text: "这是刷 单行为",
			rules: []filterRule{
				{keyword: "刷单", normalized: NormalizeKeyword("刷单"), replacement: "**"},
			},
			want: "这是**行为",
		},
		{
			name: "replace chinese keyword with full width spaces",
			text: "这是刷　单行为",
			rules: []filterRule{
				{keyword: "刷单", normalized: NormalizeKeyword("刷单"), replacement: "**"},
			},
			want: "这是**行为",
		},
		{
			name: "replace chinese keyword with zero width chars",
			text: "这是刷\u200b单行为",
			rules: []filterRule{
				{keyword: "刷单", normalized: NormalizeKeyword("刷单"), replacement: "**"},
			},
			want: "这是**行为",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := replaceTextByRules(tt.text, tt.rules)
			if got != tt.want {
				t.Fatalf("replaceTextByRules() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaskMessageElementsWithRules(t *testing.T) {
	t.Parallel()

	textRaw, err := json.Marshal(map[string]string{"Text": "客户发送 2"})
	if err != nil {
		t.Fatalf("marshal text content: %v", err)
	}

	imageRaw, err := json.Marshal(map[string]string{"UUID": "img-1"})
	if err != nil {
		t.Fatalf("marshal image content: %v", err)
	}

	elements := []tencentim.MessageElement{
		{
			MsgType:    "TIMTextElem",
			MsgContent: textRaw,
		},
		{
			MsgType:    "TIMImageElem",
			MsgContent: imageRaw,
		},
	}

	got, changed, err := maskMessageElementsWithRules(elements, []filterRule{
		{keyword: "2", normalized: NormalizeKeyword("2"), replacement: "[屏蔽]"},
	})
	if err != nil {
		t.Fatalf("maskMessageElementsWithRules() error = %v", err)
	}
	if !changed {
		t.Fatalf("maskMessageElementsWithRules() changed = false, want true")
	}
	if got[1].MsgType != "TIMImageElem" {
		t.Fatalf("non-text element should stay unchanged")
	}

	var textContent map[string]string
	if err := json.Unmarshal(got[0].MsgContent, &textContent); err != nil {
		t.Fatalf("unmarshal masked text content: %v", err)
	}
	if textContent["Text"] != "客户发送 [屏蔽]" {
		t.Fatalf("masked text = %q, want %q", textContent["Text"], "客户发送 [屏蔽]")
	}
}

func TestReplaceTextByRules_OverlappingKeywords(t *testing.T) {
	t.Parallel()

	got := replaceTextByRules("广告广告词", []filterRule{
		{keyword: "广告词", normalized: NormalizeKeyword("广告词"), replacement: "[组合]"},
		{keyword: "广告", normalized: NormalizeKeyword("广告"), replacement: "[单词]"},
	})

	if got != "[单词][组合]" {
		t.Fatalf("replaceTextByRules() = %q, want %q", got, "[单词][组合]")
	}
}

func TestNormalizeKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim spaces between chinese runes",
			input: " 刷 单 ",
			want:  "刷单",
		},
		{
			name:  "remove tabs and new lines",
			input: "刷\t\n单",
			want:  "刷单",
		},
		{
			name:  "remove zero width rune",
			input: "刷\u200b单",
			want:  "刷单",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NormalizeKeyword(tt.input)
			if got != tt.want {
				t.Fatalf("NormalizeKeyword() = %q, want %q", got, tt.want)
			}
		})
	}
}
