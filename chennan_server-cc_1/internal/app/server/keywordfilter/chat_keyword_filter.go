package keywordfilter

import (
	"context"
	"encoding/json"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

const (
	cacheTTL             = 5 * time.Second
	filterSenderCacheTTL = 30 * time.Second
)

type filterRule struct {
	keyword     string
	normalized  string
	replacement string
}

type textContent struct {
	Text string `json:"Text"`
}

type keywordCache struct {
	mu        sync.RWMutex
	rules     []filterRule
	expiresAt time.Time
}

type filterSenderCache struct {
	mu        sync.RWMutex
	userIDs   map[string]struct{}
	expiresAt time.Time
}

var chatKeywordCache keywordCache
var chatKeywordFilterSenderCache filterSenderCache

type normalizedText struct {
	runes               []rune
	originalRuneIndexes []int
}

// Invalidate 使聊天关键词缓存失效。
func Invalidate() {
	chatKeywordCache.mu.Lock()
	defer chatKeywordCache.mu.Unlock()
	chatKeywordCache.expiresAt = time.Time{}
}

// InvalidateFilterSenders 使需要关键词屏蔽的发送方缓存失效。
func InvalidateFilterSenders() {
	chatKeywordFilterSenderCache.mu.Lock()
	defer chatKeywordFilterSenderCache.mu.Unlock()
	chatKeywordFilterSenderCache.expiresAt = time.Time{}
}

// NormalizeKeyword 将关键词规范化，用于匹配与去重。
func NormalizeKeyword(keyword string) string {
	return string(normalizeRunesForMatch(keyword))
}

// ShouldFilterSender 判断发送方是否需要关键词屏蔽。
// 合规：对所有发送方生效（不再按「有无邀请码」豁免；后台配置的关键词对所有用户拦截/替换）。
func ShouldFilterSender(imUserID string) (bool, error) {
	return strings.TrimSpace(imUserID) != "", nil
}

// MaskText 将命中的关键词替换为 **。
func MaskText(text string) (string, bool, error) {
	rules, err := getEnabledRules()
	if err != nil {
		return text, false, err
	}
	masked := replaceTextByRules(text, rules)
	return masked, text != masked, nil
}

// MaskMessageElements 将消息元素中的文本消息按关键词规则替换。
func MaskMessageElements(elements []tencentim.MessageElement) ([]tencentim.MessageElement, bool, error) {
	rules, err := getEnabledRules()
	if err != nil {
		return elements, false, err
	}
	return maskMessageElementsWithRules(elements, rules)
}

// Queue workers use their own database handle, including isolated migration tests.
func MaskMessageElementsWithDB(ctx context.Context, db *gorm.DB, elements []tencentim.MessageElement) ([]tencentim.MessageElement, bool, error) {
	var rows []model.ChatKeyword
	if err := db.WithContext(ctx).Where("enabled = ?", true).Order("id DESC").Find(&rows).Error; err != nil {
		return elements, false, err
	}
	rules := make([]filterRule, 0, len(rows))
	for _, row := range rows {
		keyword := strings.TrimSpace(row.Keyword)
		normalized := NormalizeKeyword(keyword)
		if keyword == "" || normalized == "" {
			continue
		}
		replacement := strings.TrimSpace(row.Replacement)
		if replacement == "" {
			replacement = "**"
		}
		rules = append(rules, filterRule{keyword: keyword, normalized: normalized, replacement: replacement})
	}
	sort.SliceStable(rules, func(i, j int) bool {
		a, b := len([]rune(rules[i].normalized)), len([]rune(rules[j].normalized))
		if a == b {
			return rules[i].normalized < rules[j].normalized
		}
		return a > b
	})
	return maskMessageElementsWithRules(elements, rules)
}

func getEnabledRules() ([]filterRule, error) {
	now := time.Now()

	chatKeywordCache.mu.RLock()
	cachedRules := append([]filterRule(nil), chatKeywordCache.rules...)
	expiresAt := chatKeywordCache.expiresAt
	chatKeywordCache.mu.RUnlock()

	if !expiresAt.IsZero() && now.Before(expiresAt) {
		return cachedRules, nil
	}

	repo := repository.NewChatKeywordRepository()
	rows, err := repo.ListEnabled()
	if err != nil {
		if len(cachedRules) > 0 {
			return cachedRules, nil
		}
		return nil, err
	}

	rules := make([]filterRule, 0, len(rows))
	for _, row := range rows {
		keyword := strings.TrimSpace(row.Keyword)
		if keyword == "" {
			continue
		}
		normalizedKeyword := NormalizeKeyword(keyword)
		if normalizedKeyword == "" {
			continue
		}
		replacement := strings.TrimSpace(row.Replacement)
		if replacement == "" {
			replacement = "**"
		}
		rules = append(rules, filterRule{
			keyword:     keyword,
			normalized:  normalizedKeyword,
			replacement: replacement,
		})
	}

	sort.SliceStable(rules, func(i, j int) bool {
		leftLen := len([]rune(rules[i].normalized))
		rightLen := len([]rune(rules[j].normalized))
		if leftLen == rightLen {
			return rules[i].normalized < rules[j].normalized
		}
		return leftLen > rightLen
	})

	chatKeywordCache.mu.Lock()
	chatKeywordCache.rules = append([]filterRule(nil), rules...)
	chatKeywordCache.expiresAt = now.Add(cacheTTL)
	chatKeywordCache.mu.Unlock()

	return rules, nil
}

func getFilterSenderUserIDSet() (map[string]struct{}, error) {
	now := time.Now()

	chatKeywordFilterSenderCache.mu.RLock()
	cachedUserIDs := make(map[string]struct{}, len(chatKeywordFilterSenderCache.userIDs))
	for userID := range chatKeywordFilterSenderCache.userIDs {
		cachedUserIDs[userID] = struct{}{}
	}
	expiresAt := chatKeywordFilterSenderCache.expiresAt
	chatKeywordFilterSenderCache.mu.RUnlock()

	if !expiresAt.IsZero() && now.Before(expiresAt) {
		return cachedUserIDs, nil
	}

	repo := repository.NewUserRepository()
	userIDs, err := repo.ListKeywordFilterTargetIMUserIDs()
	if err != nil {
		if len(cachedUserIDs) > 0 {
			return cachedUserIDs, nil
		}
		return nil, err
	}

	nextUserIDs := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		nextUserIDs[userID] = struct{}{}
	}

	chatKeywordFilterSenderCache.mu.Lock()
	chatKeywordFilterSenderCache.userIDs = nextUserIDs
	chatKeywordFilterSenderCache.expiresAt = now.Add(filterSenderCacheTTL)
	chatKeywordFilterSenderCache.mu.Unlock()

	return nextUserIDs, nil
}

