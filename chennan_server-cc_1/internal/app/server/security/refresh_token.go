package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

const refreshTokenRandomBytes = 32

// GenerateRefreshToken 生成高熵 refresh token 明文。
func GenerateRefreshToken() (string, error) {
	buf := make([]byte, refreshTokenRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NormalizeRefreshToken 规范化 refresh token 输入。
func NormalizeRefreshToken(raw string) string {
	return strings.TrimSpace(raw)
}

// HashRefreshToken 计算 refresh token 的 SHA-256 十六进制摘要。
func HashRefreshToken(raw string) string {
	normalized := NormalizeRefreshToken(raw)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
