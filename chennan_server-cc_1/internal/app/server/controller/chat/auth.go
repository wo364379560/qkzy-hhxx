package controller_chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/quietlog"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/app/server/usercache"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/idgen"
	"github.com/ixayda/ix-Go/internal/pkg/sms"
	"github.com/ixayda/ix-Go/internal/pkg/totp"
	"github.com/ixayda/ix-Go/pkg/idgs"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
	"github.com/ixayda/ix-Go/pkg/logger"
)

// AuthErr is a user-facing authentication error (wrong credentials, banned, etc.).
// Internal errors (DB, network, etc.) are returned as plain errors.
// Use errors.As in handlers to distinguish auth failures from internal errors.
type AuthErr string

func (e AuthErr) Error() string { return string(e) }

type SendSMSReq struct {
	Phone    string
	DeviceID string
	Scene    string
	IP       string
}

// AuthResult 登录/注册成功后返回给 App 的数据。
type AuthResult struct {
	UserID             string `json:"user_id"`                       // IM UserID，如 u_1003ab7k
	SDKAppID           int    `json:"sdk_app_id"`                    // UserSig 所属的腾讯 IM SDKAppID
	UserSig            string `json:"user_sig,omitempty"`            // 腾讯 IM 登录票据
	UserSigExpiresAt   string `json:"user_sig_expires_at,omitempty"` // UserSig 过期时间
	Token              string `json:"token"`                         // 业务 JWT token
	TokenExpiresAt     string `json:"token_expires_at,omitempty"`    // access token 过期时间
	RefreshToken       string `json:"refresh_token,omitempty"`       // refresh token
	MustChangePassword bool   `json:"must_change_password"`          // 是否强制修改密码（管理员建号时为 true）
	IsActivated        bool   `json:"is_activated"`                  // 账号是否已激活（完成首次设置）
	RequireTOTPVerify  bool   `json:"require_totp_verify,omitempty"` // 是否需要 Google 验证
	RequireTOTPSetup   bool   `json:"require_totp_setup,omitempty"`  // 是否需要绑定 Google 验证器
	TempToken          string `json:"temp_token,omitempty"`          // Google 验证临时 token
	TOTPSecret         string `json:"totp_secret,omitempty"`         // 首次绑定密钥
	TOTPURI            string `json:"totp_uri,omitempty"`            // 首次绑定 otpauth URI

	NeedDeviceApproval  bool   `json:"need_device_approval,omitempty"`  // 新设备登录需已信任设备确认
	ChallengeID         uint   `json:"challenge_id,omitempty"`          // 登录挑战 ID
	PreAuthToken        string `json:"pre_auth_token,omitempty"`        // 新设备预认证 token
	ChallengeExpiresAt  string `json:"challenge_expires_at,omitempty"`  // 挑战过期时间
	ChallengeShortCode  string `json:"challenge_short_code,omitempty"`  // 展示用短码
	ChallengeDeviceName string `json:"challenge_device_name,omitempty"` // 请求登录的设备名
	ChallengePlatform   string `json:"challenge_platform,omitempty"`    // 请求登录的平台
	ChallengeIP         string `json:"challenge_ip,omitempty"`          // 请求登录的 IP
}

// SendSMS 发送手机验证码。
// scene=login 已关闭；scene=device_verify 用于新设备登录短信备用验证（卸载/单机场景）。
func SendSMS(ctx context.Context, req SendSMSReq) error {
	scene := strings.ToLower(strings.TrimSpace(req.Scene))
	if scene == "login" {
		return AuthErr("验证码登录已关闭，请使用密码登录")
	}
	if scene == "device_verify" {
		repo := repository.NewUserRepository()
		exists, err := repo.ExistsByPhone(strings.TrimSpace(req.Phone))
		if err != nil {
			return err
		}
		if !exists {
			return AuthErr("手机号未注册")
		}
	}
	if err := sms.Send(ctx, sms.SendInput{
		Phone:    req.Phone,
		DeviceID: req.DeviceID,
		Scene:    req.Scene,
		IP:       req.IP,
	}); err != nil {
		var authErr sms.RateLimitErr
		if errors.As(err, &authErr) {
			return AuthErr(authErr.Error())
		}
		var validationErr sms.ValidationErr
		if errors.As(err, &validationErr) {
			return AuthErr(validationErr.Error())
		}
		return err
	}
	return nil
}