func replaceTextByRules(text string, rules []filterRule) string {
	masked := text
	for _, rule := range rules {
		normalizedKeyword := rule.normalized
		if normalizedKeyword == "" {
			normalizedKeyword = NormalizeKeyword(rule.keyword)
		}
		if normalizedKeyword == "" {
			continue
		}

		matches := findKeywordMatches(masked, normalizedKeyword)
		if len(matches) == 0 {
			continue
		}
		masked = applyKeywordMatches(masked, matches, rule.replacement)
	}
	return masked
}

func maskMessageElementsWithRules(elements []tencentim.MessageElement, rules []filterRule) ([]tencentim.MessageElement, bool, error) {
	if len(elements) == 0 || len(rules) == 0 {
		return elements, false, nil
	}

	result := make([]tencentim.MessageElement, len(elements))
	copy(result, elements)

	changed := false
	for i := range result {
		if result[i].MsgType != "TIMTextElem" {
			continue
		}

		var content textContent
		if err := json.Unmarshal(result[i].MsgContent, &content); err != nil {
			return nil, false, err
		}

		masked := replaceTextByRules(content.Text, rules)
		if masked == content.Text {
			continue
		}

		content.Text = masked
		raw, err := json.Marshal(content)
		if err != nil {
			return nil, false, err
		}
		result[i].MsgContent = raw
		changed = true
	}

	return result, changed, nil
}

type keywordMatch struct {
	start int
	end   int
}

func normalizeRunesForMatch(text string) []rune {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	result := make([]rune, 0, len([]rune(trimmed)))
	for _, r := range []rune(trimmed) {
		if shouldIgnoreMatchRune(r) {
			continue
		}
		result = append(result, r)
	}
	return result
}

func normalizeTextForMatch(text string) normalizedText {
	originalRunes := []rune(text)
	result := normalizedText{
		runes:               make([]rune, 0, len(originalRunes)),
		originalRuneIndexes: make([]int, 0, len(originalRunes)),
	}

	for index, r := range originalRunes {
		if shouldIgnoreMatchRune(r) {
			continue
		}
		result.runes = append(result.runes, r)
		result.originalRuneIndexes = append(result.originalRuneIndexes, index)
	}

	return result
}

func shouldIgnoreMatchRune(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}

	switch r {
	case '\u200B', '\u200C', '\u200D', '\u2060', '\uFEFF':
		return true
	default:
		return false
	}
}

func findKeywordMatches(text, normalizedKeyword string) []keywordMatch {
	keywordRunes := []rune(normalizedKeyword)
	if len(keywordRunes) == 0 {
		return nil
	}

	normalized := normalizeTextForMatch(text)
	if len(normalized.runes) < len(keywordRunes) {
		return nil
	}

	matches := make([]keywordMatch, 0)
	for i := 0; i <= len(normalized.runes)-len(keywordRunes); {
		matched := true
		for j := 0; j < len(keywordRunes); j++ {
			if normalized.runes[i+j] != keywordRunes[j] {
				matched = false
				break
			}
		}
		if !matched {
			i++
			continue
		}

		start := normalized.originalRuneIndexes[i]
		end := normalized.originalRuneIndexes[i+len(keywordRunes)-1] + 1
		matches = append(matches, keywordMatch{
			start: start,
			end:   end,
		})
		i += len(keywordRunes)
	}

	return matches
}

func applyKeywordMatches(text string, matches []keywordMatch, replacement string) string {
	if len(matches) == 0 {
		return text
	}

	originalRunes := []rune(text)
	replacementRunes := []rune(replacement)
	result := make([]rune, 0, len(originalRunes))
	prev := 0

	for _, match := range matches {
		if match.start < prev || match.start > len(originalRunes) || match.end > len(originalRunes) {
			continue
		}
		result = append(result, originalRunes[prev:match.start]...)
		result = append(result, replacementRunes...)
		prev = match.end
	}
	result = append(result, originalRunes[prev:]...)
	return string(result)
}
