package controller_admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/app/server/usercache"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/idgen"
	"github.com/ixayda/ix-Go/pkg/idgs"
	"github.com/ixayda/ix-Go/pkg/logger"
)

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
var wananIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{5,19}$`)

// UserResult 用户信息响应。
type UserResult struct {
	OwnerRevision       uint64 `json:"ownerRevision"`
	OwnerState          string `json:"ownerState"`
	ID                  uint   `json:"id"`
	IMUserID            string `json:"imUserID"`
	WananID             string `json:"wananID"`
	Nickname            string `json:"nickname"`
	UserTag             string `json:"userTag"`
	AvatarURL           string `json:"avatarUrl"`
	Phone               string `json:"phone"`
	CurrentLoginIP      string `json:"currentLoginIP"`
	LastOnlineAt        string `json:"lastOnlineAt"`
	LastOfflineAt       string `json:"lastOfflineAt"`
	IPWhitelistEnabled  bool   `json:"ipWhitelistEnabled"`
	IPWhitelistIP       string `json:"ipWhitelistIP"`
	TOTPEnabled         bool   `json:"totpEnabled"`
	OwnerID             uint   `json:"ownerID"`
	OwnerName           string `json:"ownerName"`
	OwnerRole           string `json:"ownerRole"`
	OwnerStatus         string `json:"ownerStatus"`
	InviteCode          string `json:"inviteCode"`
	Status              string `json:"status"`
	AccountStatus       string `json:"accountStatus"`
	LoginStatus         string `json:"loginStatus"`
	LoginReason         string `json:"loginReason"`
	LoginCooldown       bool   `json:"loginCooldown"`
	LoginCooldownUntil  string `json:"loginCooldownUntil"`
	LoginCooldownReason string `json:"loginCooldownReason"`
	CleanupStatus       string `json:"cleanupStatus"`
	CleanupReason       string `json:"cleanupReason"`
	IsActivated         bool   `json:"isActivated"`
	CanCreateGroup      bool   `json:"canCreateGroup"`
	GloballySearchable  bool   `json:"globallySearchable"`
	Gender              int8   `json:"gender"`
	Birthday            string `json:"birthday"`
	Location            string `json:"location"`
	SelfSignature       string `json:"selfSignature"`
	CreatedAt           string `json:"createdAt"`

	UserNo                string `json:"userNo"`
	LedgerAllowed         bool   `json:"ledgerAllowed"`
	SplitAllowed          bool   `json:"splitAllowed"`
	SavingPostDisabled    bool   `json:"savingPostDisabled"`
	SavingCommentDisabled bool   `json:"savingCommentDisabled"`

	RealNameVerified bool   `json:"realNameVerified"`
	RealName         string `json:"realName"`
	IDNumberMasked   string `json:"idNumberMasked"`
}

// UserOwnerOption 是用户管理页可选归属人的最小信息。
type UserOwnerOption struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func toUserResult(
	u *model.User,
	ownerMap map[uint]*model.AdminUser,
	cleanupDispositionMap map[uint]*model.CleanupUserDisposition,
	loginCooldownMap map[string]model.LoginRiskControl,
) UserResult {
	birthday := ""
	if u.Birthday != nil {
		birthday = u.Birthday.Format("2006-01-02")
	}
	lastOnlineAt := ""
	if u.LastOnlineAt != nil {
		lastOnlineAt = u.LastOnlineAt.Format("2006-01-02 15:04:05")
	}
	lastOfflineAt := ""
	if u.LastOfflineAt != nil {
		lastOfflineAt = u.LastOfflineAt.Format("2006-01-02 15:04:05")
	}
	ownerName, ownerRole, ownerStatus := "", "", ""
	var owner *model.AdminUser
	if ownerMap != nil {
		if matchedOwner, ok := ownerMap[u.OwnerID]; ok {
			owner = matchedOwner
			ownerName = owner.Username
			ownerRole = owner.Role
			ownerStatus = owner.Status
		}
	}
	loginStatus, loginReason := deriveUserLoginState(u, owner)
	cleanupStatus, cleanupReason := "", ""
	if cleanupDispositionMap != nil {
		if disposition, ok := cleanupDispositionMap[u.ID]; ok {
			cleanupStatus = strings.TrimSpace(disposition.CleanupStatus)
			cleanupReason = strings.TrimSpace(disposition.Reason)
		}
	}
	loginCooldown, loginCooldownUntil, loginCooldownReason := false, "", ""
	if loginCooldownMap != nil {
		if cooldown, ok := loginCooldownMap[strings.TrimSpace(u.Phone)]; ok {
			loginCooldown = true
			if cooldown.CooldownUntil != nil {
				loginCooldownUntil = cooldown.CooldownUntil.Format("2006-01-02 15:04:05")
			}
			loginCooldownReason = strings.TrimSpace(cooldown.Reason)
		}
	}
	return UserResult{
		ID:                 u.ID,
		IMUserID:           u.IMUserID,
		WananID:            optionalString(u.WananID),
		Nickname:           u.Nickname,
		UserTag:            strings.TrimSpace(u.UserTag),
		AvatarURL:          u.AvatarURL,
		Phone:              u.Phone,
		LastOnlineAt:       lastOnlineAt,
		LastOfflineAt:      lastOfflineAt,
		IPWhitelistEnabled: u.IPWhitelistEnabled,
		IPWhitelistIP:      strings.TrimSpace(u.IPWhitelistIP),
		OwnerID:            u.OwnerID,
		OwnerRevision:      u.OwnerRevision, OwnerState: model.ClassifyOwner(u, ownerMap[u.OwnerID]).State,
		OwnerName:             ownerName,
		OwnerRole:             ownerRole,
		OwnerStatus:           ownerStatus,
		InviteCode:            u.InviteCode,
		Status:                loginStatus,
		AccountStatus:         u.Status,
		LoginStatus:           loginStatus,
		LoginReason:           loginReason,
		LoginCooldown:         loginCooldown,
		LoginCooldownUntil:    loginCooldownUntil,
		LoginCooldownReason:   loginCooldownReason,
		CleanupStatus:         cleanupStatus,
		CleanupReason:         cleanupReason,
		IsActivated:           u.IsActivated,
		CanCreateGroup:        u.CanCreateGroup,
		GloballySearchable:    u.GloballySearchable,
		Gender:                u.Gender,
		Birthday:              birthday,
		Location:              u.Location,
		SelfSignature:         u.SelfSignature,
		CreatedAt:             u.CreatedAt.Format("2006-01-02 15:04:05"),
		UserNo:                optionalString(u.WananID),
		LedgerAllowed:         true,
		SplitAllowed:          u.CanCreateGroup,
		SavingPostDisabled:    u.CommunityPostDisabled,
		SavingCommentDisabled: u.CommunityCommentDisabled,
	}
}

func toUserOwnerOption(admin *model.AdminUser) UserOwnerOption {
	return UserOwnerOption{
		ID:       admin.ID,
		Username: admin.Username,
		Role:     admin.Role,
		Status:   admin.Status,
	}
}

func deriveUserLoginState(user *model.User, owner *model.AdminUser) (string, string) {
	if user.Status == "deleted" {
		return "deleted", ""
	}
	if user.Status == "banned" {
		return "banned", ""
	}
	if owner != nil && owner.Status != "active" {
		return "banned", "owner_banned"
	}
	return "active", ""
}

func ensureUserNotDeleted(user *model.User) error {
	if user != nil && user.Status == "deleted" {
		return errors.New("已注销用户不能执行该操作")
	}
	return nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// ListUsersReq 查询参数。
type ListUsersReq struct {
	Page               int
	PageSize           int
	OwnerID            uint // 0 表示不限制（管理员）
	ScopeOwnerIDs      []uint
	Keyword            string
	Status             string
	CurrentLoginIP     string
	IsActivated        *bool // nil = 不过滤
	IPWhitelistEnabled *bool
}

// ExportUsersReq 表示导出用户资料的筛选参数。
type ExportUsersReq struct {
	OwnerID            uint
	ScopeOwnerIDs      []uint
	Keyword            string
	Status             string
	CurrentLoginIP     string
	IsActivated        *bool
	IPWhitelistEnabled *bool
}

// ExportUserRow 表示导出的用户资料行。
type ExportUserRow struct {
	Nickname       string
	IMUserID       string
	WananID        string
	Phone          string
	CurrentLoginIP string
	LastOnlineAt   string
	LastOfflineAt  string
	OwnerName      string
	OwnerRole      string
	Status         string
	LoginStatus    string
	IsActivated    bool
	InviteCode     string
	CreatedAt      string
}

// ListUserOwners 返回当前操作者可用的归属人列表。
func ListUserOwners(actorID uint, isAdmin bool) ([]UserOwnerOption, error) {
	actor, err := BuildActorContext(actorID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && !IsAssistantRole(actor.Role) {
		return nil, errors.New("无权查看归属人列表")
	}

	adminRepo := repository.NewAdminUserRepository()
	owners, err := adminRepo.FindByIDs(actor.ScopeAdminIDs)
	if err != nil {
		return nil, err
	}
	sort.Slice(owners, func(i, j int) bool {
		return strings.ToLower(owners[i].Username) < strings.ToLower(owners[j].Username)
	})
	options := make([]UserOwnerOption, 0, len(owners))
	for i := range owners {
		if owners[i].Status != "active" {
			continue
		}
		options = append(options, toUserOwnerOption(&owners[i]))
	}
	return options, nil
}

// ListUsers 分页列出用户。
func ListUsers(req ListUsersReq) (PageResult[UserResult], error) {
	repo := repository.NewUserRepository()
	users, total, err := repo.ListPaged(
		req.Page,
		req.PageSize,
		req.ScopeOwnerIDs,
		req.OwnerID,
		req.Keyword,
		req.Status,
		req.CurrentLoginIP,
		req.IsActivated,
		req.IPWhitelistEnabled,
	)
	if err != nil {
		return PageResult[UserResult]{}, err
	}

	list, err := buildUserResultList(users)
	if err != nil {
		return PageResult[UserResult]{}, err
	}
	return PageResult[UserResult]{List: list, Total: total}, nil
}

// ExportUsers 查询符合筛选条件的全部用户资料，用于导出。
func ExportUsers(req ExportUsersReq) ([]ExportUserRow, error) {
	repo := repository.NewUserRepository()
	users, err := repo.ListForExport(
		req.ScopeOwnerIDs,
		req.OwnerID,
		req.Keyword,
		req.Status,
		req.CurrentLoginIP,
		req.IsActivated,
		req.IPWhitelistEnabled,
	)
	if err != nil {
		return nil, err
	}

	list, err := buildUserResultList(users)
	if err != nil {
		return nil, err
	}

	rows := make([]ExportUserRow, 0, len(list))
	for _, item := range list {
		rows = append(rows, ExportUserRow{
			Nickname:       item.Nickname,
			IMUserID:       item.IMUserID,
			WananID:        item.WananID,
			Phone:          item.Phone,
			CurrentLoginIP: item.CurrentLoginIP,
			LastOnlineAt:   item.LastOnlineAt,
			LastOfflineAt:  item.LastOfflineAt,
			OwnerName:      item.OwnerName,
			OwnerRole:      item.OwnerRole,
			Status:         item.Status,
			LoginStatus:    item.LoginStatus,
			IsActivated:    item.IsActivated,
			InviteCode:     item.InviteCode,
			CreatedAt:      item.CreatedAt,
		})
	}
	return rows, nil
}

func buildUserResultList(users []model.User) ([]UserResult, error) {
	// 批量查归属人信息（避免 N+1）
	ownerIDs := make([]uint, 0, len(users))
	seen := map[uint]bool{}
	for _, u := range users {
		if u.OwnerID > 0 && !seen[u.OwnerID] {
			ownerIDs = append(ownerIDs, u.OwnerID)
			seen[u.OwnerID] = true
		}
	}
	adminRepo := repository.NewAdminUserRepository()
	owners, err := adminRepo.FindByIDs(ownerIDs)
	if err != nil {
		return nil, err
	}
	ownerMap := make(map[uint]*model.AdminUser, len(owners))
	for i := range owners {
		ownerMap[owners[i].ID] = &owners[i]
	}

	userIDs := make([]uint, 0, len(users))
	phones := make([]string, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		if phone := strings.TrimSpace(user.Phone); phone != "" {
			phones = append(phones, phone)
		}
	}
	sessionMap, err := repository.NewUserRefreshSessionRepository().FindLatestActiveByUserIDs(userIDs)
	if err != nil {
		return nil, err
	}
	totpEnabledMap, err := repository.NewUserTOTPRepository().EnabledMap(userIDs)
	if err != nil {
		return nil, err
	}

	realNameMap, err := repository.NewUserRealNameRepository().FindByUserIDs(userIDs)
	if err != nil {
		return nil, err
	}

	cleanupDispositionMap := make(map[uint]*model.CleanupUserDisposition)
	if len(userIDs) > 0 {
		var dispositions []model.CleanupUserDisposition
		if err := datasource.Datasource.Mysql.GetClient().
			Where("user_id IN ?", userIDs).
			Find(&dispositions).Error; err != nil {
			return nil, err
		}
		for i := range dispositions {
			cleanupDispositionMap[dispositions[i].UserID] = &dispositions[i]
		}
	}

	loginCooldownMap, err := repository.NewLoginRiskControlRepository().ActiveAccountCooldownsByPhones(phones, time.Now())
	if err != nil {
		return nil, err
	}

	list := make([]UserResult, 0, len(users))
	for i := range users {
		item := toUserResult(&users[i], ownerMap, cleanupDispositionMap, loginCooldownMap)
		if session, ok := sessionMap[users[i].ID]; ok {
			item.CurrentLoginIP = strings.TrimSpace(session.IP)
		}
		item.TOTPEnabled = totpEnabledMap[users[i].ID]
		if record, ok := realNameMap[users[i].ID]; ok {
			item.RealNameVerified, item.RealName, item.IDNumberMasked = realNameFields(record)
		}
		list = append(list, item)
	}
	return list, nil
}

// CreateUserReq 管理员建号参数。
type CreateUserReq struct {
	Phone     string // 必填，登录账号
	Password  string // 必填，明文，函数内 hash
	Nickname  string // 选填，为空则自动占位，首登时用户设置
	AvatarURL string
	OwnerID   uint
	CreatedBy uint
	// 默认 false：管理员建号和模板生成账号不允许建群。
	CanCreateGroup bool
}

// CreateUser 管理员手动建号（a_ 前缀）。
func CreateUser(ctx context.Context, actorID uint, isAdmin bool, req CreateUserReq) (*UserResult, error) {
	repo := repository.NewUserRepository()
	ownerID, err := resolveManagedOwnerID(actorID, isAdmin, req.OwnerID)
	if err != nil {
		return nil, err
	}
	phone := strings.TrimSpace(req.Phone)
	if !phonePattern.MatchString(phone) {
		return nil, errors.New("手机号格式无效")
	}

	// 手机号唯一性检查
	exists, err := repo.ExistsByPhone(phone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("手机号已注册")
	}

	// 生成 IM UserID（a_ 前缀 = 管理员建号）
	imUserID, err := idgen.GenUserID(ctx, "a")
	if err != nil {
		return nil, err
	}

	// 密码 hash
	hash, err := authwork.GeneratePrivateFromPassword([]byte(req.Password), authwork.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 昵称选填：为空则用占位符，首登时替换
	nickname := req.Nickname
	if nickname == "" {
		nickname = "用户_" + imUserID[2:] // strip "a_" prefix
	}

	inviteCode, inviteCodeID, err := resolveOwnerInviteCode(ownerID)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		BizID:              idgs.Global.UserGenerate().Int64(),
		IMUserID:           imUserID,
		Phone:              phone,
		PasswordHash:       string(hash),
		Nickname:           nickname,
		AvatarURL:          req.AvatarURL,
		OwnerID:            ownerID,
		InviteCode:         inviteCode,
		CreatedBy:          req.CreatedBy,
		AllowType:          0,
		CanCreateGroup:     req.CanCreateGroup,
		Status:             "active",
		IsActivated:        false,
		MustChangePassword: true,
		CreatedAt:          time.Now(),
	}
	if err = repo.Create(user); err != nil {
		return nil, err
	}

	// 同步到腾讯 IM（管理建号同步调用，需立即可用）
	imClient := tencentim.NewClient()
	if err = imClient.AccountImport(imUserID, nickname, req.AvatarURL); err != nil {
		_ = repo.Delete(user.ID)
		return nil, errors.New("同步 IM 失败，请稍后重试")
	}
	// 默认关闭「加我为好友需要验证」。
	if err = imClient.SetAllowType(imUserID, user.AllowType); err != nil {
		_ = repo.Delete(user.ID)
		_ = imClient.AccountDelete(imUserID)
		return nil, errors.New("同步 IM 加好友策略失败，请稍后重试")
	}

	incrementInviteCodeUsage(inviteCodeID)

	EnsureAssignedRealname(user)

	result := toUserResult(user, nil, nil, nil)
	return &result, nil
}

// BatchCreateUsers 批量建号，普通单条失败不中断；认证 CPU 保护触发时立即停止。
func BatchCreateUsers(ctx context.Context, actorID uint, isAdmin bool, reqs []CreateUserReq) ([]BatchResult, error) {
	results := make([]BatchResult, len(reqs))
	for i, req := range reqs {
		if _, err := CreateUser(ctx, actorID, isAdmin, req); err != nil {
			if errors.Is(err, authwork.ErrBusy) {
				if i == 0 {
					return nil, err
				}
				for pending := i; pending < len(reqs); pending++ {
					results[pending] = BatchResult{Index: pending, Success: false, Message: "服务繁忙，未执行"}
				}
				return results, nil
			}
			results[i] = BatchResult{Index: i, Success: false, Message: batchCreateUserErrorMessage(err)}
		} else {
			results[i] = BatchResult{Index: i, Success: true, Message: "创建成功"}
		}
	}
	return results, nil
}

// batchCreateUserErrorMessage 只向客户端返回明确的业务错误，避免泄露数据库/驱动细节。
func batchCreateUserErrorMessage(err error) string {
	if err == nil {
		return "创建失败，请稍后重试"
	}
	switch err.Error() {
	case "手机号格式无效",
		"手机号已注册",
		"助理不能指定其他归属人",
		"归属人不存在",
		"无权指定该归属人",
		"同步 IM 失败，请稍后重试",
		"同步 IM 加好友策略失败，请稍后重试":
		return err.Error()
	default:
		return "创建失败，请稍后重试"
	}
}

// BanUser 封禁用户（同时踢下线 + 禁言，尽力而为）。
func BanUser(ctx context.Context, actorID uint, isAdmin bool, userID uint) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(actorID, isAdmin, userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}
	if err = repo.UpdateStatus(userID, "banned"); err != nil {
		return err
	}
	usercache.InvalidateByIMUserID(ctx, user.IMUserID)
	keywordfilter.InvalidateFilterSenders()
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		_ = imClient.AccountKick(user.IMUserID)
		_ = imClient.AccountBan(user.IMUserID)
	}
	return upsertCleanupDisposition(datasource.Datasource.Mysql.GetClient(), user, CleanupStatusManualBanned, "用户管理手动封禁", actorID)
}

// UnbanUser 解封用户。
func UnbanUser(ctx context.Context, actorID uint, isAdmin bool, userID uint) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(actorID, isAdmin, userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}
	if user.OwnerID > 0 {
		owner, ownerErr := repository.NewAdminUserRepository().FindByID(user.OwnerID)
		if ownerErr != nil {
			return ownerErr
		}
		if owner.Status != "active" {
			return errors.New("该用户无法解封，其归属助理未解封")
		}
	}
	if err = repo.UpdateStatus(userID, "active"); err != nil {
		return err
	}
	usercache.InvalidateByIMUserID(ctx, user.IMUserID)
	keywordfilter.InvalidateFilterSenders()
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		_ = imClient.AccountUnban(user.IMUserID)
	}
	return upsertCleanupDisposition(datasource.Datasource.Mysql.GetClient(), user, CleanupStatusRestored, "用户管理手动解封", actorID)
}

// KickUser 强制踢下线（尽力而为，IM 调用失败不影响结果）。
func KickUser(ctx context.Context, actorID uint, isAdmin bool, userID uint) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(actorID, isAdmin, userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}
	if err := repo.UpdateFields(user.ID, map[string]any{
		"auth_version": gorm.Expr("GREATEST(auth_version, 1) + 1"),
	}); err != nil {
		return err
	}
	if err := repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID); err != nil {
		return err
	}
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		_ = imClient.AccountKick(user.IMUserID)
	}
	return nil
}

// DeleteUser 注销用户（软删除本地账号 + 尽力踢下线）。
func DeleteUser(ctx context.Context, actorID uint, isAdmin bool, userID uint) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(actorID, isAdmin, userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}
	if err := repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID); err != nil {
		return err
	}
	if err := repo.SoftDelete(user.ID); err != nil {
		return err
	}
	usercache.InvalidateByIMUserID(ctx, user.IMUserID)
	keywordfilter.InvalidateFilterSenders()

	go func(imUserID string) {
		imClient := tencentim.NewClient()
		if !imClient.IsConfigured() {
			return
		}
		_ = imClient.AccountKick(imUserID)
	}(user.IMUserID)

	return nil
}

// UserDeletePreviewResult 表示注销用户前的影响预览。
type UserDeletePreviewResult struct {
	UserID              uint   `json:"userID"`
	Nickname            string `json:"nickname"`
	Phone               string `json:"phone"`
	IMUserID            string `json:"imUserID"`
	Mode                string `json:"mode"`
	WillDeleteLocalData bool   `json:"willDeleteLocalData"`
	WillDeleteTencentIM bool   `json:"willDeleteTencentIM"`
	WillKickOnline      bool   `json:"willKickOnline"`
	OperationDetail     string `json:"operationDetail"`
}

// PreviewDeleteUser 预览注销用户将影响的本地数据量。
func PreviewDeleteUser(ctx context.Context, actorID uint, isAdmin bool, userID uint) (*UserDeletePreviewResult, error) {
	user, err := findManagedUser(actorID, isAdmin, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return nil, err
	}

	return &UserDeletePreviewResult{
		UserID:              user.ID,
		Nickname:            user.Nickname,
		Phone:               user.Phone,
		IMUserID:            user.IMUserID,
		Mode:                "soft_delete",
		WillDeleteLocalData: false,
		WillDeleteTencentIM: false,
		WillKickOnline:      true,
		OperationDetail:     "系统会将账号标记为已注销，撤销所有登录态并踢下线；本地业务数据和腾讯 IM 账号不会删除。",
	}, nil
}

// TransferUser 转移用户归属。
func TransferUser(ctx context.Context, actorID uint, isAdmin bool, userID, targetAdminID uint) error {
	_, err := TransferUserWithAssignment(ctx, actorID, isAdmin, userID, targetAdminID)
	return err
}

func TransferUserWithAssignment(ctx context.Context, actorID uint, isAdmin bool, userID, targetAdminID uint) (*model.OwnerAssignment, error) {
	if !isAdmin {
		return nil, errors.New("仅管理员可转移归属")
	}
	resolvedTargetOwnerID, err := resolveManagedOwnerID(actorID, true, targetAdminID)
	if err != nil {
		return nil, err
	}
	assignment, err := repository.NewOwnershipRepository().Transfer(ctx, actorID, userID, resolvedTargetOwnerID)
	if err != nil {
		var policy repository.OwnershipError
		if errors.As(err, &policy) {
			return nil, fmt.Errorf("归属转移未完成：%s", policy.Code)
		}
		return nil, err
	}
	usercache.InvalidateByIMUserID(ctx, assignment.Target.IMUserID)
	keywordfilter.InvalidateFilterSenders()
	return assignment, nil
}

func resolveOwnerInviteCode(ownerID uint) (code string, codeID uint, err error) {
	if ownerID == 0 {
		return "", 0, nil
	}

	codeRepo := repository.NewInviteCodeRepository()
	ic, err := codeRepo.FindActiveByOwner(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, nil
		}
		return "", 0, err
	}

	return strings.TrimSpace(ic.Code), ic.ID, nil
}

func incrementInviteCodeUsage(codeID uint) {
	if codeID == 0 {
		return
	}
	_ = repository.NewInviteCodeRepository().IncrementUsedCount(codeID)
}

// SyncUserPreview 用户同步预览项。
type SyncUserPreview struct {
	IMUserID       string   `json:"imUserID"`
	LocalUserID    uint     `json:"localUserId"`
	LocalNickname  string   `json:"localNickname"`
	IMNickname     string   `json:"imNickname"`
	LocalAvatarURL string   `json:"localAvatarUrl"`
	IMAvatarURL    string   `json:"imAvatarUrl"`
	DiffFields     []string `json:"diffFields"`
	Status         string   `json:"status"`
	Action         string   `json:"action"`
}

// SyncUsersPreview 基于本地 users 索引与腾讯 IM 资料生成同步预览列表。
func SyncUsersPreview(adminID uint, isAdmin bool) ([]SyncUserPreview, error) {
	if !isAdmin {
		return nil, errors.New("仅管理员可执行同步操作")
	}
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	userRepo := repository.NewUserRepository()
	var users []model.User
	if IsSuperAdminRole(actor.Role) {
		users, err = userRepo.FindAll()
	} else {
		users, err = userRepo.FindAllByOwners(actor.ScopeAdminIDs)
	}
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return []SyncUserPreview{}, nil
	}

	imUserIDs := make([]string, 0, len(users))
	for _, user := range users {
		if strings.TrimSpace(user.IMUserID) == "" {
			continue
		}
		imUserIDs = append(imUserIDs, user.IMUserID)
	}
	if len(imUserIDs) == 0 {
		return []SyncUserPreview{}, nil
	}

	infoMap, err := imClient.AccountGetInfo(imUserIDs)
	if err != nil {
		return nil, err
	}

	previews := make([]SyncUserPreview, 0)
	for _, user := range users {
		if strings.TrimSpace(user.IMUserID) == "" {
			continue
		}

		info, ok := infoMap[user.IMUserID]
		preview := SyncUserPreview{
			IMUserID:       user.IMUserID,
			LocalUserID:    user.ID,
			LocalNickname:  user.Nickname,
			LocalAvatarURL: user.AvatarURL,
			Action:         "skip",
		}

		if !ok {
			preview.Status = "missing_in_im"
			previews = append(previews, preview)
			continue
		}

		preview.IMNickname = info.Nickname
		preview.IMAvatarURL = info.FaceURL
		preview.DiffFields = collectUserSyncDiffFields(&user, info)

		switch {
		case len(preview.DiffFields) == 0:
			preview.Status = "synced"
		case len(preview.DiffFields) > 1:
			preview.Status = "changed"
		case preview.DiffFields[0] == "昵称":
			preview.Status = "nickname_changed"
		case preview.DiffFields[0] == "头像":
			preview.Status = "avatar_changed"
		default:
			preview.Status = "profile_changed"
		}

		if preview.Status != "synced" {
			previews = append(previews, preview)
		}
	}

	return previews, nil
}

// SyncUserSelection 用户同步选择项。
type SyncUserSelection struct {
	IMUserID string `json:"imUserID"`
	Action   string `json:"action"`
}

// BatchUserActionDetail 表示批量用户操作单条结果。
type BatchUserActionDetail struct {
	UserID   uint   `json:"userID"`
	Nickname string `json:"nickname"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}