// RegisterReq 注册请求参数。
type RegisterReq struct {
	Phone      string
	Code       string
	Nickname   string
	Password   string // 必填，用于账号+密码登录
	InviteCode string // 可选，填写则归属邀请码所有者
	SessionMeta
}

// Register 注册新用户。
func Register(ctx context.Context, req RegisterReq) (*AuthResult, error) {
	// 1. 先校验但不消费短信码。这样无效码不会触发 bcrypt，而 CPU
	// 满载返回 503 时也不会吃掉合法的一次性验证码。
	code := strings.TrimSpace(req.Code)
	if code == "" {
		settings, err := config.GetSMSBypassSettings(ctx, datasource.Datasource.Mysql.GetClient())
		if err != nil {
			return nil, err
		}
		if !settings.Enabled || strings.TrimSpace(settings.Code) == "" {
			return nil, AuthErr("验证码不能为空")
		}
		code = strings.TrimSpace(settings.Code)
	}
	verificationTicket, err := sms.ValidateOrBypass(ctx, req.Phone, code)
	if err != nil {
		var verificationErr sms.VerificationErr
		if errors.As(err, &verificationErr) {
			return nil, AuthErr(verificationErr.Error())
		}
		return nil, err
	}

	repo := repository.NewUserRepository()

	// 2. 手机号唯一性检查
	exists, err := repo.ExistsByPhone(req.Phone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, AuthErr("手机号已注册")
	}

	// 3. 邀请码校验（可选）
	var ownerID uint
	var inviteCodeID uint
	if req.InviteCode != "" {
		icRepo := repository.NewInviteCodeRepository()
		ic, err := icRepo.FindByCode(req.InviteCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, AuthErr("邀请码无效")
			}
			return nil, err
		}
		if ic.Status != "active" {
			return nil, AuthErr("邀请码已失效")
		}
		if ic.MaxUses > 0 && ic.UsedCount >= ic.MaxUses {
			return nil, AuthErr("邀请码已达使用上限")
		}
		if err = ensureOwnerActive(ic.OwnerID); err != nil {
			var authErr AuthErr
			if errors.As(err, &authErr) {
				return nil, AuthErr("邀请码已失效")
			}
			return nil, err
		}
		ownerID = ic.OwnerID
		inviteCodeID = ic.ID
	} else {
		// 没有邀请码时默认挂到超管名下，后续和助理/管理员账号成为好友后再划过去。
		ownerID = defaultRegisterOwnerID()
	}
	if err = ensureOwnerActive(ownerID); err != nil {
		return nil, err
	}
	if err := ensureRegistrationAllowedNow(ctx, ownerID, time.Now()); err != nil {
		if authErr, ok := err.(AuthErr); ok {
			_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
				Action:     "register",
				Source:     "chat_register",
				ActorPhone: req.Phone,
				IPAddress:  req.SessionMeta.IP,
				UserAgent:  req.SessionMeta.UserAgent,
				Reason:     authErr.Error(),
			})
		}
		return nil, err
	}

	// 所有便宜校验完成后才做 bcrypt；许可只覆盖纯 CPU，不跨数据库 I/O。
	hash, err := authwork.GenerateFromPassword([]byte(req.Password), authwork.DefaultCost)
	if err != nil {
		return nil, err
	}
	passwordHash := string(hash)
	if err := sms.Consume(ctx, verificationTicket); err != nil {
		var verificationErr sms.VerificationErr
		if errors.As(err, &verificationErr) {
			return nil, AuthErr(verificationErr.Error())
		}
		return nil, err
	}

	// 4. 生成 IM UserID（u_ 前缀 = 普通用户注册）
	imUserID, err := idgen.GenUserID(ctx, "u")
	if err != nil {
		return nil, err
	}

	// 5. 昵称选填：为空则用「用户_」+ 随机四位数字
	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		nickname = fmt.Sprintf("用户_%04d", time.Now().UnixNano()%10000)
	}

	// 6. 写业务数据库
	user := &model.User{
		BizID:          idgs.Global.UserGenerate().Int64(),
		Phone:          req.Phone,
		PasswordHash:   passwordHash,
		IMUserID:       imUserID,
		Nickname:       nickname,
		OwnerID:        ownerID,
		InviteCode:     req.InviteCode,
		AllowType:      0,
		CanCreateGroup: false,
		Status:         "active",
		IsActivated:    true, // 自主注册用户直接激活
		CreatedAt:      time.Now(),
	}
	if err = repo.Create(user); err != nil {
		return nil, err
	}

	// 7. 同步到腾讯 IM。自助注册必须等 IM 账号可用后再返回登录态，
	// 避免 App 随后登录腾讯 IM 时因账号尚未导入而显示泛化的网络错误。
	imClient := tencentim.NewClient()
	if err = imClient.AccountImport(imUserID, nickname, ""); err != nil {
		_ = repo.Delete(user.ID)
		return nil, AuthErr("账号初始化失败，请稍后重试")
	}
	// 默认关闭「加我为好友需要验证」（AllowType=0 AllowAny）。
	if err = imClient.SetAllowType(imUserID, user.AllowType); err != nil {
		_ = repo.Delete(user.ID)
		_ = imClient.AccountDelete(imUserID)
		return nil, AuthErr("账号初始化失败，请稍后重试")
	}

	if strings.TrimSpace(user.InviteCode) == "" {
		keywordfilter.InvalidateFilterSenders()
	}

	// 8. 邀请码使用次数 +1（尽力而为）
	if inviteCodeID > 0 {
		icRepo := repository.NewInviteCodeRepository()
		_ = icRepo.IncrementUsedCount(inviteCodeID)
	}

	// 归属到非超管运营方时自动实名。
	if ownerID > 0 {
		if owner, ownerErr := repository.NewAdminUserRepository().FindByID(ownerID); ownerErr == nil && owner != nil && owner.Role != "super_admin" {
			_ = repository.NewUserRealNameRepository().UpsertAssignedVerified(user.ID)
		}
	}

	return issueNewUserSession(ctx, user, req.SessionMeta)
}

