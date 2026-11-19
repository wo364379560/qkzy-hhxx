package repository

import (
	"errors"
	"sort"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	GroupRoleOwner         = "owner"
	GroupRoleTencentAdmin  = "tencent_admin"
	GroupRoleBusinessAdmin = "business_admin"

	GroupAdminSourceTencentRoleSync = "tencent_role_sync"
)

// GroupBusinessAdminRepository handles local business administrator grants.
type GroupBusinessAdminRepository struct {
	db *gorm.DB
}

func NewGroupBusinessAdminRepository() *GroupBusinessAdminRepository {
	return &GroupBusinessAdminRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewGroupBusinessAdminRepositoryWithDB(db *gorm.DB) *GroupBusinessAdminRepository {
	return &GroupBusinessAdminRepository{db: db}
}

func (r *GroupBusinessAdminRepository) ListActiveByGroup(groupID string) ([]model.GroupBusinessAdmin, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return []model.GroupBusinessAdmin{}, nil
	}
	var rows []model.GroupBusinessAdmin
	err := r.db.
		Where("group_id = ? AND status = ?", groupID, "active").
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *GroupBusinessAdminRepository) FindActive(groupID, userID string) (*model.GroupBusinessAdmin, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	var row model.GroupBusinessAdmin
	if err := r.db.
		Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, "active").
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GroupBusinessAdminRepository) IsActive(groupID, userID string) (bool, error) {
	return r.IsActiveRole(groupID, userID, GroupRoleBusinessAdmin)
}

func (r *GroupBusinessAdminRepository) IsActiveRole(groupID, userID string, roles ...string) (bool, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return false, nil
	}
	normalizedRoles := make([]string, 0, len(roles))
	for _, role := range roles {
		normalizedRoles = append(normalizedRoles, normalizeGroupAdminRole(role))
	}
	var count int64
	query := r.db.Model(&model.GroupBusinessAdmin{}).
		Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, "active")
	if len(normalizedRoles) > 0 {
		query = query.Where("role IN ?", normalizedRoles)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *GroupBusinessAdminRepository) HasSyncedTencentOwner(groupID string) (bool, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return false, nil
	}
	var count int64
	err := r.db.Model(&model.GroupBusinessAdmin{}).
		Where("group_id = ? AND role = ? AND source = ? AND status = ?", groupID, GroupRoleOwner, GroupAdminSourceTencentRoleSync, "active").
		Count(&count).Error
	return count > 0, err
}

func (r *GroupBusinessAdminRepository) UpsertActive(groupID, userID, createdBy, source string) (*model.GroupBusinessAdmin, error) {
	return r.UpsertActiveRole(groupID, userID, createdBy, source, GroupRoleBusinessAdmin)
}

