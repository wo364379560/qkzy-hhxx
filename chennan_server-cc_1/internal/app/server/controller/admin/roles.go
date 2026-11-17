package controller_admin

import (
	"errors"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleAssistant  = "assistant"
)

// ActorContext 表示后台账号的角色与可管理范围。
type ActorContext struct {
	AdminID       uint
	Role          string
	ParentAdminID *uint
	ScopeAdminIDs []uint
}

// IsSuperAdminRole 判断是否为超级管理员角色。
func IsSuperAdminRole(role string) bool {
	return role == RoleSuperAdmin
}

// HasManagerPrivileges 判断是否具备管理域权限（超级管理员或管理员）。
func HasManagerPrivileges(role string) bool {
	return role == RoleSuperAdmin || role == RoleAdmin
}

// IsAssistantRole 判断是否为助理角色。
func IsAssistantRole(role string) bool {
	return role == RoleAssistant
}

// BuildActorContext 构建后台操作者上下文。
func BuildActorContext(adminID uint) (*ActorContext, error) {
	adminRepo := repository.NewAdminUserRepository()
	admin, err := adminRepo.FindByID(adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("账号不存在")
		}
		return nil, err
	}

	scopeAdminIDs, err := ResolveScopeAdminIDs(admin)
	if err != nil {
		return nil, err
	}

	return &ActorContext{
		AdminID:       admin.ID,
		Role:          admin.Role,
		ParentAdminID: admin.ParentAdminID,
		ScopeAdminIDs: scopeAdminIDs,
	}, nil
}

// ResolveScopeAdminIDs 计算后台账号可管理的后台账号 ID 集合。
func ResolveScopeAdminIDs(admin *model.AdminUser) ([]uint, error) {
	adminRepo := repository.NewAdminUserRepository()

	switch admin.Role {
	case RoleSuperAdmin:
		return adminRepo.ListAllIDs()
	case RoleAdmin:
		assistants, err := adminRepo.ListByParentAdminID(admin.ID)
		if err != nil {
			return nil, err
		}
		scopeIDs := make([]uint, 0, len(assistants)+1)
		scopeIDs = append(scopeIDs, admin.ID)
		for _, assistant := range assistants {
			if assistant.Role != RoleAssistant {
				continue
			}
			scopeIDs = append(scopeIDs, assistant.ID)
		}
		return scopeIDs, nil
	case RoleAssistant:
		return []uint{admin.ID}, nil
	default:
		return []uint{admin.ID}, nil
	}
}

// CanManageOwnerID 判断某 owner 是否在当前操作者作用域内。
func CanManageOwnerID(actor *ActorContext, ownerID uint) bool {
	if actor == nil {
		return false
	}
	// 超管可管理任意归属，含 owner_id=0 与历史已删除后台账号留下的悬空归属。
	if actor.Role == RoleSuperAdmin {
		return true
	}
	for _, scopeID := range actor.ScopeAdminIDs {
		if scopeID == ownerID {
			return true
		}
	}
	return false
}

// CanManageUser 判断当前操作者能否管理目标用户。
func CanManageUser(actor *ActorContext, user *model.User) bool {
	return CanManageOwnerID(actor, user.OwnerID)
}

// CanManageAdminUser 判断当前操作者能否管理目标后台账号。
func CanManageAdminUser(actor *ActorContext, admin *model.AdminUser) bool {
	switch actor.Role {
	case RoleSuperAdmin:
		return true
	case RoleAdmin:
		return admin.Role == RoleAssistant && admin.ParentAdminID != nil && *admin.ParentAdminID == actor.AdminID
	case RoleAssistant:
		return admin.ID == actor.AdminID
	default:
		return false
	}
}

// ResolveScopeAdminUsernames 根据作用域后台账号 ID 集合解析用户名，用于审计日志按旧数据兼容过滤。
func ResolveScopeAdminUsernames(scopeAdminIDs []uint) ([]string, error) {
	if len(scopeAdminIDs) == 0 {
		return []string{}, nil
	}

	adminRepo := repository.NewAdminUserRepository()
	admins, err := adminRepo.FindByIDs(scopeAdminIDs)
	if err != nil {
		return nil, err
	}

	usernames := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin.Username == "" {
			continue
		}
		usernames = append(usernames, admin.Username)
	}
	return usernames, nil
}
