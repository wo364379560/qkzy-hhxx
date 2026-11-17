package security

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
)

// NormalizeIPWhitelistRule 标准化单个白名单规则，支持单 IP 或 CIDR。
func NormalizeIPWhitelistRule(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("IP 白名单规则不能为空")
	}

	if prefix, err := netip.ParsePrefix(trimmed); err == nil {
		if !prefix.Addr().IsValid() {
			return "", fmt.Errorf("IP 白名单规则无效")
		}
		return prefix.Masked().String(), nil
	}

	addr, err := netip.ParseAddr(trimmed)
	if err != nil {
		return "", fmt.Errorf("IP 白名单规则格式错误")
	}

	bits := 32
	if addr.Is6() {
		bits = 128
	}
	return netip.PrefixFrom(addr, bits).Masked().String(), nil
}

// MatchIPWhitelist 判断客户端 IP 是否命中任一白名单规则。
func MatchIPWhitelist(clientIP string, rules []model.AdminUserIPWhitelist) (bool, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(clientIP))
	if err != nil {
		return false, fmt.Errorf("客户端 IP 无法识别")
	}

	for _, rule := range rules {
		prefix, err := netip.ParsePrefix(rule.CIDR)
		if err != nil {
			return false, fmt.Errorf("白名单规则格式异常")
		}
		if prefix.Contains(addr) {
			return true, nil
		}
	}

	return false, nil
}

// EvaluateAdminUserIPWhitelist 判断后台受限账号当前 IP 是否允许访问。
func EvaluateAdminUserIPWhitelist(admin *model.AdminUser, clientIP string, rules []model.AdminUserIPWhitelist) (bool, error) {
	if admin == nil {
		return false, fmt.Errorf("账号不存在")
	}
	if (admin.Role != "assistant" && admin.Role != "admin") || !admin.IPWhitelistEnabled {
		return true, nil
	}
	if len(rules) == 0 {
		return false, nil
	}
	return MatchIPWhitelist(clientIP, rules)
}
