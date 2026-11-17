package security

import (
	"fmt"
	"net/netip"
	"strings"
)

const maxSingleIPWhitelistEntries = 50

// NormalizeSingleIP 标准化单个 IP 地址，仅支持精确 IP，不支持 CIDR。
func NormalizeSingleIP(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("IP 地址不能为空")
	}

	addr, err := netip.ParseAddr(trimmed)
	if err != nil || !addr.IsValid() {
		return "", fmt.Errorf("IP 地址格式错误")
	}

	return addr.String(), nil
}

// MatchSingleIP 判断两个单 IP 是否完全一致。
func MatchSingleIP(clientIP, allowedIP string) (bool, error) {
	normalizedClientIP, err := NormalizeSingleIP(clientIP)
	if err != nil {
		return false, fmt.Errorf("客户端 IP 无法识别")
	}

	normalizedAllowedIP, err := NormalizeSingleIP(allowedIP)
	if err != nil {
		return false, fmt.Errorf("白名单 IP 配置无效")
	}

	return normalizedClientIP == normalizedAllowedIP, nil
}

// NormalizeSingleIPList 标准化多条精确 IP 白名单规则，支持换行、逗号、分号分隔。
func NormalizeSingleIPList(value string) ([]string, error) {
	normalizedDelimiters := strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"，", ",",
		"；", ";",
	).Replace(value)

	parts := strings.FieldsFunc(normalizedDelimiters, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	if len(parts) == 0 {
		return nil, fmt.Errorf("IP 地址不能为空")
	}

	seen := make(map[string]struct{}, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		normalizedIP, err := NormalizeSingleIP(part)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalizedIP]; ok {
			continue
		}
		seen[normalizedIP] = struct{}{}
		result = append(result, normalizedIP)
		if len(result) > maxSingleIPWhitelistEntries {
			return nil, fmt.Errorf("白名单 IP 最多 %d 条", maxSingleIPWhitelistEntries)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("IP 地址不能为空")
	}

	return result, nil
}

func JoinSingleIPList(ips []string) string {
	if len(ips) == 0 {
		return ""
	}
	return strings.Join(ips, "\n")
}

// MatchSingleIPList 判断客户端 IP 是否命中任一精确 IP 白名单。
func MatchSingleIPList(clientIP string, allowedIPs []string) (bool, error) {
	normalizedClientIP, err := NormalizeSingleIP(clientIP)
	if err != nil {
		return false, fmt.Errorf("客户端 IP 无法识别")
	}

	for _, allowedIP := range allowedIPs {
		normalizedAllowedIP, err := NormalizeSingleIP(allowedIP)
		if err != nil {
			return false, fmt.Errorf("白名单 IP 配置无效")
		}
		if normalizedClientIP == normalizedAllowedIP {
			return true, nil
		}
	}

	return false, nil
}

// MatchSingleIPListText 判断客户端 IP 是否命中存储为文本的多条精确 IP 白名单。
func MatchSingleIPListText(clientIP, allowedIPs string) (bool, error) {
	normalizedIPs, err := NormalizeSingleIPList(allowedIPs)
	if err != nil {
		return false, fmt.Errorf("白名单 IP 配置无效")
	}
	return MatchSingleIPList(clientIP, normalizedIPs)
}
