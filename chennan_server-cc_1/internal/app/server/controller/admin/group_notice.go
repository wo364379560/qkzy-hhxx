package controller_admin

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

type GroupNoticeResult struct {
	Pinned   bool   `json:"pinned"`
	GroupID  uint   `json:"groupID"`
	Content  string `json:"content"`
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	HasValue bool   `json:"hasValue"`
}

func GetAdminGroupNotice(groupID, adminID uint) (*GroupNoticeResult, error) {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return nil, err
	}

	notice, err := repository.NewGroupNoticeRepository().FindByGroupID(group.ID)
	if err == nil {
		return &GroupNoticeResult{
			GroupID:  group.ID,
			Content:  notice.Content,
			Pinned:   notice.Pinned,
			Summary:  notice.Summary,
			Source:   "backend",
			HasValue: strings.TrimSpace(notice.Content) != "",
		}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &GroupNoticeResult{
		GroupID:  group.ID,
		Content:  "",
		Summary:  "",
		Source:   "fallback",
		HasValue: false,
	}, nil
}

func SaveAdminGroupNotice(groupID, adminID uint, content string) (*GroupNoticeResult, error) {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if Utf8ByteLength(content) > 5000 {
		return nil, errors.New("群公告长度不能超过5000字节")
	}

	summary := BuildNoticeSummary(content, 240)
	notice := &model.GroupNotice{
		GroupID:          group.ID,
		Content:          content,
		Summary:          summary,
		UpdatedByAdminID: &adminID,
	}
	if err := repository.NewGroupNoticeRepository().Upsert(notice); err != nil {
		return nil, err
	}
	return &GroupNoticeResult{
		GroupID:  group.ID,
		Content:  content,
		Pinned:   notice.Pinned,
		Summary:  summary,
		Source:   "backend",
		HasValue: content != "",
	}, nil
}

func Utf8ByteLength(value string) int {
	return len([]byte(value))
}

func BuildNoticeSummary(value string, maxBytes int) string {
	value = strings.Join(strings.Fields(value), " ")
	if Utf8ByteLength(value) <= maxBytes {
		return value
	}
	if maxBytes <= 0 {
		return ""
	}

	suffix := ""
	suffixBytes := Utf8ByteLength("...")
	if maxBytes > suffixBytes {
		suffix = "..."
		maxBytes -= suffixBytes
	}

	var builder strings.Builder
	currentBytes := 0
	for _, r := range value {
		runeBytes := len([]byte(string(r)))
		if currentBytes+runeBytes > maxBytes {
			break
		}
		builder.WriteRune(r)
		currentBytes += runeBytes
	}
	return builder.String() + suffix
}
