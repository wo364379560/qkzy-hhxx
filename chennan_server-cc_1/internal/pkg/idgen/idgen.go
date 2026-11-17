// Package idgen 生成腾讯云 IM UserID。
// 格式：{prefix}_{序号}{4 位盐值}，序号从 1000 起步，每次在当前最大值基础上随机递增 1-9。
// 与 server-old 保持一致，使用 DB 查询而非 Redis 序号，无额外中间件依赖。
package idgen

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

const (
	maxSeq    = int64(99_999_999)
	saltChars = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// GenUserID 生成唯一 IM UserID，格式：{prefix}_{序号}{4 位盐值}。
//
// prefix 取值：
//
//	"u" — 用户自助注册
//	"a" — 管理员手动建号
//	"t" — 测试账号
func GenUserID(ctx context.Context, prefix string) (string, error) {
	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)

	maxNum := findMaxSeq(db, prefix)

	for range 10 {
		inc, err := rand.Int(rand.Reader, big.NewInt(9))
		if err != nil {
			return "", fmt.Errorf("idgen: rand: %w", err)
		}
		num := maxNum + inc.Int64() + 1
		if num > maxSeq {
			num = maxSeq
		}

		salt, err := genSalt(4)
		if err != nil {
			return "", fmt.Errorf("idgen: salt: %w", err)
		}

		uid := fmt.Sprintf("%s_%d%s", prefix, num, salt)

		var count int64
		db.Raw("SELECT COUNT(*) FROM users WHERE im_user_id = ?", uid).Scan(&count)
		if count == 0 {
			return uid, nil
		}
		// 碰撞：用当前 num 作为新基线继续尝试
		maxNum = num
	}
	return "", fmt.Errorf("idgen: max retries exceeded")
}

// findMaxSeq 在 users 表中查出 {prefix}_ 开头的最大序号（默认 999，首个用户从 1000 起步）。
func findMaxSeq(db *gorm.DB, prefix string) int64 {
	maxNum := int64(999)
	prefixLen := len(prefix) + 1
	startPos := prefixLen + 1
	minLength := prefixLen + 5
	pattern := fmt.Sprintf("%s\\_%%", prefix)

	db.Raw(
		"SELECT COALESCE(MAX(CAST(SUBSTRING(im_user_id, ?, CHAR_LENGTH(im_user_id) - ? - 4) AS UNSIGNED)), 999) "+
			"FROM users WHERE im_user_id LIKE ? AND CHAR_LENGTH(im_user_id) >= ?",
		startPos, prefixLen, pattern, minLength,
	).Scan(&maxNum)

	if maxNum < 999 {
		maxNum = 999
	}
	return maxNum
}

// GenGroupID 生成唯一 IM GroupID，格式：{prefix}_{序号}{4 位盐值}。
//
// prefix 取值：
//
//	"ga" — 管理员后台建群
//	"gu" — 用户 App 建群（通过 TIM 回调同步）
func GenGroupID(ctx context.Context, prefix string) (string, error) {
	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)

	maxNum := findMaxGroupSeq(db, prefix)

	for range 10 {
		inc, err := rand.Int(rand.Reader, big.NewInt(9))
		if err != nil {
			return "", fmt.Errorf("idgen: rand: %w", err)
		}
		num := maxNum + inc.Int64() + 1
		if num > maxSeq {
			num = maxSeq
		}

		salt, err := genSalt(4)
		if err != nil {
			return "", fmt.Errorf("idgen: salt: %w", err)
		}

		gid := fmt.Sprintf("%s_%d%s", prefix, num, salt)

		var count int64
		db.Raw("SELECT COUNT(*) FROM `groups` WHERE im_group_id = ?", gid).Scan(&count)
		if count == 0 {
			return gid, nil
		}
		maxNum = num
	}
	return "", fmt.Errorf("idgen: max retries exceeded")
}

// findMaxGroupSeq 在 groups 表中查出指定前缀开头的最大序号（默认 999）。
func findMaxGroupSeq(db *gorm.DB, prefix string) int64 {
	maxNum := int64(999)
	prefixLen := len(prefix) + 1
	startPos := prefixLen + 1
	minLength := prefixLen + 5
	pattern := fmt.Sprintf("%s\\_%%", prefix)

	db.Raw(
		"SELECT COALESCE(MAX(CAST(SUBSTRING(im_group_id, ?, CHAR_LENGTH(im_group_id) - ? - 4) AS UNSIGNED)), 999) "+
			"FROM `groups` WHERE im_group_id LIKE ? AND CHAR_LENGTH(im_group_id) >= ?",
		startPos, prefixLen, pattern, minLength,
	).Scan(&maxNum)

	if maxNum < 999 {
		maxNum = 999
	}
	return maxNum
}

func genSalt(n int) (string, error) {
	charLen := big.NewInt(int64(len(saltChars)))
	salt := make([]byte, n)
	for i := range salt {
		idx, err := rand.Int(rand.Reader, charLen)
		if err != nil {
			return "", err
		}
		salt[i] = saltChars[idx.Int64()]
	}
	return string(salt), nil
}