// BatchUserActionResult 表示批量用户操作汇总结果。
type BatchUserActionResult struct {
	Success int                     `json:"success"`
	Failed  int                     `json:"failed"`
	Details []BatchUserActionDetail `json:"details"`
}

// BatchUserIPWhitelistDetail 表示批量设置用户登录白名单单条结果。
type BatchUserIPWhitelistDetail struct {
	UserID       uint   `json:"userID"`
	Nickname     string `json:"nickname"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ForcedLogout bool   `json:"forced_logout"`
}

// BatchUserIPWhitelistResult 表示批量设置用户登录白名单汇总结果。
type BatchUserIPWhitelistResult struct {
	Success      int                          `json:"success"`
	Failed       int                          `json:"failed"`
	ForcedLogout int                          `json:"forced_logout"`
	Details      []BatchUserIPWhitelistDetail `json:"details"`
}

// SyncUsersConfirm 确认同步选中的用户资料。
func SyncUsersConfirm(adminID uint, isAdmin bool, selections []SyncUserSelection) (map[string]interface{}, error) {
	if !isAdmin {
		return nil, errors.New("仅管理员可执行同步操作")
	}
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	userRepo := repository.NewUserRepository()

	success := 0
	failed := 0

	for _, sel := range selections {
		if sel.Action == "skip" {
			continue
		}
		if sel.Action != "update" {
			return nil, errors.New("用户同步仅支持 update 操作")
		}

		localUser, err := userRepo.FindByIMUserID(sel.IMUserID)
		if err != nil {
			failed++
			continue
		}
		if !CanManageUser(actor, localUser) {
			failed++
			continue
		}

		infoMap, err := imClient.AccountGetInfo([]string{sel.IMUserID})
		if err != nil {
			failed++
			continue
		}
		info, ok := infoMap[sel.IMUserID]
		if !ok {
			failed++
			continue
		}

		localUser.Nickname = info.Nickname
		localUser.AvatarURL = info.FaceURL
		localUser.SelfSignature = info.SelfSignature
		localUser.Gender = info.Gender
		localUser.AllowType = info.AllowType
		localUser.Birthday = syncBirthdayToTime(info.Birthday)
		if err := userRepo.Update(localUser); err != nil {
			failed++
			continue
		}
		success++
	}

	return map[string]interface{}{
		"success": success,
		"failed":  failed,
	}, nil
}

func collectUserSyncDiffFields(localUser *model.User, imUser tencentim.UserInfo) []string {
	fields := make([]string, 0, 6)
	if localUser.Nickname != imUser.Nickname {
		fields = append(fields, "昵称")
	}
	if localUser.AvatarURL != imUser.FaceURL {
		fields = append(fields, "头像")
	}
	if localUser.SelfSignature != imUser.SelfSignature {
		fields = append(fields, "签名")
	}
	if localUser.Gender != imUser.Gender {
		fields = append(fields, "性别")
	}
	if birthdayToInt(localUser.Birthday) != imUser.Birthday {
		fields = append(fields, "生日")
	}
	if localUser.AllowType != imUser.AllowType {
		fields = append(fields, "加好友策略")
	}
	return fields
}

func birthdayToInt(value *time.Time) int {
	if value == nil {
		return 0
	}
	return value.Year()*10000 + int(value.Month())*100 + value.Day()
}

func syncBirthdayToTime(value int) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.Date(
		value/10000,
		time.Month((value/100)%100),
		value%100,
		0, 0, 0, 0, time.Local,
	)
	return &t
}

// BatchBanUsers 批量封禁用户。
func BatchBanUsers(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint) (BatchUserActionResult, error) {
	return batchProcessUsers(ctx, actorID, isAdmin, userIDs, func(userID uint) error {
		return BanUser(ctx, actorID, isAdmin, userID)
	}, "已封禁")
}

// BatchUnbanUsers 批量解封用户。
func BatchUnbanUsers(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint) (BatchUserActionResult, error) {
	return batchProcessUsers(ctx, actorID, isAdmin, userIDs, func(userID uint) error {
		return UnbanUser(ctx, actorID, isAdmin, userID)
	}, "已解封")
}

// BatchDeleteUsers 批量注销用户。
func BatchDeleteUsers(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint) (BatchUserActionResult, error) {
	return batchProcessUsers(ctx, actorID, isAdmin, userIDs, func(userID uint) error {
		return DeleteUser(ctx, actorID, isAdmin, userID)
	}, "已注销")
}

// BatchTransferUsers 批量转移用户归属。
func BatchTransferUsers(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint, targetAdminID uint) (BatchUserActionResult, error) {
	if !isAdmin {
		return BatchUserActionResult{}, errors.New("仅管理员可批量分配用户")
	}
	if _, err := resolveManagedOwnerID(actorID, true, targetAdminID); err != nil {
		return BatchUserActionResult{}, err
	}
	return batchProcessUsers(ctx, actorID, isAdmin, userIDs, func(userID uint) error {
		return TransferUser(ctx, actorID, isAdmin, userID, targetAdminID)
	}, "已分配")
}

func batchProcessUsers(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint, action func(userID uint) error, successMessage string) (BatchUserActionResult, error) {
	result := BatchUserActionResult{
		Details: make([]BatchUserActionDetail, 0, len(userIDs)),
	}
	if len(userIDs) == 0 {
		return result, errors.New("请选择至少一个用户")
	}

	seen := make(map[uint]bool, len(userIDs))
	for _, userID := range userIDs {
		if seen[userID] {
			continue
		}
		seen[userID] = true

		user, err := findManagedUser(actorID, isAdmin, userID)
		if err != nil {
			result.Failed++
			result.Details = append(result.Details, BatchUserActionDetail{
				UserID:  userID,
				Success: false,
				Message: err.Error(),
			})
			continue
		}

		if err := action(userID); err != nil {
			result.Failed++
			result.Details = append(result.Details, BatchUserActionDetail{
				UserID:   user.ID,
				Nickname: user.Nickname,
				Success:  false,
				Message:  err.Error(),
			})
			continue
		}

		result.Success++
		result.Details = append(result.Details, BatchUserActionDetail{
			UserID:   user.ID,
			Nickname: user.Nickname,
			Success:  true,
			Message:  successMessage,
		})
	}

	return result, nil
}

// GetUser 获取用户详情。
func GetUser(userID, adminID uint, role string) (map[string]interface{}, error) {
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return nil, err
	}

	// 获取归属管理员信息
	var ownerName string
	var ownerStatus string
	var owner *model.AdminUser
	if user.OwnerID > 0 {
		adminRepo := repository.NewAdminUserRepository()
		owner, _ = adminRepo.FindByID(user.OwnerID)
		if owner != nil {
			ownerName = owner.Username
			ownerStatus = owner.Status
		}
	}
	loginStatus, loginReason := deriveUserLoginState(user, owner)
	currentLoginIP := ""
	session, err := repository.NewUserRefreshSessionRepository().FindLatestActiveByUserID(user.ID)
	if err == nil && session != nil {
		currentLoginIP = strings.TrimSpace(session.IP)
	}
	cleanupStatus, cleanupReason := "", ""
	var disposition model.CleanupUserDisposition
	if err := datasource.Datasource.Mysql.GetClient().Where("user_id = ?", user.ID).First(&disposition).Error; err == nil {
		cleanupStatus = strings.TrimSpace(disposition.CleanupStatus)
		cleanupReason = strings.TrimSpace(disposition.Reason)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	realNameVerified, realName, idNumberMasked := false, "", ""
	if record, err := repository.NewUserRealNameRepository().FindByUserID(user.ID); err == nil {
		realNameVerified, realName, idNumberMasked = realNameFields(record)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return map[string]interface{}{
		"id":                         user.ID,
		"im_user_id":                 user.IMUserID,
		"wanan_id":                   optionalString(user.WananID),
		"phone":                      user.Phone,
		"nickname":                   user.Nickname,
		"user_tag":                   strings.TrimSpace(user.UserTag),
		"avatar_url":                 user.AvatarURL,
		"self_signature":             user.SelfSignature,
		"gender":                     user.Gender,
		"birthday":                   user.Birthday,
		"owner_id":                   user.OwnerID,
		"owner_name":                 ownerName,
		"owner_revision":             user.OwnerRevision,
		"owner_state":                model.SummarizeOwner(user, owner).OwnerState,
		"owner_status":               ownerStatus,
		"status":                     loginStatus,
		"account_status":             user.Status,
		"login_status":               loginStatus,
		"login_reason":               loginReason,
		"cleanup_status":             cleanupStatus,
		"cleanup_reason":             cleanupReason,
		"current_login_ip":           currentLoginIP,
		"is_activated":               user.IsActivated,
		"can_create_group":           user.CanCreateGroup,
		"globally_searchable":        user.GloballySearchable,
		"community_post_disabled":    user.CommunityPostDisabled,
		"community_comment_disabled": user.CommunityCommentDisabled,
		"saving_post_disabled":       user.CommunityPostDisabled,
		"saving_comment_disabled":    user.CommunityCommentDisabled,
		"ip_whitelist_enabled":       user.IPWhitelistEnabled,
		"ip_whitelist_ip":            strings.TrimSpace(user.IPWhitelistIP),
		"real_name_verified":         realNameVerified,
		"real_name":                  realName,
		"id_number_masked":           idNumberMasked,
		"created_at":                 user.CreatedAt,
	}, nil
}

// UpdateUser 更新用户基本资料。
func UpdateUser(userID, adminID uint, role string, req UpdateUserReq) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return err
	}

	// 更新字段
	var profileItems []tencentim.ProfileItemDetail
	if req.WananID != nil {
		wananID := strings.ToLower(strings.TrimSpace(*req.WananID))
		switch {
		case wananID == "":
			user.WananID = nil
		case !wananIDPattern.MatchString(wananID):
			return errors.New("无忧号需为 6-20 位字母或数字，可含下划线或短横线")
		default:
			phoneExists, err := repo.ExistsByPhone(wananID)
			if err != nil {
				return err
			}
			if phoneExists {
				return errors.New("弥洛蓝序号不能与手机号相同，请更换后重试！")
			}

			currentWananID := optionalString(user.WananID)
			if currentWananID != wananID {
				exists, err := repo.ExistsByWananID(wananID)
				if err != nil {
					return err
				}
				if exists {
					return errors.New("当前弥洛蓝序号已被使用，请更换后重试！")
				}
			}
			user.WananID = &wananID
		}
	}
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
		profileItems = append(profileItems, tencentim.ProfileItemDetail{
			Tag:   tencentim.TagNick,
			Value: *req.Nickname,
		})
	}
	if req.UserTag != nil {
		userTag, err := normalizeUserTag(*req.UserTag)
		if err != nil {
			return err
		}
		user.UserTag = userTag
		profileItems = append(profileItems, tencentim.ProfileItemDetail{
			Tag:   tencentim.TagUserTag,
			Value: userTag,
		})
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
		profileItems = append(profileItems, tencentim.ProfileItemDetail{
			Tag:   tencentim.TagImage,
			Value: *req.AvatarURL,
		})
	}
	if req.SelfSignature != nil {
		user.SelfSignature = *req.SelfSignature
		profileItems = append(profileItems, tencentim.ProfileItemDetail{
			Tag:   tencentim.TagSelfSignature,
			Value: *req.SelfSignature,
		})
	}
	if req.Gender != nil {
		user.Gender = *req.Gender
		profileItems = append(profileItems, tencentim.ProfileItemDetail{
			Tag:   tencentim.TagGender,
			Value: genderTag(*req.Gender),
		})
	}
	if req.Birthday != nil {
		user.Birthday = req.Birthday
		if req.Birthday != nil && !req.Birthday.IsZero() {
			birthdayValue := req.Birthday.Year()*10000 +
				int(req.Birthday.Month())*100 +
				req.Birthday.Day()
			profileItems = append(profileItems, tencentim.ProfileItemDetail{
				Tag:   tencentim.TagBirthDay,
				Value: birthdayValue,
			})
		}
	}
	if req.CanCreateGroup != nil {
		user.CanCreateGroup = *req.CanCreateGroup
	}
	if req.GloballySearchable != nil {
		user.GloballySearchable = *req.GloballySearchable
	}

	if err := repo.Update(user); err != nil {
		return err
	}

	im := tencentim.NewClient()
	if im.IsConfigured() && len(profileItems) > 0 {
		_ = im.PortraitSet(user.IMUserID, profileItems)
	}

	return nil
}

// BatchSetUserCreateGroupPermission 批量更新用户建群权限。
func BatchSetUserCreateGroupPermission(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint, enabled bool) (BatchUserActionResult, error) {
	if len(userIDs) == 0 {
		return BatchUserActionResult{}, errors.New("请选择至少一个用户")
	}

	repo := repository.NewUserRepository()
	return batchProcessUsers(ctx, actorID, isAdmin, userIDs, func(userID uint) error {
		user, err := findManagedUser(actorID, isAdmin, userID)
		if err != nil {
			return err
		}
		if err := ensureUserNotDeleted(user); err != nil {
			return err
		}
		return repo.UpdateFields(userID, map[string]any{
			"can_create_group": enabled,
		})
	}, func() string {
		if enabled {
			return "已允许建群"
		}
		return "已禁止建群"
	}())
}

// BatchSetUserGlobalSearch 批量更新用户全局搜索权限。
func BatchSetUserGlobalSearch(ctx context.Context, actorID uint, isAdmin bool, userIDs []uint, enabled bool) (BatchUserActionResult, error) {
	if len(userIDs) == 0 {
		return BatchUserActionResult{}, errors.New("请选择至少一个用户")
	}

	repo := repository.NewUserRepository()
	return batchProcessUsers(ctx, actorID, isAdmin, userIDs, func(userID uint) error {
		user, err := findManagedUser(actorID, isAdmin, userID)
		if err != nil {
			return err
		}
		if err := ensureUserNotDeleted(user); err != nil {
			return err
		}
		return repo.UpdateFields(userID, map[string]any{
			"globally_searchable": enabled,
		})
	}, func() string {
		if enabled {
			return "已开启全局搜索"
		}
		return "已关闭全局搜索"
	}())
}

// ResetUserPassword 重置用户密码。
func ResetUserPassword(userID, adminID uint, role string, req ResetUserPasswordReq) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}

	password, err := resolveUserResetPassword(req, config.Cfg.Get().App.AdminUserResetPassword)
	if err != nil {
		return err
	}

	hash, err := authwork.GeneratePrivateFromPassword([]byte(password), authwork.DefaultCost)
	if err != nil {
		return err
	}

	if err := repo.UpdateFields(user.ID, map[string]any{
		"password_hash":        string(hash),
		"must_change_password": req.MustChange,
		"auth_version":         gorm.Expr("GREATEST(auth_version, 1) + 1"),
	}); err != nil {
		return err
	}
	return repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID)
}

// UpdateUserPhone 修改用户手机号。
func UpdateUserPhone(userID, adminID uint, role string, newPhone string) error {
	repo := repository.NewUserRepository()
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}

	// 检查新手机号是否已被使用
	existing, _ := repo.FindByPhone(newPhone)
	if existing != nil && existing.ID != userID {
		return errors.New("手机号已被使用")
	}

	user.Phone = newPhone
	return repo.Update(user)
}

// SaveUserIPWhitelist 保存用户登录白名单。
func SaveUserIPWhitelist(userID, adminID uint, role string, req SaveUserIPWhitelistReq) (*UserIPWhitelistResult, error) {
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return nil, err
	}

	enabled := req.Enabled
	whitelistIP := ""
	normalizedIPs := []string(nil)
	if enabled {
		if req.IP != nil {
			whitelistIP = strings.TrimSpace(*req.IP)
		}
		if whitelistIP == "" {
			whitelistIP, err = resolveUserCurrentLoginIP(user.ID)
			if err != nil {
				return nil, err
			}
		}
		if whitelistIP == "" {
			return nil, errors.New("未获取到用户当前登录 IP，请手动填写")
		}
		normalizedIPs, err = security.NormalizeSingleIPList(whitelistIP)
		if err != nil {
			return nil, errors.New("请输入合法的 IP 地址，支持换行、逗号或分号分隔多个 IP")
		}
		whitelistIP = security.JoinSingleIPList(normalizedIPs)
	}

	forcedLogout := false
	err = datasource.Datasource.Mysql.GetClient().Transaction(func(tx *gorm.DB) error {
		fields := map[string]any{
			"ip_whitelist_enabled": enabled,
			"ip_whitelist_ip":      whitelistIP,
		}
		if !enabled {
			fields["ip_whitelist_ip"] = ""
		}

		if err := tx.Model(&model.User{}).Where("id = ?", user.ID).Updates(fields).Error; err != nil {
			return err
		}
		if !enabled {
			return nil
		}

		sessionRepo := repository.NewUserRefreshSessionRepositoryWithDB(tx)
		mismatch, err := hasUserActiveSessionIPMismatch(sessionRepo, user.ID, whitelistIP)
		if err != nil {
			return err
		}
		if !mismatch {
			return nil
		}

		forcedLogout = true
		if err := tx.Model(&model.User{}).
			Where("id = ?", user.ID).
			Update("auth_version", gorm.Expr("GREATEST(auth_version, 1) + 1")).Error; err != nil {
			return err
		}
		return sessionRepo.RevokeAllByUserID(user.ID)
	})
	if err != nil {
		return nil, err
	}

	if forcedLogout {
		kickUserSession(user)
	}

	return &UserIPWhitelistResult{
		Enabled:      enabled,
		IP:           whitelistIP,
		ForcedLogout: forcedLogout,
	}, nil
}

// ResolveUserLoginCooldown 解除用户当前登录冷却。
func ResolveUserLoginCooldown(userID, adminID uint, role string) error {
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return err
	}
	ok, err := repository.NewLoginRiskControlRepository().ResolveAccountCooldownByUser(
		user,
		adminID,
		"用户列表解除登录冷却",
		time.Now(),
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("当前用户未处于登录冷却")
	}
	return nil
}

// BatchSaveUserIPWhitelist 批量保存用户登录白名单。
func BatchSaveUserIPWhitelist(ctx context.Context, adminID uint, role string, userIDs []uint, req SaveUserIPWhitelistReq) (BatchUserIPWhitelistResult, error) {
	_ = ctx
	result := BatchUserIPWhitelistResult{
		Details: make([]BatchUserIPWhitelistDetail, 0, len(userIDs)),
	}
	if len(userIDs) == 0 {
		return result, errors.New("请选择至少一个用户")
	}

	enabled := req.Enabled
	whitelistIP := ""
	if enabled {
		if req.IP != nil {
			whitelistIP = strings.TrimSpace(*req.IP)
		}
		if whitelistIP == "" {
			return result, errors.New("启用白名单后请填写至少 1 个 IP 地址")
		}
		normalizedIPs, err := security.NormalizeSingleIPList(whitelistIP)
		if err != nil {
			return result, errors.New("请输入合法的 IP 地址，支持换行、逗号或分号分隔多个 IP")
		}
		whitelistIP = security.JoinSingleIPList(normalizedIPs)
	}

	isAdmin := HasManagerPrivileges(role)
	seen := make(map[uint]bool, len(userIDs))
	managedUsers := make([]*model.User, 0, len(userIDs))
	managedUserIDs := make([]uint, 0, len(userIDs))
	for _, userID := range userIDs {
		if seen[userID] {
			continue
		}
		seen[userID] = true

		user, err := findManagedUser(adminID, isAdmin, userID)
		if err != nil {
			result.Failed++
			result.Details = append(result.Details, BatchUserIPWhitelistDetail{
				UserID:  userID,
				Success: false,
				Message: err.Error(),
			})
			continue
		}
		if err := ensureUserNotDeleted(user); err != nil {
			result.Failed++
			result.Details = append(result.Details, BatchUserIPWhitelistDetail{
				UserID:   user.ID,
				Nickname: user.Nickname,
				Success:  false,
				Message:  err.Error(),
			})
			continue
		}
		managedUsers = append(managedUsers, user)
		managedUserIDs = append(managedUserIDs, user.ID)
	}

	if len(managedUsers) == 0 {
		return result, nil
	}

	forcedLogoutIDs := map[uint]bool{}
	err := datasource.Datasource.Mysql.GetClient().Transaction(func(tx *gorm.DB) error {
		fields := map[string]any{
			"ip_whitelist_enabled": enabled,
			"ip_whitelist_ip":      whitelistIP,
		}
		if !enabled {
			fields["ip_whitelist_ip"] = ""
		}

		if err := tx.Model(&model.User{}).Where("id IN ?", managedUserIDs).Updates(fields).Error; err != nil {
			return err
		}
		if !enabled {
			return nil
		}

		sessionRepo := repository.NewUserRefreshSessionRepositoryWithDB(tx)
		for _, user := range managedUsers {
			mismatch, err := hasUserActiveSessionIPMismatch(sessionRepo, user.ID, whitelistIP)
			if err != nil {
				return err
			}
			if !mismatch {
				continue
			}
			forcedLogoutIDs[user.ID] = true
			if err := sessionRepo.RevokeAllByUserID(user.ID); err != nil {
				return err
			}
		}
		if len(forcedLogoutIDs) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(forcedLogoutIDs))
		for userID := range forcedLogoutIDs {
			ids = append(ids, userID)
		}
		return tx.Model(&model.User{}).
			Where("id IN ?", ids).
			Update("auth_version", gorm.Expr("GREATEST(auth_version, 1) + 1")).Error
	})
	if err != nil {
		return result, err
	}

	for _, user := range managedUsers {
		forcedLogout := forcedLogoutIDs[user.ID]
		if forcedLogout {
			kickUserSession(user)
			result.ForcedLogout++
		}
		result.Success++
		message := "登录白名单已保存"
		if forcedLogout {
			message = "登录白名单已保存，用户已被强制下线"
		}
		result.Details = append(result.Details, BatchUserIPWhitelistDetail{
			UserID:       user.ID,
			Nickname:     user.Nickname,
			Success:      true,
			Message:      message,
			ForcedLogout: forcedLogout,
		})
	}

	return result, nil
}

// GetUserStatus 获取用户在线状态。
func GetUserStatus(userID, adminID uint, role string) (map[string]interface{}, error) {
	user, err := findManagedUser(adminID, HasManagerPrivileges(role), userID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserNotDeleted(user); err != nil {
		return nil, err
	}

	// 调用腾讯云 IM API 查询在线状态
	status, platform, err := tencentim.QueryUserStatus(user.IMUserID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	onlineIMUserIDs := []string{}
	offlineIMUserIDs := []string{}
	if status == "online" {
		onlineIMUserIDs = append(onlineIMUserIDs, user.IMUserID)
	} else {
		offlineIMUserIDs = append(offlineIMUserIDs, user.IMUserID)
	}
	if err := repository.NewUserRepository().BatchRecordPresenceStatusByIMUserIDs(onlineIMUserIDs, offlineIMUserIDs, now); err != nil {
		logger.Warnf("record presence status after user status refresh failed: im_user_id=%s status=%s err=%v", user.IMUserID, status, err)
	}
	lastOfflineAt := ""
	if refreshedUser, err := repository.NewUserRepository().FindByIMUserID(user.IMUserID); err == nil && refreshedUser.LastOfflineAt != nil {
		lastOfflineAt = refreshedUser.LastOfflineAt.Format("2006-01-02 15:04:05")
	} else if err != nil {
		logger.Warnf("reload user last_offline_at after status refresh failed: im_user_id=%s err=%v", user.IMUserID, err)
	}

	return map[string]interface{}{
		"im_user_id":      user.IMUserID,
		"status":          status,
		"platform":        platform,
		"last_offline_at": lastOfflineAt,
	}, nil
}

// BatchGetUserStatus 批量获取用户在线状态。
func BatchGetUserStatus(imUserIDs []string) ([]map[string]interface{}, error) {
	// 调用腾讯云 IM API 批量查询在线状态
	results, err := tencentim.BatchQueryUserStatus(imUserIDs)
	if err != nil {
		return nil, err
	}
	onlineIMUserIDs := make([]string, 0, len(results))
	offlineIMUserIDs := make([]string, 0, len(results))
	for _, result := range results {
		imUserID, _ := result["im_user_id"].(string)
		status, _ := result["status"].(string)
		if imUserID != "" && status == "online" {
			onlineIMUserIDs = append(onlineIMUserIDs, imUserID)
		} else if imUserID != "" {
			offlineIMUserIDs = append(offlineIMUserIDs, imUserID)
		}
	}
	now := time.Now()
	repo := repository.NewUserRepository()
	if err := repo.BatchRecordPresenceStatusByIMUserIDs(onlineIMUserIDs, offlineIMUserIDs, now); err != nil {
		logger.Warnf("batch record presence status after status refresh failed: online_count=%d offline_count=%d err=%v", len(onlineIMUserIDs), len(offlineIMUserIDs), err)
	}
	users, err := repo.FindByIMUserIDs(imUserIDs)
	if err != nil {
		logger.Warnf("batch load last_offline_at after status refresh failed: count=%d err=%v", len(imUserIDs), err)
		return results, nil
	}
	lastOfflineAtByIMUserID := make(map[string]string, len(users))
	for i := range users {
		if users[i].LastOfflineAt != nil {
			lastOfflineAtByIMUserID[users[i].IMUserID] = users[i].LastOfflineAt.Format("2006-01-02 15:04:05")
		}
	}
	for _, result := range results {
		imUserID, _ := result["im_user_id"].(string)
		result["last_offline_at"] = lastOfflineAtByIMUserID[imUserID]
	}
	return results, nil
}

func resolveUserCurrentLoginIP(userID uint) (string, error) {
	session, err := repository.NewUserRefreshSessionRepository().FindLatestActiveByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(session.IP), nil
}

func hasUserActiveSessionIPMismatch(sessionRepo *repository.UserRefreshSessionRepository, userID uint, whitelistIP string) (bool, error) {
	sessions, err := sessionRepo.ListActiveByUserID(userID)
	if err != nil {
		return false, err
	}
	for _, session := range sessions {
		sessionIP := strings.TrimSpace(session.IP)
		if sessionIP == "" {
			return true, nil
		}

		matched, err := security.MatchSingleIPListText(sessionIP, whitelistIP)
		if err != nil || !matched {
			return true, nil
		}
	}
	return false, nil
}

func kickUserSession(user *model.User) {
	if user == nil {
		return
	}

	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		_ = imClient.AccountKick(user.IMUserID)
	}
}

func normalizeUserTag(raw string) (string, error) {
	userTag := strings.TrimSpace(raw)
	if userTag == "" {
		return "", nil
	}
	if strings.ContainsAny(userTag, "\r\n\t") {
		return "", errors.New("用户标签格式无效")
	}
	if utf8.RuneCountInString(userTag) > 12 {
		return "", errors.New("用户标签最多支持 12 个字")
	}
	return userTag, nil
}

func genderTag(g int8) string {
	switch g {
	case 1:
		return "Gender_Type_Male"
	case 2:
		return "Gender_Type_Female"
	default:
		return "Gender_Type_Unknown"
	}
}

func resolveUserResetPassword(req ResetUserPasswordReq, configured string) (string, error) {
	if req.Mode == "set" {
		if req.NewPassword == nil || *req.NewPassword == "" {
			return "", errors.New("新密码不能为空")
		}
		return *req.NewPassword, nil
	}
	if req.Mode != "default" {
		return "", errors.New("重置方式无效")
	}
	if len(configured) < 6 || len(configured) > 32 {
		return "", errors.New("服务器未配置有效的默认重置密码，请指定新密码或联系运维")
	}
	return configured, nil
}