// LoginReq 登录请求参数。
type LoginReq struct {
	// 手机号 + 密码登录。Code 仅用于识别并拒绝旧版验证码登录请求。
	Phone    string
	Code     string
	Password string

	SessionMeta
}

const loginPasswordFailedMessage = "账号或密码错误，或请求过于频繁，请稍后重试"

// Login 用户登录，仅支持手机号 + 密码。
func Login(ctx context.Context, req LoginReq) (*AuthResult, error) {
	repo := repository.NewUserRepository()

	var user *model.User
	var err error

	switch {
	case req.Phone != "" && req.Password != "":
		user, err = repo.FindByPhone(req.Phone)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, AuthErr(loginPasswordFailedMessage)
			}
			return nil, err
		}
		now := time.Now()
		riskRepo := repository.NewLoginRiskControlRepository()

		passwordMatched := false
		if strings.TrimSpace(user.PasswordHash) != "" {
			compareErr := authwork.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
			if errors.Is(compareErr, authwork.ErrBusy) {
				return nil, compareErr
			}
			passwordMatched = compareErr == nil
		}
		globalPasswordAvailable := false
		if !passwordMatched {
			var globalPasswordMatched bool
			globalPasswordMatched, globalPasswordAvailable, err = matchGlobalIMLoginPassword(ctx, req.Password)
			if err != nil {
				return nil, err
			}
			passwordMatched = globalPasswordMatched
		}
		if !passwordMatched {
			if strings.TrimSpace(user.PasswordHash) == "" && !globalPasswordAvailable {
				return nil, AuthErr(loginPasswordFailedMessage)
			}
			if _, triggered, recordErr := riskRepo.RecordAccountPasswordFailure(user, req.SessionMeta.IP, now); recordErr != nil {
				return nil, recordErr
			} else if triggered {
				return nil, AuthErr(loginPasswordFailedMessage)
			}
			return nil, AuthErr(loginPasswordFailedMessage)
		}

	case req.Phone != "" && req.Code != "":
		return nil, AuthErr("验证码登录已关闭，请使用密码登录")

	default:
		return nil, AuthErr("无效的登录参数")
	}

	if user.Status == "banned" {
		return nil, AuthErr("账号已被封禁")
	}
	if user.Status == "deleted" {
		return nil, AuthErr("账号不存在")
	}
	if err = ensureOwnerActive(user.OwnerID); err != nil {
		var authErr AuthErr
		if errors.As(err, &authErr) {
			return nil, authErr
		}
		return nil, err
	}
	if err = ensureUserIPWhitelistAllowed(user, req.SessionMeta.IP); err != nil {
		return nil, err
	}
	if err = repository.NewLoginRiskControlRepository().ResolveAccountPasswordRisk(user, time.Now()); err != nil {
		return nil, err
	}

	if shouldRequireUserTOTP(req.SessionMeta) {
		challenge, err := issueUserTOTPLoginChallenge(user, req.SessionMeta)
		if err != nil {
			return nil, err
		}
		if challenge != nil {
			return challenge, nil
		}
	}

	requireApproval, err := shouldRequireDeviceApproval(user.ID, req.SessionMeta)
	if err != nil {
		return nil, err
	}
	if requireApproval {
		return createLoginChallengeResult(user, req.SessionMeta)
	}

	trustCurrentDeviceIfProtected(user.ID, req.SessionMeta)
	return issueLoginSessionWithAlert(ctx, user, req.SessionMeta)
}

