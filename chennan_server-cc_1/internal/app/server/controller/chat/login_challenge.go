package controller_chat

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/sms"
)

const loginChallengeInvalidMessage = "登录授权已失效，请重新登录"

// LoginChallengeItem 待确认登录挑战摘要。
type LoginChallengeItem struct {
	ID         uint   `json:"id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
	IP         string `json:"ip"`
	ShortCode  string `json:"short_code"`
	Status     string `json:"status"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
}

// LoginChallengeStatusResult 新设备轮询挑战状态。
type LoginChallengeStatusResult struct {
	ID        uint   `json:"id"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
}

func shouldRequireDeviceApproval(userID uint, meta SessionMeta) (bool, error) {
	normalized := normalizeSessionMeta(meta)
	// Web 登录只走 Google 验证，不受 App「新设备登录验证」约束。
	if isWebDevice(normalized.Platform, normalized.DeviceName) {
		return false, nil
	}
	deviceID := normalizeDeviceID(normalized.DeviceID)
	if deviceID == "" {
		return false, nil
	}

	settings, err := repository.NewUserSecuritySettingsRepository().GetOrDefault(userID)
	if err != nil {
		return false, err
	}
	if !settings.LoginVerification {
		return false, nil
	}

	trusted, err := repository.NewUserTrustedDeviceRepository().Exists(userID, deviceID)
	if err != nil {
		return false, err
	}
	if trusted {
		return false, nil
	}

	sessionRepo := repository.NewUserRefreshSessionRepository()
	hasOwn, err := sessionRepo.HasActiveSessionForDevice(userID, deviceID)
	if err != nil {
		return false, err
	}
	if hasOwn {
		return false, nil
	}

	hasOther, err := sessionRepo.HasActiveSessionOnOtherDevice(userID, deviceID)
	if err != nil {
		return false, err
	}
	return hasOther, nil
}

// trustCurrentDeviceIfProtected 免挑战登录成功后，把本机记入信任列表。
// 「只信任一次」走挑战兑换路径，不会调用此函数。
func trustCurrentDeviceIfProtected(userID uint, meta SessionMeta) {
	normalized := normalizeSessionMeta(meta)
	if isWebDevice(normalized.Platform, normalized.DeviceName) {
		return
	}
	settings, err := repository.NewUserSecuritySettingsRepository().GetOrDefault(userID)
	if err != nil || !settings.LoginVerification {
		return
	}
	if normalized.DeviceID == "" {
		return
	}
	_ = repository.NewUserTrustedDeviceRepository().Upsert(
		userID,
		normalized.DeviceID,
		normalized.DeviceName,
		normalized.Platform,
	)
}

func createLoginChallengeResult(user *model.User, meta SessionMeta) (*AuthResult, error) {
	normalizedMeta := normalizeSessionMeta(meta)
	preAuthToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(repository.LoginChallengeTTL())
	shortCode, err := generateLoginChallengeShortCode()
	if err != nil {
		return nil, err
	}

	challenge := &model.UserLoginChallenge{
		UserID:           user.ID,
		DeviceID:         normalizedMeta.DeviceID,
		DeviceName:       normalizedMeta.DeviceName,
		Platform:         normalizedMeta.Platform,
		IP:               normalizedMeta.IP,
		UserAgent:        normalizedMeta.UserAgent,
		Status:           model.UserLoginChallengeStatusPending,
		PreAuthTokenHash: security.HashRefreshToken(preAuthToken),
		ShortCode:        shortCode,
		ExpiresAt:        expiresAt,
	}
	if err := repository.NewUserLoginChallengeRepository().Create(challenge); err != nil {
		return nil, err
	}

	NotifyLoginChallengePending(user, challenge)

	return &AuthResult{
		UserID:              user.IMUserID,
		SDKAppID:            config.Cfg.Get().TencentIM.SDKAppID,
		NeedDeviceApproval:  true,
		ChallengeID:         challenge.ID,
		PreAuthToken:        preAuthToken,
		ChallengeExpiresAt:  formatRFC3339(expiresAt),
		ChallengeShortCode:  shortCode,
		ChallengeDeviceName: deviceDisplayName(normalizedMeta.DeviceName, normalizedMeta.Platform),
		ChallengePlatform:   normalizedMeta.Platform,
		ChallengeIP:         normalizedMeta.IP,
		MustChangePassword:  user.MustChangePassword,
		IsActivated:         user.IsActivated,
	}, nil
}

func generateLoginChallengeShortCode() (string, error) {
	var parts [2]int64
	for i := range parts {
		n, err := rand.Int(rand.Reader, big.NewInt(1000))
		if err != nil {
			return "", err
		}
		parts[i] = n.Int64()
	}
	return fmt.Sprintf("%03d %03d", parts[0], parts[1]), nil
}

func normalizeLoginChallenge(challenge *model.UserLoginChallenge, now time.Time) string {
	if challenge == nil {
		return model.UserLoginChallengeStatusExpired
	}
	status := strings.TrimSpace(challenge.Status)
	if status == model.UserLoginChallengeStatusPending && !challenge.ExpiresAt.After(now) {
		_ = repository.NewUserLoginChallengeRepository().MarkExpired(challenge.ID, now)
		return model.UserLoginChallengeStatusExpired
	}
	return status
}

// ListPendingLoginChallenges 返回当前用户待确认登录挑战。
func ListPendingLoginChallenges(imUserID string) ([]LoginChallengeItem, error) {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	_ = repository.NewUserLoginChallengeRepository().ExpireStalePending(time.Now())

	items, err := repository.NewUserLoginChallengeRepository().ListPendingByUserID(user.ID)
	if err != nil {
		return nil, err
	}

	result := make([]LoginChallengeItem, 0, len(items))
	for _, item := range items {
		if isWebDevice(item.Platform, item.DeviceName) {
			continue
		}
		result = append(result, loginChallengeItemFromModel(item))
	}
	return result, nil
}

func loginChallengeItemFromModel(item model.UserLoginChallenge) LoginChallengeItem {
	return LoginChallengeItem{
		ID:         item.ID,
		DeviceID:   item.DeviceID,
		DeviceName: deviceDisplayName(item.DeviceName, item.Platform),
		Platform:   item.Platform,
		IP:         item.IP,
		ShortCode:  item.ShortCode,
		Status:     item.Status,
		ExpiresAt:  formatRFC3339(item.ExpiresAt),
		CreatedAt:  formatRFC3339(item.CreatedAt),
	}
}

// GetLoginChallengeStatus 新设备轮询挑战状态。
func GetLoginChallengeStatus(challengeID uint, preAuthToken string) (*LoginChallengeStatusResult, error) {
	preAuthToken = security.NormalizeRefreshToken(preAuthToken)
	if preAuthToken == "" {
		return nil, AuthErr(loginChallengeInvalidMessage)
	}

	challenge, err := repository.NewUserLoginChallengeRepository().FindByIDAndTokenHash(
		challengeID,
		security.HashRefreshToken(preAuthToken),
	)
	if err != nil {
		return nil, AuthErr(loginChallengeInvalidMessage)
	}

	now := time.Now()
	status := normalizeLoginChallenge(challenge, now)
	return &LoginChallengeStatusResult{
		ID:        challenge.ID,
		Status:    status,
		ExpiresAt: formatRFC3339(challenge.ExpiresAt),
	}, nil
}

// ApproveLoginChallenge 已登录设备批准新设备登录。
// trustDevice=true 表示「信任登录」（永久信任该设备）；false 表示「只信任一次」。
func ApproveLoginChallenge(imUserID string, challengeID uint, approverDeviceID string, trustDevice bool) error {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return err
	}
	approverDeviceID = normalizeDeviceID(approverDeviceID)
	if approverDeviceID == "" {
		return AuthErr("无法识别当前设备，请重新登录后再试")
	}

	challenge, err := repository.NewUserLoginChallengeRepository().FindByID(challengeID)
	if err != nil {
		return AuthErr(loginChallengeInvalidMessage)
	}
	if challenge.UserID != user.ID {
		return AuthErr(loginChallengeInvalidMessage)
	}
	if challenge.DeviceID == approverDeviceID {
		return AuthErr("不能批准本机发起的登录请求")
	}

	now := time.Now()
	status := normalizeLoginChallenge(challenge, now)
	if status != model.UserLoginChallengeStatusPending {
		return AuthErr("该登录请求已处理或已过期")
	}

	ok, err := repository.NewUserLoginChallengeRepository().Approve(challengeID, approverDeviceID, now)
	if err != nil {
		return err
	}
	if !ok {
		return AuthErr("该登录请求已处理或已过期")
	}
	if trustDevice {
		_ = repository.NewUserTrustedDeviceRepository().Upsert(
			user.ID,
			challenge.DeviceID,
			challenge.DeviceName,
			challenge.Platform,
		)
	}
	return nil
}

// DenyLoginChallenge 已登录设备拒绝新设备登录。
func DenyLoginChallenge(imUserID string, challengeID uint, approverDeviceID string) error {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return err
	}
	approverDeviceID = normalizeDeviceID(approverDeviceID)
	if approverDeviceID == "" {
		return AuthErr("无法识别当前设备，请重新登录后再试")
	}

	challenge, err := repository.NewUserLoginChallengeRepository().FindByID(challengeID)
	if err != nil {
		return AuthErr(loginChallengeInvalidMessage)
	}
	if challenge.UserID != user.ID {
		return AuthErr(loginChallengeInvalidMessage)
	}
	if challenge.DeviceID == approverDeviceID {
		return AuthErr("不能拒绝本机发起的登录请求")
	}

	now := time.Now()
	status := normalizeLoginChallenge(challenge, now)
	if status != model.UserLoginChallengeStatusPending {
		return AuthErr("该登录请求已处理或已过期")
	}

	ok, err := repository.NewUserLoginChallengeRepository().Deny(challengeID, approverDeviceID, now)
	if err != nil {
		return err
	}
	if !ok {
		return AuthErr("该登录请求已处理或已过期")
	}
	return nil
}

// ApproveLoginChallengeBySMS 无已登录设备可确认时，用账号绑定手机号短信验证码批准并信任该设备。
// 适用于卸载重装、仅一台手机等场景。
func ApproveLoginChallengeBySMS(ctx context.Context, challengeID uint, preAuthToken, code string) error {
	preAuthToken = security.NormalizeRefreshToken(preAuthToken)
	code = strings.TrimSpace(code)
	if preAuthToken == "" || code == "" {
		return AuthErr(loginChallengeInvalidMessage)
	}

	challenge, err := repository.NewUserLoginChallengeRepository().FindByIDAndTokenHash(
		challengeID,
		security.HashRefreshToken(preAuthToken),
	)
	if err != nil {
		return AuthErr(loginChallengeInvalidMessage)
	}

	now := time.Now()
	status := normalizeLoginChallenge(challenge, now)
	if status != model.UserLoginChallengeStatusPending {
		return AuthErr("该登录请求已处理或已过期")
	}

	user, err := repository.NewUserRepository().FindByID(challenge.UserID)
	if err != nil {
		return err
	}
	phone := strings.TrimSpace(user.Phone)
	if phone == "" {
		return AuthErr("账号未绑定手机号，无法短信验证")
	}

	if err := sms.VerifyOrBypass(ctx, phone, code); err != nil {
		return AuthErr(err.Error())
	}

	ok, err := repository.NewUserLoginChallengeRepository().Approve(challengeID, "sms:"+phone, now)
	if err != nil {
		return err
	}
	if !ok {
		return AuthErr("该登录请求已处理或已过期")
	}
	_ = repository.NewUserTrustedDeviceRepository().Upsert(
		user.ID,
		challenge.DeviceID,
		challenge.DeviceName,
		challenge.Platform,
	)
	return nil
}

// ExchangeLoginChallenge 新设备在批准后兑换正式会话。
func ExchangeLoginChallenge(ctx context.Context, challengeID uint, preAuthToken string) (*AuthResult, error) {
	preAuthToken = security.NormalizeRefreshToken(preAuthToken)
	if preAuthToken == "" {
		return nil, AuthErr(loginChallengeInvalidMessage)
	}

	challenge, err := repository.NewUserLoginChallengeRepository().FindByIDAndTokenHash(
		challengeID,
		security.HashRefreshToken(preAuthToken),
	)
	if err != nil {
		return nil, AuthErr(loginChallengeInvalidMessage)
	}

	now := time.Now()
	status := normalizeLoginChallenge(challenge, now)
	switch status {
	case model.UserLoginChallengeStatusApproved:
	case model.UserLoginChallengeStatusPending:
		return nil, AuthErr("请先在已登录设备上确认此次登录")
	case model.UserLoginChallengeStatusDenied:
		return nil, AuthErr("登录请求已被拒绝")
	default:
		return nil, AuthErr(loginChallengeInvalidMessage)
	}

	ok, err := repository.NewUserLoginChallengeRepository().Consume(challengeID, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, AuthErr(loginChallengeInvalidMessage)
	}

	user, err := repository.NewUserRepository().FindByID(challenge.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status == "banned" || user.Status == "deleted" {
		return nil, AuthErr("账号已失效，请重新登录")
	}

	meta := SessionMeta{
		DeviceID:   challenge.DeviceID,
		DeviceName: challenge.DeviceName,
		Platform:   challenge.Platform,
		IP:         challenge.IP,
		UserAgent:  challenge.UserAgent,
	}
	result, err := issueNewUserSession(ctx, user, meta)
	if err != nil {
		return nil, err
	}
	NotifyNewDeviceApproved(user, meta)
	return result, nil
}
