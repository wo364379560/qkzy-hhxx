package repository

import (
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

type GroupNoticeRepository struct {
	db *gorm.DB
}

func NewGroupNoticeRepository() *GroupNoticeRepository {
	return &GroupNoticeRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewGroupNoticeRepositoryWithDB(db *gorm.DB) *GroupNoticeRepository {
	return &GroupNoticeRepository{db: db}
}

func (r *GroupNoticeRepository) FindByGroupID(groupID uint) (*model.GroupNotice, error) {
	var notice model.GroupNotice
	if err := r.db.Where("group_id = ?", groupID).First(&notice).Error; err != nil {
		return nil, err
	}
	return &notice, nil
}

func (r *GroupNoticeRepository) Upsert(notice *model.GroupNotice, pinned ...*bool) error {
	fields := map[string]any{
		"content": notice.Content, "summary": notice.Summary,
		"updated_by_admin_id": notice.UpdatedByAdminID, "updated_by_user_id": notice.UpdatedByUserID,
	}
	if len(pinned) > 0 && pinned[0] != nil {
		fields["pinned"] = *pinned[0]
	}
	return r.db.Where("group_id = ?", notice.GroupID).Assign(fields).FirstOrCreate(notice).Error
}

// SetPinned changes only the flag; a concurrent content edit is never replaced.
func (r *GroupNoticeRepository) SetPinned(groupID uint, pinned bool) error {
	return r.db.Model(&model.GroupNotice{}).Where("group_id = ?", groupID).Update("pinned", pinned).Error
}
