package controller_admin

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// EnableUserTOTPByAdmin 打开用户 Google 验证。Web 下次登录需绑定或校验动态码；App 暂不强制。
func EnableUserTOTPByAdmin(actorID, userID uint) (*model.User, error) {
	user, err := findManagedUser(actorID, true, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return nil, err
	}
	enabled, err := repository.NewUserTOTPRepository().IsEnabled(user.ID)
	if err != nil {
		return nil, err
	}
	if enabled {
		return nil, errors.New("该用户已开启 Google 验证")
	}
	if err := repository.NewUserTOTPRepository().Require(user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

// DisableUserTOTPByAdmin 关闭用户 Google 验证，并撤销当前登录态。
func DisableUserTOTPByAdmin(actorID, userID uint) (*model.User, error) {
	user, err := findManagedUser(actorID, true, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return nil, err
	}
	enabled, err := repository.NewUserTOTPRepository().IsEnabled(user.ID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, errors.New("该用户未开启 Google 验证")
	}

	db := datasource.Datasource.Mysql.GetClient()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := repository.NewUserTOTPRepositoryWithDB(tx).Disable(user.ID); err != nil {
			return err
		}
		if err := repository.NewUserTOTPBackupCodeRepositoryWithDB(tx).DeleteByUserID(user.ID); err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&model.UserRefreshSession{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Update("revoked_at", now).Error
	}); err != nil {
		return nil, err
	}

	return user, nil
}