func (r *GroupBusinessAdminRepository) UpsertActiveRole(groupID, userID, createdBy, source, role string) (*model.GroupBusinessAdmin, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	createdBy = strings.TrimSpace(createdBy)
	source = strings.TrimSpace(source)
	if source == "" {
		source = "tencent_overflow"
	}
	role = normalizeGroupAdminRole(role)

	row := model.GroupBusinessAdmin{
		GroupID:   groupID,
		UserID:    userID,
		Role:      role,
		Source:    source,
		Status:    "active",
		CreatedBy: createdBy,
	}
	// The API and Tencent callback may confirm the same grant concurrently.
	// Use the (group_id, user_id) unique key rather than SELECT-then-INSERT.
	if err := r.db.Clauses(clause.OnConflict{DoUpdates: clause.Assignments(map[string]any{
		"role": role, "source": source, "status": "active", "updated_at": gorm.Expr("NOW(3)"),
		"created_by": gorm.Expr("IF(created_by = '', VALUES(created_by), created_by)"),
	})}).Create(&row).Error; err != nil {
		return nil, err
	}
	if err := r.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// SyncTencentRoles mirrors a freshly read Tencent role snapshot atomically.
// Partial snapshots only touch the requested users. A full snapshot also retires
// departed members. An active business grant survives Tencent's Member role.
func (r *GroupBusinessAdminRepository) SyncTencentRoles(groupID string, roles map[string]string, updatedBy string, fullSnapshot bool) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("group_id不能为空")
	}
	userIDs := make([]string, 0, len(roles))
	memberIDs := make([]string, 0, len(roles))
	managerIDs := make([]string, 0)
	for userID, role := range roles {
		if userID == "" || userID != strings.TrimSpace(userID) {
			return errors.New("腾讯群成员身份数据不完整")
		}
		switch role {
		case "Owner", "Admin", "Member", "NotMember":
		default:
			return errors.New("腾讯群成员身份数据不完整")
		}
		userIDs = append(userIDs, userID)
		if role != "NotMember" {
			memberIDs = append(memberIDs, userID)
		}
		if role == "Owner" || role == "Admin" {
			managerIDs = append(managerIDs, userID)
		}
	}
	sort.Strings(userIDs)
	return r.db.Transaction(func(tx *gorm.DB) error {
		repo := &GroupBusinessAdminRepository{db: tx}
		for _, userID := range userIDs {
			switch roles[userID] {
			case "Owner", "Admin":
				role, source := GroupRoleTencentAdmin, "tencent"
				if roles[userID] == "Owner" {
					role, source = GroupRoleOwner, GroupAdminSourceTencentRoleSync
				}
				if _, err := repo.UpsertActiveRole(groupID, userID, updatedBy, source, role); err != nil {
					return err
				}
			case "Member":
				if fullSnapshot {
					continue // Clear obsolete native grants in one group-scoped update below.
				}
				if err := tx.Model(&model.GroupBusinessAdmin{}).
					Where("group_id = ? AND user_id = ? AND status = ? AND role IN ?", groupID, userID, "active", []string{GroupRoleOwner, GroupRoleTencentAdmin}).
					Updates(map[string]any{"status": "deleted", "updated_at": gorm.Expr("NOW(3)")}).Error; err != nil {
					return err
				}
			case "NotMember":
				if fullSnapshot {
					continue
				}
				if err := repo.MarkDeleted(groupID, userID); err != nil {
					return err
				}
			}
		}
		if fullSnapshot {
			nativeGrants := tx.Model(&model.GroupBusinessAdmin{}).
				Where("group_id = ? AND status = ? AND role IN ?", groupID, "active", []string{GroupRoleOwner, GroupRoleTencentAdmin})
			if len(managerIDs) > 0 {
				nativeGrants = nativeGrants.Where("user_id NOT IN ?", managerIDs)
			}
			if err := nativeGrants.Updates(map[string]any{"status": "deleted", "updated_at": gorm.Expr("NOW(3)")}).Error; err != nil {
				return err
			}
			query := tx.Model(&model.GroupBusinessAdmin{}).Where("group_id = ? AND status = ?", groupID, "active")
			if len(memberIDs) > 0 {
				query = query.Where("user_id NOT IN ?", memberIDs)
			}
			return query.Updates(map[string]any{"status": "deleted", "updated_at": gorm.Expr("NOW(3)")}).Error
		}
		return nil
	})
}

func normalizeGroupAdminRole(role string) string {
	switch strings.TrimSpace(role) {
	case GroupRoleOwner, GroupRoleTencentAdmin, GroupRoleBusinessAdmin:
		return strings.TrimSpace(role)
	default:
		return GroupRoleBusinessAdmin
	}
}

func (r *GroupBusinessAdminRepository) MarkDeleted(groupID, userID string) error {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return nil
	}
	return r.db.Model(&model.GroupBusinessAdmin{}).
		Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, "active").
		Update("status", "deleted").Error
}

func (r *GroupBusinessAdminRepository) MarkActiveRoleDeletedExcept(groupID, role, keepUserID string) error {
	groupID = strings.TrimSpace(groupID)
	role = normalizeGroupAdminRole(role)
	keepUserID = strings.TrimSpace(keepUserID)
	if groupID == "" || keepUserID == "" {
		return nil
	}
	return r.db.Model(&model.GroupBusinessAdmin{}).
		Where("group_id = ? AND role = ? AND status = ?", groupID, role, "active").
		Where("user_id <> ?", keepUserID).
		Updates(map[string]any{
			"status":     "deleted",
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *GroupBusinessAdminRepository) MarkActiveRoleDeletedExceptMany(groupID, role string, keepUserIDs []string) error {
	groupID = strings.TrimSpace(groupID)
	role = normalizeGroupAdminRole(role)
	if groupID == "" {
		return nil
	}
	keepSet := make(map[string]struct{}, len(keepUserIDs))
	normalizedKeepUserIDs := make([]string, 0, len(keepUserIDs))
	for _, userID := range keepUserIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, exists := keepSet[userID]; exists {
			continue
		}
		keepSet[userID] = struct{}{}
		normalizedKeepUserIDs = append(normalizedKeepUserIDs, userID)
	}
	query := r.db.Model(&model.GroupBusinessAdmin{}).
		Where("group_id = ? AND role = ? AND status = ?", groupID, role, "active")
	if len(normalizedKeepUserIDs) > 0 {
		query = query.Where("user_id NOT IN ?", normalizedKeepUserIDs)
	}
	return query.Updates(map[string]any{
		"status":     "deleted",
		"updated_at": gorm.Expr("NOW()"),
	}).Error
}
