package controller_admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/messagelimit"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/cloudauth"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

var adminIDNumberPattern = regexp.MustCompile(`^[1-9]\d{5}(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)

// GetRealnameVerifyEnabled 获取全局实名认证开关。
func GetRealnameVerifyEnabled() (bool, error) {
	return config.GetRealnameVerifyEnabled(context.Background(), datasource.Datasource.Mysql.GetClient())
}

// SetRealnameVerifyEnabled 设置全局实名认证开关。
func SetRealnameVerifyEnabled(adminID uint, currentPassword string, enabled bool) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}
	return config.SetRealnameVerifyEnabledTx(
		context.Background(),
		datasource.Datasource.Mysql.GetClient(),
		enabled,
		adminID,
	)
}

// ResetUserRealname 重置指定用户的实名认证状态（超管）。
func ResetUserRealname(adminID uint, userID uint) error {
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return err
	}
	if !IsSuperAdminRole(actor.Role) {
		return errors.New("仅超级管理员可操作")
	}

	user, err := repository.NewUserRepository().FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}
	if user.Status == "deleted" {
		return errors.New("已注销用户不能执行该操作")
	}

	if err := repository.NewUserRealNameRepository().Reset(userID); err != nil {
		return err
	}
	messagelimit.InvalidateRealNameVerified(context.Background(), userID)
	return nil
}

// SetUserRealnameVerified 后台简化实名开关：已实名 / 未实名（不强制走阿里云）。
// verified=true 时写入匿名占位实名，只用于业务状态判断，不展示真实文字信息。
func SetUserRealnameVerified(adminID, userID uint, verified bool) error {
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return err
	}
	if !IsSuperAdminRole(actor.Role) {
		return errors.New("仅超级管理员可操作")
	}

	user, err := repository.NewUserRepository().FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}
	if user.Status == "deleted" {
		return errors.New("已注销用户不能执行该操作")
	}

	repo := repository.NewUserRealNameRepository()
	if !verified {
		if err := repo.Reset(userID); err != nil {
			return err
		}
		messagelimit.InvalidateRealNameVerified(context.Background(), userID)
		return nil
	}
	if err := markUserRealnameVerified(user); err != nil {
		return err
	}
	messagelimit.InvalidateRealNameVerified(context.Background(), userID)
	return nil
}

// VerifyUserRealname 超管代用户提交二要素核验（调用阿里云 Id2MetaVerify）。
func VerifyUserRealname(adminID, userID uint, realName, idNumber string) error {
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return err
	}
	if !IsSuperAdminRole(actor.Role) {
		return errors.New("仅超级管理员可操作")
	}

	user, err := repository.NewUserRepository().FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}
	if user.Status == "deleted" {
		return errors.New("已注销用户不能执行该操作")
	}

	realName = strings.TrimSpace(realName)
	idNumber = strings.ToUpper(strings.TrimSpace(idNumber))
	if realName == "" {
		return errors.New("姓名不能为空")
	}
	if !adminIDNumberPattern.MatchString(idNumber) {
		return errors.New("身份证号格式无效")
	}

	repo := repository.NewUserRealNameRepository()
	existing, err := repo.FindByUserID(user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil && existing.Status == model.UserRealNameStatusVerified {
		return errors.New("该用户已完成实名认证，请先重置")
	}

	idHash := cloudauth.HashIDNumber(idNumber)
	occupied, err := repo.FindVerifiedByIDNumberHash(idHash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if occupied != nil && occupied.UserID != user.ID {
		return errors.New("该身份证号已被其他账号认证")
	}

	creds, err := cloudauth.LoadCredentialsFromEnv()
	if err != nil {
		if errors.Is(err, cloudauth.ErrNotConfigured) {
			return cloudauth.ErrNotConfigured
		}
		return err
	}

	verifyResult, err := cloudauth.Id2MetaVerify(creds, realName, idNumber)
	if err != nil {
		return err
	}

	idMasked := cloudauth.MaskIDNumber(idNumber)
	if !verifyResult.Matched {
		if saveErr := repo.SaveFailed(user.ID, idHash, idMasked); saveErr != nil {
			return saveErr
		}
		messagelimit.InvalidateRealNameVerified(context.Background(), user.ID)
		return errors.New("姓名与身份证号不一致")
	}

	now := time.Now()
	if err := repo.UpsertVerified(&model.UserRealName{
		UserID:          user.ID,
		RealName:        realName,
		IDNumberHash:    idHash,
		IDNumberMasked:  idMasked,
		Status:          model.UserRealNameStatusVerified,
		VerifiedAt:      &now,
		AliyunRequestID: verifyResult.RequestID,
	}); err != nil {
		return err
	}
	messagelimit.InvalidateRealNameVerified(context.Background(), user.ID)
	return nil
}

func realNameFields(record *model.UserRealName) (verified bool, realName, idNumberMasked string) {
	if record == nil || record.Status != model.UserRealNameStatusVerified {
		return false, "", ""
	}
	// 后台/列表统一脱敏展示；自动实名占位记录直接展示星号。
	return true, cloudauth.MaskRealName(record.RealName), record.IDNumberMasked
}

const (
	adminAssignedRealNamePlaceholder = "***"
	adminAssignedIDNumberMasked      = "******************"
)

// ownerIsAssignedTenant 归属到非超管后台账号（助理/管理员）视为「已归属」。
func ownerIsAssignedTenant(ownerID uint) bool {
	if ownerID == 0 {
		return false
	}
	owner, err := repository.NewAdminUserRepository().FindByID(ownerID)
	if err != nil || owner == nil {
		return false
	}
	return !IsSuperAdminRole(owner.Role)
}

// EnsureAssignedRealname 用户归属到非超管运营方时，自动标记已实名。
func EnsureAssignedRealname(user *model.User) {
	if user == nil || !ownerIsAssignedTenant(user.OwnerID) {
		return
	}
	if err := repository.NewUserRealNameRepository().UpsertAssignedVerified(user.ID); err == nil {
		messagelimit.InvalidateRealNameVerified(context.Background(), user.ID)
	}
}

func markUserRealnameVerified(user *model.User) error {
	if user == nil {
		return errors.New("用户不存在")
	}
	repo := repository.NewUserRealNameRepository()
	existing, err := repo.FindByUserID(user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil && existing.Status == model.UserRealNameStatusVerified {
		return nil
	}

	now := time.Now()
	// 匿名占位实名只用于业务状态判断；hash 按用户隔离，避免唯一约束冲突。
	idNumber := fmt.Sprintf("assigned-%d", user.ID)

	return repo.UpsertVerified(&model.UserRealName{
		UserID:         user.ID,
		RealName:       adminAssignedRealNamePlaceholder,
		IDNumberHash:   cloudauth.HashIDNumber(idNumber),
		IDNumberMasked: adminAssignedIDNumberMasked,
		Status:         model.UserRealNameStatusVerified,
		VerifiedAt:     &now,
	})
}