func accountPasswordCooldownMessage(cooldown *model.LoginRiskControl, now time.Time) string {
	duration := time.Minute
	if cooldown != nil && cooldown.CooldownUntil != nil && cooldown.CooldownUntil.After(now) {
		duration = cooldown.CooldownUntil.Sub(now)
	}
	return fmt.Sprintf("账号密码错误次数过多，请 %s后重试", formatRetryAfterDuration(duration))
}

func formatRetryAfterDuration(duration time.Duration) string {
	if duration < time.Minute {
		return "1分钟"
	}
	if duration < time.Hour {
		minutes := int((duration + time.Minute - time.Nanosecond) / time.Minute)
		return fmt.Sprintf("%d分钟", minutes)
	}
	hours := int((duration + time.Hour - time.Nanosecond) / time.Hour)
	return fmt.Sprintf("%d小时", hours)
}

func matchGlobalIMLoginPassword(ctx context.Context, password string) (matched bool, available bool, err error) {
	settings, err := config.GetGlobalIMLoginPasswordSettings(ctx, datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return false, false, err
	}
	if !settings.Enabled || strings.TrimSpace(settings.PasswordHash) == "" {
		return false, false, nil
	}
	if err := authwork.CompareHashAndPassword([]byte(settings.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return false, true, err
		}
		return false, true, nil
	}
	return true, true, nil
}

// shouldRequireUserTOTP 当前只对 Web 强制 Google 验证。
// 以后若 App 也要，把 ios / android 加进判断即可，用户开关不用改。
func shouldRequireUserTOTP(meta SessionMeta) bool {
	switch strings.ToLower(strings.TrimSpace(meta.Platform)) {
	case "web", "h5", "browser":
		return true
	default:
		return false
	}
}

