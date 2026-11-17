package handler_chat

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type loginChallengeExchangeReq struct {
	PreAuthToken string `json:"pre_auth_token" binding:"required,max=256"`
}

type loginChallengeSMSVerifyReq struct {
	PreAuthToken string `json:"pre_auth_token" binding:"required,max=256"`
	Code         string `json:"code" binding:"required,max=10"`
}

type loginChallengeActionReq struct {
	CurrentDeviceID string `json:"current_device_id" binding:"omitempty,max=128"`
	TrustDevice     *bool  `json:"trust_device"`
}

// ListPendingLoginChallenges 获取待确认的新设备登录请求。
func ListPendingLoginChallenges(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	items, err := controller_chat.ListPendingLoginChallenges(imUserID)
	if err != nil {
		c.Error(err)
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"items": items})
}

// GetLoginChallengeStatus 新设备轮询挑战状态。
func GetLoginChallengeStatus(c *gin.Context) {
	challengeID, err := parseLoginChallengeID(c.Param("id"))
	if err != nil {
		response.InvalidParams(c, "无效的挑战 ID")
		return
	}
	preAuthToken := strings.TrimSpace(c.Query("pre_auth_token"))
	if preAuthToken == "" || len(preAuthToken) > 256 {
		response.InvalidParams(c, "无效的授权令牌")
		return
	}
	if !authlimit.Enforce(c, "login_challenge_status", []ratelimit.SubjectRules{
		{Subject: loginChallengeRateSubject(challengeID), Rules: ratelimit.PublicChallengeStatusSubjectRules},
	}) {
		return
	}
	result, err := controller_chat.GetLoginChallengeStatus(challengeID, preAuthToken)
	if err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, result)
}

// ApproveLoginChallenge 已登录设备批准新设备登录。
func ApproveLoginChallenge(c *gin.Context) {
	challengeID, err := parseLoginChallengeID(c.Param("id"))
	if err != nil {
		response.InvalidParams(c, "无效的挑战 ID")
		return
	}

	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req loginChallengeActionReq
	_ = c.ShouldBindJSON(&req)

	approverDeviceID := normalizeDeviceIDForHandler(req.CurrentDeviceID)
	if approverDeviceID == "" {
		approverDeviceID = resolveCurrentDeviceID(c, imUserID)
	}

	trustDevice := true
	if req.TrustDevice != nil {
		trustDevice = *req.TrustDevice
	}

	if err := controller_chat.ApproveLoginChallenge(imUserID, challengeID, approverDeviceID, trustDevice); err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, nil)
}

// DenyLoginChallenge 已登录设备拒绝新设备登录。
func DenyLoginChallenge(c *gin.Context) {
	handleLoginChallengeDeny(c)
}

func handleLoginChallengeDeny(c *gin.Context) {
	challengeID, err := parseLoginChallengeID(c.Param("id"))
	if err != nil {
		response.InvalidParams(c, "无效的挑战 ID")
		return
	}

	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req loginChallengeActionReq
	_ = c.ShouldBindJSON(&req)

	approverDeviceID := normalizeDeviceIDForHandler(req.CurrentDeviceID)
	if approverDeviceID == "" {
		approverDeviceID = resolveCurrentDeviceID(c, imUserID)
	}

	if err := controller_chat.DenyLoginChallenge(imUserID, challengeID, approverDeviceID); err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, nil)
}

// VerifyLoginChallengeBySMS 新设备用短信验证码自助确认（无其它已登录设备时）。
func VerifyLoginChallengeBySMS(c *gin.Context) {
	challengeID, err := parseLoginChallengeID(c.Param("id"))
	if err != nil {
		response.InvalidParams(c, "无效的挑战 ID")
		return
	}

	var req loginChallengeSMSVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.PreAuthToken = strings.TrimSpace(req.PreAuthToken)
	if !authlimit.Enforce(c, "login_challenge_sms", []ratelimit.SubjectRules{
		{Subject: loginChallengeRateSubject(challengeID), Rules: ratelimit.PublicChallengeSMSRules},
	}) {
		return
	}

	if err := controller_chat.ApproveLoginChallengeBySMS(c.Request.Context(), challengeID, req.PreAuthToken, req.Code); err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, nil)
}

// ExchangeLoginChallenge 新设备兑换正式登录会话。
func ExchangeLoginChallenge(c *gin.Context) {
	challengeID, err := parseLoginChallengeID(c.Param("id"))
	if err != nil {
		response.InvalidParams(c, "无效的挑战 ID")
		return
	}

	var req loginChallengeExchangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.PreAuthToken = strings.TrimSpace(req.PreAuthToken)
	if !authlimit.Enforce(c, "login_challenge_exchange", []ratelimit.SubjectRules{
		{Subject: loginChallengeRateSubject(challengeID), Rules: ratelimit.PublicChallengeExchangeRules},
	}) {
		return
	}

	result, err := controller_chat.ExchangeLoginChallenge(c.Request.Context(), challengeID, req.PreAuthToken)
	if err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, result)
}

func parseLoginChallengeID(raw string) (uint, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid challenge id")
	}
	return uint(id), nil
}

func loginChallengeRateSubject(challengeID uint) string {
	return ratelimit.HashSubject("challenge", strconv.FormatUint(uint64(challengeID), 10))
}

func normalizeDeviceIDForHandler(deviceID string) string {
	return controller_chat.NormalizeDeviceID(deviceID)
}

func resolveCurrentDeviceID(c *gin.Context, imUserID string) string {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return ""
	}
	refreshToken := security.NormalizeRefreshToken(c.GetHeader("X-Refresh-Token"))
	if refreshToken == "" {
		return ""
	}
	deviceID, err := repository.NewUserRefreshSessionRepository().FindActiveSessionDeviceID(
		user.ID,
		security.HashRefreshToken(refreshToken),
	)
	if err != nil {
		return ""
	}
	return normalizeDeviceIDForHandler(deviceID)
}
