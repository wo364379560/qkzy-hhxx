package repository

import (
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminChatQuietHoursPolicyRepository 处理后台账号级静默时段策略。
type AdminChatQuietHoursPolicyRepository struct {
	db *gorm.DB
}

func NewAdminChatQuietHoursPolicyRepository() *AdminChatQuietHoursPolicyRepository {
	return &AdminChatQuietHoursPolicyRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *AdminChatQuietHoursPolicyRepository) FindByAdminUserID(adminUserID uint) (*model.AdminChatQuietHoursPolicy, error) {
	var policy model.AdminChatQuietHoursPolicy
	if err := r.db.Where("admin_user_id = ?", adminUserID).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *AdminChatQuietHoursPolicyRepository) Upsert(policy *model.AdminChatQuietHoursPolicy) error {
	return r.db.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "admin_user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"enabled":            policy.Enabled,
				"time_zone":          policy.TimeZone,
				"start":              policy.Start,
				"end":                policy.End,
				"block_register":     policy.BlockRegister,
				"block_send_message": policy.BlockSendMessage,
				"block_create_group": policy.BlockCreateGroup,
				"block_add_friend":   policy.BlockAddFriend,
				"updated_by":         policy.UpdatedBy,
			}),
		}).
		Create(policy).Error
}

func (r *AdminChatQuietHoursPolicyRepository) DeleteByAdminUserID(adminUserID uint) error {
	return r.db.Where("admin_user_id = ?", adminUserID).Delete(&model.AdminChatQuietHoursPolicy{}).Error
}