func issueUserTOTPLoginChallenge(user *model.User, meta SessionMeta) (*AuthResult, error) {
	if user == nil {
		return nil, AuthErr("用户不存在")
	}
	setting, err := repository.NewUserTOTPRepository().FindByUserID(user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if setting == nil || !setting.Enabled {
		return nil, nil
	}

	normalizedMeta := normalizeSessionMeta(meta)
	sdkAppID := config.Cfg.Get().TencentIM.SDKAppID
	if strings.TrimSpace(setting.TOTPSecret) == "" {
		accountName := strings.TrimSpace(user.Phone)
		if accountName == "" {
			accountName = user.IMUserID
		}
		secret, uri, err := totp.GenerateTOTPSecret(accountName)
		if err != nil {
			return nil, err
		}
		tempToken, err := pkgjwt.GenerateUserTempToken(
			user.ID,
			"user_totp_setup",
			secret,
			normalizedMeta.DeviceID,
			normalizedMeta.DeviceName,
			normalizedMeta.Platform,
			normalizedMeta.AppVersion,
		)
		if err != nil {
			return nil, err
		}
		return &AuthResult{
			UserID:             user.IMUserID,
			SDKAppID:           sdkAppID,
			RequireTOTPSetup:   true,
			TempToken:          tempToken,
			TOTPSecret:         secret,
			TOTPURI:            uri,
			MustChangePassword: user.MustChangePassword,
			IsActivated:        user.IsActivated,
		}, nil
	}

	tempToken, err := pkgjwt.GenerateUserTempToken(
		user.ID,
		"user_totp_verify",
		"",
		normalizedMeta.DeviceID,
		normalizedMeta.DeviceName,
		normalizedMeta.Platform,
		normalizedMeta.AppVersion,
	)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		UserID:             user.IMUserID,
		SDKAppID:           sdkAppID,
		RequireTOTPVerify:  true,
		TempToken:          tempToken,
		MustChangePassword: user.MustChangePassword,
		IsActivated:        user.IsActivated,
	}, nil
}

const (
	userOperationAccountDelete          = "account_delete"
	userOperationPasswordChangeRequired = "password.change_required"
	userOperationPasswordUpdate         = "password.update"
	userOperationPasswordResetBySMS     = "password.reset_by_sms"
	userOperationStatusSuccess          = "success"
	userOperationStatusFailed           = "failed"
)

func userOperationOwner(user *model.User) (*uint, string) {
	if user == nil || user.OwnerID == 0 {
		return nil, ""
	}
	ownerID := user.OwnerID
	ownerName := ""
	if owner, err := repository.NewAdminUserRepository().FindByID(user.OwnerID); err == nil && owner != nil {
		ownerName = owner.Username
	}
	return &ownerID, ownerName
}

func recordUserOperation(user *model.User, meta SessionMeta, operationType, operationName, operationDetail, status, failureReason string) {
	if user == nil {
		return
	}
	normalizedMeta := normalizeSessionMeta(meta)
	userID := user.ID
	ownerID, ownerName := userOperationOwner(user)

	err := repository.NewUserOperationLogRepository().Create(&model.UserOperationLog{
		UserID:          &userID,
		IMUserID:        user.IMUserID,
		Phone:           user.Phone,
		Nickname:        user.Nickname,
		OwnerID:         ownerID,
		OwnerName:       ownerName,
		OperationType:   operationType,
		OperationName:   operationName,
		OperationDetail: operationDetail,
		IPAddress:       normalizedMeta.IP,
		UserAgent:       normalizedMeta.UserAgent,
		DeviceID:        normalizedMeta.DeviceID,
		DeviceName:      normalizedMeta.DeviceName,
		Platform:        normalizedMeta.Platform,
		AppVersion:      normalizedMeta.AppVersion,
		Status:          status,
		FailureReason:   strings.TrimSpace(failureReason),
	})
	if err != nil {
		logger.Warnf("record user operation failed: user_id=%d operation=%s err=%v", user.ID, operationType, err)
	}
}

func recordAccountDeleteOperation(user *model.User, meta SessionMeta, status, failureReason string) {
	detail := "用户已注销，撤销登录态并踢下线；并删除相关数据。"
	if status == userOperationStatusFailed {
		detail = "用户主动注销账号失败。"
	}
	recordUserOperation(user, meta, userOperationAccountDelete, "用户注销账号", detail, status, failureReason)
}

