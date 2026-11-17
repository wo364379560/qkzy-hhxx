package controller_admin

import (
	"errors"
	"regexp"
	"strings"

	mysqlerr "github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

var adminLoginNamePattern = regexp.MustCompile(`^[A-Za-z0-9]{3,32}$`)

func normalizeAdminLoginName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !adminLoginNamePattern.MatchString(value) {
		return "", errors.New("登录账号需为 3–32 位英文字母或数字，中文请填写在昵称中")
	}
	return value, nil
}

// The caller has already checked management scope. Renaming uses the immutable
// admin ID and never moves users, groups, assistants or invitation ownership.
// A nil username keeps the nickname-only API compatible with older clients.
func updateManagedAdminProfile(admin *model.AdminUser, username *string, displayName string) (*model.AdminUser, error) {
	updates := map[string]any{"display_name": displayName}
	if username != nil {
		name, err := normalizeAdminLoginName(*username)
		if err != nil {
			return nil, err
		}
		if name != admin.Username {
			updates["username"] = name
			updates["auth_version"] = gorm.Expr("GREATEST(COALESCE(auth_version, 1), 1) + 1")
		}
	}
	if len(updates) == 1 && displayName == admin.DisplayName {
		return admin, nil
	}
	result := datasource.Datasource.Mysql.GetClient().Model(&model.AdminUser{}).
		Where("id = ? AND username = ?", admin.ID, admin.Username).Updates(updates)
	if result.Error != nil {
		var duplicate *mysqlerr.MySQLError
		if errors.As(result.Error, &duplicate) && duplicate.Number == 1062 {
			return nil, errors.New("登录账号已存在")
		}
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, errors.New("账号资料已更新，请刷新后重试")
	}
	return repository.NewAdminUserRepository().FindByID(admin.ID)
}
