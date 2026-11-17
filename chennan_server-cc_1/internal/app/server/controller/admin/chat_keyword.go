package controller_admin

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

// ChatKeywordResult 表示关键词规则响应。
type ChatKeywordResult struct {
	ID          uint   `json:"id"`
	Keyword     string `json:"keyword"`
	Replacement string `json:"replacement"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ListChatKeywordsReq 表示关键词列表查询参数。
type ListChatKeywordsReq struct {
	Page     int
	PageSize int
	Keyword  string
}

// SaveChatKeywordReq 表示创建或更新关键词的参数。
type SaveChatKeywordReq struct {
	Keyword     string `json:"keyword"`
	Replacement string `json:"replacement"`
	Enabled     bool   `json:"enabled"`
}

// BatchImportChatKeywordsReq 表示批量导入关键词的参数。
type BatchImportChatKeywordsReq struct {
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}

// ChatKeywordImportItem 表示批量导入单行结果。
type ChatKeywordImportItem struct {
	Line        int    `json:"line"`
	Keyword     string `json:"keyword"`
	Replacement string `json:"replacement"`
	Action      string `json:"action"`
	Message     string `json:"message"`
}

// BatchImportChatKeywordsResult 表示批量导入结果。
type BatchImportChatKeywordsResult struct {
	Created int                     `json:"created"`
	Updated int                     `json:"updated"`
	Skipped int                     `json:"skipped"`
	Results []ChatKeywordImportItem `json:"results"`
}

func toChatKeywordResult(row *model.ChatKeyword) ChatKeywordResult {
	return ChatKeywordResult{
		ID:          row.ID,
		Keyword:     row.Keyword,
		Replacement: row.Replacement,
		Enabled:     row.Enabled,
		CreatedAt:   row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   row.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListChatKeywords 分页列出聊天关键词规则。
func ListChatKeywords(actorID uint, req ListChatKeywordsReq) (PageResult[ChatKeywordResult], error) {
	if err := ensureSuperAdmin(actorID); err != nil {
		return PageResult[ChatKeywordResult]{}, err
	}

	repo := repository.NewChatKeywordRepository()
	rows, total, err := repo.ListPaged(req.Page, req.PageSize, req.Keyword)
	if err != nil {
		return PageResult[ChatKeywordResult]{}, err
	}

	list := make([]ChatKeywordResult, 0, len(rows))
	for i := range rows {
		list = append(list, toChatKeywordResult(&rows[i]))
	}
	return PageResult[ChatKeywordResult]{List: list, Total: total}, nil
}

// CreateChatKeyword 创建聊天关键词规则。
func CreateChatKeyword(actorID uint, req SaveChatKeywordReq) (*ChatKeywordResult, error) {
	if err := ensureSuperAdmin(actorID); err != nil {
		return nil, err
	}

	keyword, err := normalizeChatKeyword(req.Keyword)
	if err != nil {
		return nil, err
	}
	replacement, err := normalizeChatKeywordReplacement(req.Replacement)
	if err != nil {
		return nil, err
	}

	repo := repository.NewChatKeywordRepository()
	if _, err := findChatKeywordByNormalizedKeyword(repo, keyword); err == nil {
		return nil, errors.New("关键词已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	row := &model.ChatKeyword{
		Keyword:     keyword,
		Replacement: replacement,
		Enabled:     req.Enabled,
		CreatedBy:   &actorID,
		UpdatedBy:   &actorID,
	}
	if err := repo.Create(row); err != nil {
		return nil, err
	}

	keywordfilter.Invalidate()
	result := toChatKeywordResult(row)
	return &result, nil
}

// UpdateChatKeyword 更新聊天关键词规则。
func UpdateChatKeyword(actorID, id uint, req SaveChatKeywordReq) (*ChatKeywordResult, error) {
	if err := ensureSuperAdmin(actorID); err != nil {
		return nil, err
	}

	keyword, err := normalizeChatKeyword(req.Keyword)
	if err != nil {
		return nil, err
	}
	replacement, err := normalizeChatKeywordReplacement(req.Replacement)
	if err != nil {
		return nil, err
	}

	repo := repository.NewChatKeywordRepository()
	row, err := repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("关键词不存在")
		}
		return nil, err
	}

	if existing, err := findChatKeywordByNormalizedKeyword(repo, keyword); err == nil && existing.ID != row.ID {
		return nil, errors.New("关键词已存在")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	updatedBy := actorID
	fields := map[string]any{
		"keyword":     keyword,
		"replacement": replacement,
		"enabled":     req.Enabled,
		"updated_by":  &updatedBy,
	}
	if err := repo.Update(id, fields); err != nil {
		return nil, err
	}

	keywordfilter.Invalidate()
	row, err = repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	result := toChatKeywordResult(row)
	return &result, nil
}

// DeleteChatKeyword 删除聊天关键词规则。
func DeleteChatKeyword(actorID, id uint) error {
	if err := ensureSuperAdmin(actorID); err != nil {
		return err
	}

	repo := repository.NewChatKeywordRepository()
	if _, err := repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("关键词不存在")
		}
		return err
	}

	if err := repo.Delete(id); err != nil {
		return err
	}

	keywordfilter.Invalidate()
	return nil
}

// BatchImportChatKeywords 批量导入聊天关键词规则。
func BatchImportChatKeywords(actorID uint, req BatchImportChatKeywordsReq) (*BatchImportChatKeywordsResult, error) {
	if err := ensureSuperAdmin(actorID); err != nil {
		return nil, err
	}

	content := strings.ReplaceAll(req.Content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil, errors.New("导入内容不能为空")
	}

	repo := repository.NewChatKeywordRepository()
	result := &BatchImportChatKeywordsResult{
		Results: make([]ChatKeywordImportItem, 0, len(lines)),
	}

	updatedBy := actorID
	changed := false

	for idx, rawLine := range lines {
		lineNo := idx + 1
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		keyword, replacement, err := parseImportLine(line)
		if err != nil {
			result.Skipped++
			result.Results = append(result.Results, ChatKeywordImportItem{
				Line:    lineNo,
				Action:  "error",
				Message: err.Error(),
			})
			continue
		}

		existing, err := findChatKeywordByNormalizedKeyword(repo, keyword)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			row := &model.ChatKeyword{
				Keyword:     keyword,
				Replacement: replacement,
				Enabled:     req.Enabled,
				CreatedBy:   &actorID,
				UpdatedBy:   &actorID,
			}
			if err := repo.Create(row); err != nil {
				result.Skipped++
				result.Results = append(result.Results, ChatKeywordImportItem{
					Line:        lineNo,
					Keyword:     keyword,
					Replacement: replacement,
					Action:      "error",
					Message:     err.Error(),
				})
				continue
			}

			result.Created++
			changed = true
			result.Results = append(result.Results, ChatKeywordImportItem{
				Line:        lineNo,
				Keyword:     keyword,
				Replacement: replacement,
				Action:      "created",
				Message:     "已创建",
			})
			continue
		}

		if existing.Replacement == replacement && existing.Enabled == req.Enabled {
			result.Skipped++
			result.Results = append(result.Results, ChatKeywordImportItem{
				Line:        lineNo,
				Keyword:     keyword,
				Replacement: replacement,
				Action:      "skipped",
				Message:     "无变更，已跳过",
			})
			continue
		}

		if err := repo.Update(existing.ID, map[string]any{
			"replacement": replacement,
			"enabled":     req.Enabled,
			"updated_by":  &updatedBy,
		}); err != nil {
			result.Skipped++
			result.Results = append(result.Results, ChatKeywordImportItem{
				Line:        lineNo,
				Keyword:     keyword,
				Replacement: replacement,
				Action:      "error",
				Message:     err.Error(),
			})
			continue
		}

		result.Updated++
		changed = true
		result.Results = append(result.Results, ChatKeywordImportItem{
			Line:        lineNo,
			Keyword:     keyword,
			Replacement: replacement,
			Action:      "updated",
			Message:     "已更新",
		})
	}

	if changed {
		keywordfilter.Invalidate()
	}

	return result, nil
}

func ensureSuperAdmin(actorID uint) error {
	actor, err := BuildActorContext(actorID)
	if err != nil {
		return err
	}
	if !IsSuperAdminRole(actor.Role) {
		return errors.New("仅超级管理员可操作")
	}
	return nil
}

func normalizeChatKeyword(keyword string) (string, error) {
	keyword = keywordfilter.NormalizeKeyword(keyword)
	if keyword == "" {
		return "", errors.New("关键词不能为空")
	}
	if len([]rune(keyword)) > 64 {
		return "", errors.New("关键词长度不能超过 64 个字符")
	}
	return keyword, nil
}

func normalizeChatKeywordReplacement(replacement string) (string, error) {
	replacement = strings.TrimSpace(replacement)
	if replacement == "" {
		return "**", nil
	}
	if len([]rune(replacement)) > 64 {
		return "", errors.New("替换文本长度不能超过 64 个字符")
	}
	return replacement, nil
}

func parseImportLine(line string) (string, string, error) {
	idx := strings.IndexAny(line, ",，\t")
	if idx < 0 {
		keyword, err := normalizeChatKeyword(line)
		if err != nil {
			return "", "", err
		}
		return keyword, "**", nil
	}

	keyword, err := normalizeChatKeyword(line[:idx])
	if err != nil {
		return "", "", err
	}

	replacement, err := normalizeChatKeywordReplacement(line[idx+1:])
	if err != nil {
		return "", "", err
	}
	return keyword, replacement, nil
}

func findChatKeywordByNormalizedKeyword(repo *repository.ChatKeywordRepository, normalizedKeyword string) (*model.ChatKeyword, error) {
	rows, err := repo.ListAll()
	if err != nil {
		return nil, err
	}

	for i := range rows {
		if keywordfilter.NormalizeKeyword(rows[i].Keyword) == normalizedKeyword {
			return &rows[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}