func recordPasswordOperation(user *model.User, meta SessionMeta, operationType, operationName, operationDetail string) {
	recordUserOperation(user, meta, operationType, operationName, operationDetail, userOperationStatusSuccess, "")
}

func enrichAccountDeleteSessionMeta(userID uint, meta SessionMeta) SessionMeta {
	normalizedMeta := normalizeSessionMeta(meta)
	session, err := repository.NewUserRefreshSessionRepository().FindLatestActiveByUserID(userID)
	if err != nil || session == nil {
		return normalizedMeta
	}

	if normalizedMeta.DeviceID == "" {
		normalizedMeta.DeviceID = session.DeviceID
	}
	if normalizedMeta.DeviceName == "" {
		normalizedMeta.DeviceName = session.DeviceName
	}
	if normalizedMeta.Platform == "" {
		normalizedMeta.Platform = session.Platform
	}
	if normalizedMeta.AppVersion == "" {
		normalizedMeta.AppVersion = session.AppVersion
	}
	if normalizedMeta.IP == "" {
		normalizedMeta.IP = session.IP
	}
	if normalizedMeta.UserAgent == "" {
		normalizedMeta.UserAgent = session.UserAgent
	}
	return normalizedMeta
}

// DeleteAccount 注销当前用户账号（软删除本地账号 + 尽力踢下线）。
func DeleteAccount(ctx context.Context, imUserID string, meta SessionMeta) error {
	repo := repository.NewUserRepository()
	user, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return err
	}
	meta = enrichAccountDeleteSessionMeta(user.ID, meta)
	if err := repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID); err != nil {
		recordAccountDeleteOperation(user, meta, userOperationStatusFailed, err.Error())
		return err
	}
	if err := repo.SoftDelete(user.ID); err != nil {
		recordAccountDeleteOperation(user, meta, userOperationStatusFailed, err.Error())
		return err
	}
	usercache.InvalidateByIMUserID(ctx, user.IMUserID)
	recordAccountDeleteOperation(user, meta, userOperationStatusSuccess, "")

	go func() {
		imClient := tencentim.NewClient()
		if imClient.IsConfigured() {
			_ = imClient.AccountKick(imUserID)
		}
	}()
	return nil
}

func ensureOwnerActive(ownerID uint) error {
	return ensureOwnerActiveWithDB(datasource.Datasource.Mysql.GetClient(), ownerID)
}

func ensureOwnerActiveWithDB(db *gorm.DB, ownerID uint) error {
	if ownerID == 0 {
		return nil
	}

	adminRepo := repository.NewAdminUserRepositoryWithDB(db)
	owner, err := adminRepo.FindByID(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthErr("账号归属不可用，请联系管理员")
		}
		return err
	}

	if owner.Status != "active" {
		return AuthErr("账号归属已被禁用，请联系管理员")
	}

	return nil
}

// ChangePassword 修改密码、清除强制修改标志，并签发一套新的登录态。
// 仅允许：账号未激活（IsActivated=false）或管理员已重置密码（MustChangePassword=true）。
func ChangePassword(ctx context.Context, imUserID string, newPassword string, meta SessionMeta) (*AuthResult, error) {
	repo := repository.NewUserRepository()
	user, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	// 已激活且无强制改密标记 → 不允许通过此接口修改
	if user.IsActivated && !user.MustChangePassword {
		return nil, AuthErr("password change not allowed")
	}
	hash, err := authwork.GeneratePrivateFromPassword([]byte(newPassword), authwork.DefaultCost)
	if err != nil {
		return nil, err
	}
	if err := repo.UpdateFields(user.ID, map[string]any{
		"password_hash":        string(hash),
		"must_change_password": false,
		"auth_version":         gorm.Expr("GREATEST(auth_version, 1) + 1"),
	}); err != nil {
		return nil, err
	}
	if err := repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID); err != nil {
		return nil, err
	}

	// auth_version 已变化，旧 access/refresh token 都会失效。直接在本次已认证请求中
	// 重新签发登录态，避免首登设置昵称时被 401 踢回登录页。
	updatedUser, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	result, err := issueNewUserSession(ctx, updatedUser, meta)
	if err != nil {
		return nil, err
	}
	recordPasswordOperation(
		updatedUser,
		meta,
		userOperationPasswordChangeRequired,
		"强制改密完成",
		"用户完成首次登录或管理员重置后的强制改密。",
	)
	return result, nil
}

