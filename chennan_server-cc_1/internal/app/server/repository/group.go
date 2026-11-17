package repository

import (
	"context"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// GroupRepository 处理 groups 表的数据库操作。
type GroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository() *GroupRepository {
	return &GroupRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *GroupRepository) FindByID(id uint) (*model.Group, error) {
	var group model.Group
	if err := r.db.First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) Create(group *model.Group) error {
	if group.OwnerBinding == "independent" {
		return r.db.Create(group).Error
	}
	return r.SaveMirror(context.Background(), group)
}

func (r *GroupRepository) Delete(id uint) error {
	return r.db.Delete(&model.Group{}, id).Error
}

func (r *GroupRepository) Update(group *model.Group) error {
	if group == nil || group.ID == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.Model(&model.Group{}).Where("id = ?", group.ID).Select("*").
		Omit("id", "owner_id", "created_by", "owner_revision", "owner_binding", "mirror_revision").Updates(group).Error
}

func (r *GroupRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.Group{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateOwnerByCreatedBy 将某用户创建的有效群归属同步到新 owner（转移归属时用）。
func (r *GroupRepository) UpdateOwnerByCreatedBy(createdByUserID, newOwnerID uint) error {
	return ownershipTransaction(context.Background(), r.db, func(tx *gorm.DB) error {
		s, err := lockOwnerRows(tx, []uint{createdByUserID}, nil)
		if err != nil {
			return err
		}
		u := s.Users[createdByUserID]
		if u.OwnerID != newOwnerID {
			return OwnershipError{"revision_conflict"}
		}
		_, err = s.move(u, u.OwnerID, u.InviteCode)
		return err
	})
}

func (r *GroupRepository) UpdateSlowMode(id uint, enabled bool, seconds int, version int64) error {
	return r.db.Model(&model.Group{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"slow_mode_enabled": enabled,
			"slow_mode_seconds": seconds,
			"slow_mode_version": version,
		}).Error
}

// ListByOwner 不分页（供内部使用）。
func (r *GroupRepository) ListByOwner(ownerID uint) ([]model.Group, error) {
	var groups []model.Group
	err := r.db.Where("owner_id = ?", ownerID).Find(&groups).Error
	return groups, err
}

// ListPaged 分页查询群组列表。ownerID=0 时使用 scopeOwnerIDs 过滤；scopeOwnerIDs 为空表示不限制归属。
func (r *GroupRepository) ListPaged(page, pageSize int, scopeOwnerIDs []uint, ownerID uint, keyword, status, groupType string) ([]model.Group, int64, error) {
	q := r.db.Model(&model.Group{})
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", scopeOwnerIDs)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR im_group_id LIKE ?", like, like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if groupType != "" {
		q = q.Where("group_type = ?", groupType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var groups []model.Group
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

// Count 统计群组总数。
func (r *GroupRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Group{}).Count(&count).Error
	return count, err
}

// CountByOwners 统计指定 owner 集合下的群组数量。
func (r *GroupRepository) CountByOwners(ownerIDs []uint) (int64, error) {
	if len(ownerIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.Model(&model.Group{}).Where("owner_id IN ?", ownerIDs).Count(&count).Error
	return count, err
}

// CountActiveByOwners 统计指定 owner 集合下有效群组数量。
func (r *GroupRepository) CountActiveByOwners(ownerIDs []uint) (int64, error) {
	if len(ownerIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.Model(&model.Group{}).
		Where("owner_id IN ? AND status = 'active'", ownerIDs).
		Count(&count).Error
	return count, err
}

// FindAll 查询所有群组（不分页）。
func (r *GroupRepository) FindAll() ([]model.Group, error) {
	var groups []model.Group
	err := r.db.Find(&groups).Error
	return groups, err
}

// FindAllByOwners 查询指定 owner 集合下的所有群组（不分页）。
func (r *GroupRepository) FindAllByOwners(ownerIDs []uint) ([]model.Group, error) {
	if len(ownerIDs) == 0 {
		return []model.Group{}, nil
	}
	var groups []model.Group
	err := r.db.Where("owner_id IN ?", ownerIDs).Find(&groups).Error
	return groups, err
}

// FindByIMGroupID 根据 IM 群组 ID 查询。
func (r *GroupRepository) FindByIMGroupID(imGroupID string) (*model.Group, error) {
	var group model.Group
	if err := r.db.Where("im_group_id = ?", imGroupID).First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// FindActiveByIMGroupID 根据 IM 群组 ID 查询有效群组。
func (r *GroupRepository) FindActiveByIMGroupID(imGroupID string) (*model.Group, error) {
	var group model.Group
	if err := r.db.
		Where("im_group_id = ? AND status = 'active'", imGroupID).
		First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// FindActiveByOwnerAndIMGroupID 按归属和 IM 群组 ID 查询有效群组。
func (r *GroupRepository) FindActiveByOwnerAndIMGroupID(ownerID uint, imGroupID string) (*model.Group, error) {
	var group model.Group
	if err := r.db.
		Where("owner_id = ? AND im_group_id = ? AND status = 'active'", ownerID, imGroupID).
		First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// SearchForIMAdmin 在后台按群组名称搜索群组候选。
// ownerID=0 时使用 scopeOwnerIDs 过滤；scopeOwnerIDs 为空表示全量搜索。
func (r *GroupRepository) SearchForIMAdmin(scopeOwnerIDs []uint, ownerID uint, keyword string, limit int) ([]model.Group, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	q := r.db.Model(&model.Group{}).Where("im_group_id <> ''")
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", scopeOwnerIDs)
	}

	var groups []model.Group
	if strings.TrimSpace(keyword) == "" {
		err := q.
			Order("name ASC, id DESC").
			Limit(limit).
			Find(&groups).Error
		return groups, err
	}

	like := "%" + keyword + "%"
	err := q.
		Where("name LIKE ?", like).
		Order("name ASC, id DESC").
		Limit(limit).
		Find(&groups).Error
	return groups, err
}

// SearchByOwner 在同归属下按群组名称或 IM GroupID 模糊搜索群组。
// 用于 Flutter App 端的加群搜索功能。
func (r *GroupRepository) SearchByOwner(ownerID uint, keyword string) ([]model.Group, error) {
	like := "%" + keyword + "%"
	var groups []model.Group
	err := r.db.
		Where("owner_id = ? AND status = 'active'", ownerID).
		Where("name LIKE ? OR im_group_id LIKE ?", like, like).
		Limit(20).
		Find(&groups).Error
	return groups, err
}
