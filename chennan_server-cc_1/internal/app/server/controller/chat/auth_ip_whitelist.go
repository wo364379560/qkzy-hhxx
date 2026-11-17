package controller_chat

import (
	"fmt"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/security"
)

const (
	userWhitelistDeniedMessageFormat  = "当前登录 IP（%s）不在白名单内，请联系管理员"
	userWhitelistInvalidConfigMessage = "账号白名单配置不完整，请联系管理员"
	userWhitelistUnknownIPMessage     = "当前登录 IP 无法识别，请稍后重试"
)

func ensureUserIPWhitelistAllowed(user *model.User, clientIP string) error {
	if user == nil || !user.IPWhitelistEnabled {
		return nil
	}

	allowedIP := strings.TrimSpace(user.IPWhitelistIP)
	if allowedIP == "" {
		return AuthErr(userWhitelistInvalidConfigMessage)
	}

	normalizedClientIP, err := security.NormalizeSingleIP(clientIP)
	if err != nil {
		return AuthErr(userWhitelistUnknownIPMessage)
	}

	normalizedAllowedIPs, err := security.NormalizeSingleIPList(allowedIP)
	if err != nil {
		return AuthErr(userWhitelistInvalidConfigMessage)
	}

	matched, err := security.MatchSingleIPList(normalizedClientIP, normalizedAllowedIPs)
	if err != nil {
		return AuthErr(userWhitelistInvalidConfigMessage)
	}
	if !matched {
		return AuthErr(fmt.Sprintf(userWhitelistDeniedMessageFormat, normalizedClientIP))
	}

	return nil
}