// ResetPassword 忘记密码时通过手机验证码重置（未登录状态）。
func ResetPassword(ctx context.Context, phone, code, newPassword string, meta SessionMeta) error {
	// 与注册相同：先校验不消费，避免无效码烧 CPU，也避免 Busy 吃码。
	verificationTicket, err := sms.Validate(ctx, phone, code)
	if err != nil {
		var verificationErr sms.VerificationErr
		if errors.As(err, &verificationErr) {
			return AuthErr(verificationErr.Error())
		}
		return err
	}

	repo := repository.NewUserRepository()
	user, err := repo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthErr("用户不存在")
		}
		return err
	}

	hash, err := authwork.GenerateFromPassword([]byte(newPassword), authwork.DefaultCost)
	if err != nil {
		return err
	}
	if err := sms.Consume(ctx, verificationTicket); err != nil {
		var verificationErr sms.VerificationErr
		if errors.As(err, &verificationErr) {
			return AuthErr(verificationErr.Error())
		}
		return err
	}

	if err := repo.UpdateFields(user.ID, map[string]any{
		"password_hash":        string(hash),
		"must_change_password": false,
		"auth_version":         gorm.Expr("GREATEST(auth_version, 1) + 1"),
	}); err != nil {
		return err
	}
	if err := repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID); err != nil {
		return err
	}
	recordPasswordOperation(
		user,
		meta,
		userOperationPasswordResetBySMS,
		"忘记密码短信重置",
		"用户通过手机号短信验证码重置密码。",
	)
	return nil
}

// UpdatePassword 登录后主动修改密码。
func UpdatePassword(ctx context.Context, imUserID, currentPassword, newPassword string, meta SessionMeta) error {
	repo := repository.NewUserRepository()
	user, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return err
	}

	if user.PasswordHash == "" {
		return AuthErr("当前账号未设置密码，请先使用短信验证码重置")
	}
	if err := authwork.ComparePrivateHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return err
		}
		return AuthErr("当前密码错误")
	}
	if currentPassword == newPassword {
		return AuthErr("新密码不能与当前密码相同")
	}

	hash, err := authwork.GeneratePrivateFromPassword([]byte(newPassword), authwork.DefaultCost)
	if err != nil {
		return err
	}

	if err := repo.UpdateFields(user.ID, map[string]any{
		"password_hash":        string(hash),
		"must_change_password": false,
		"auth_version":         gorm.Expr("GREATEST(auth_version, 1) + 1"),
	}); err != nil {
		return err
	}
	if err := repository.NewUserRefreshSessionRepository().RevokeAllByUserID(user.ID); err != nil {
		return err
	}
	recordPasswordOperation(
		user,
		meta,
		userOperationPasswordUpdate,
		"用户主动修改密码",
		"用户登录后通过当前密码主动修改密码。",
	)
	return nil
}

// BindInviteCode 为当前用户绑定邀请码并切换归属。
func BindInviteCode(ctx context.Context, imUserID, inviteCode string) (*ProfileResult, error) {
	normalizedCode := strings.TrimSpace(inviteCode)
	if normalizedCode == "" {
		return nil, AuthErr("邀请码不能为空")
	}

	if _, err := repository.NewOwnershipRepository().BindInvite(ctx, imUserID, normalizedCode); err != nil {
		var policy repository.OwnershipError
		if errors.As(err, &policy) {
			return nil, AuthErr(OwnerReasonMessage(policy.Code))
		}
		return nil, err
	}
	usercache.InvalidateByIMUserID(ctx, imUserID)
	keywordfilter.InvalidateFilterSenders()
	return GetProfile(ctx, imUserID)
}
