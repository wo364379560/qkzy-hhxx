package setup

import (
	"context"
	"errors"
	"fmt"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/sms"
	"github.com/ixayda/ix-Go/pkg/idgs"
	"github.com/ixayda/ix-Go/pkg/logger"
)

// Database 执行 AutoMigrate 并写入初始数据（管理员 + 测试用户）。
func Database() {
	db := datasource.Datasource.Mysql.GetClient()
	if err := idgs.Global.Init(1, 2); err != nil {
		logger.Panicf("init idgs failed: %v", err)
	}

	if err := db.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci").
		AutoMigrate(
			&model.AdminUser{},
			&model.AdminChatQuietHoursPolicy{},
			&model.AdminUserIPWhitelist{},
			&model.AdminAccessAudit{},
			&model.AdminLoginAudit{},
			&model.LoginRiskControl{},
			&model.IPAccessBlock{},
			&model.CleanupUserWhitelist{},
			&model.CleanupUserDisposition{},
			&model.ChatQuietHoursHitLog{},
			&model.QuietHoursGroupRisk{},
			&model.QuietHoursGroupProtection{},
			&model.AdminTOTPBackupCode{},
			&model.SystemSetting{},
			&model.ChatAttachment{},
			&model.ChatMultipartUpload{},
			&model.ForwardJob{}, &model.ForwardDelivery{}, &model.ForwardGate{},
			&model.ForwardTarget{}, &model.ForwardMessage{}, &model.ForwardRetry{},
			&model.ChatKeyword{},
			&model.User{},
			&model.FriendOwnerOperation{},
			&model.UserTOTPSetting{},
			&model.UserTOTPBackupCode{},
			&model.UserSecuritySettings{},
			&model.UserLoginChallenge{},
			&model.UserTrustedDevice{},
			&model.UserDevice{},
			&model.UserConversationCustomPin{},
			&model.InviteCode{},
			&model.Group{},
			&model.GroupNotice{},
			&model.GroupMessagePin{},
			&model.GroupBusinessAdmin{},
			&model.GroupTodo{},
			&model.GroupSharedEvent{},
			&model.GroupSharedLedger{},
			&model.UserRefreshSession{},
			&model.UserOperationLog{},
			&model.SMSSendLog{},
			&model.UserHabit{},
			&model.HabitCheckin{},
			&model.Challenge{},
			&model.ChallengeMember{},
			&model.CommunityPost{},
			&model.CommunityPostLike{},
			&model.CommunityPostComment{},
			&model.OfflineActivity{},
			&model.OfflineActivityRegistration{},
			&model.Report{},
			&model.UserRealName{},
			&model.UserUnverifiedMessageLimit{},
		); err != nil {
		logger.Panicf("auto migrate failed: %v", err)
	}
	if err := sms.MigrateSchema(db); err != nil {
		logger.Panicf("auto migrate sms schema failed: %v", err)
	}
	if err := repository.EnsureOwnershipSchema(db); err != nil {
		logger.Panicf("ownership schema/legacy writer guard failed: %v", err)
	}
	logger.Infof("database migrate [ ok ]")

	if err := ensureAdminUserStatusEnum(db); err != nil {
		logger.Panicf("ensure admin_users.status enum failed: %v", err)
	}
	if err := ensureAdminUserIPWhitelistCIDRColumn(db); err != nil {
		logger.Panicf("ensure admin_user_ip_whitelists.cidr failed: %v", err)
	}
	if err := ensureUserNicknameNonUnique(db); err != nil {
		logger.Panicf("ensure users.nickname non-unique failed: %v", err)
	}
	if err := ensureUserBizIDs(db); err != nil {
		logger.Panicf("ensure users.biz_id failed: %v", err)
	}
	if err := ensureGroupBizIDs(db); err != nil {
		logger.Panicf("ensure groups.biz_id failed: %v", err)
	}
	if err := ensureGroupSlowModeColumns(db); err != nil {
		logger.Panicf("ensure groups slow mode columns failed: %v", err)
	}
	if err := ensureUserAuthVersions(db); err != nil {
		logger.Panicf("ensure users.auth_version failed: %v", err)
	}
	if err := ensureAdminAuthVersions(db); err != nil {
		logger.Panicf("ensure admin_users.auth_version failed: %v", err)
	}
	if err := ensureUserIPWhitelistColumn(db); err != nil {
		logger.Panicf("ensure users.ip_whitelist_ip failed: %v", err)
	}
	if err := ensureSystemSettingValueColumn(db); err != nil {
		logger.Panicf("ensure system_settings.value failed: %v", err)
	}
	if err := ensureUserRefreshSessionIPActiveIndex(db); err != nil {
		logger.Panicf("ensure user_refresh_sessions ip active index failed: %v", err)
	}
	if err := ensureAdminLoginAuditRiskSyncIndex(db); err != nil {
		logger.Panicf("ensure admin_login_audits risk sync index failed: %v", err)
	}
	if err := backfillIPAccessBlockTriggers(db); err != nil {
		logger.Warnf("backfill ip_access_blocks trigger fields failed: %v", err)
	}

	seedAdmin(db)
	purgeBootstrapAdmins(db)
	seedTestUser(db)
}

type ipAccessBlockTriggerCandidate struct {
	Username     string
	FailureCount int
}

func backfillIPAccessBlockTriggers(db *gorm.DB) error {
	var rows []model.IPAccessBlock
	if err := db.
		Where("trigger_username = ? AND reason LIKE ?", "", "后台登录疑似撞库%").
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var candidate ipAccessBlockTriggerCandidate
		windowStart := row.CreatedAt.Add(-time.Hour)
		windowEnd := row.CreatedAt.Add(time.Minute)
		err := db.Model(&model.AdminLoginAudit{}).
			Select("username, COUNT(*) AS failure_count").
			Where("result = ? AND stage = ? AND ip_address = ? AND created_at BETWEEN ? AND ?", "failed", "password", row.IPAddress, windowStart, windowEnd).
			Group("username").
			Order("failure_count DESC").
			Limit(1).
			Scan(&candidate).Error
		if err != nil {
			return err
		}
		candidate.Username = strings.TrimSpace(candidate.Username)
		if candidate.Username == "" || candidate.FailureCount <= 0 {
			continue
		}
		reason := fmt.Sprintf("后台登录疑似撞库：账号 %s 近1小时密码错误%d次，来源 IP 已自动封禁，可在服务器防护页解封", candidate.Username, candidate.FailureCount)
		if err := db.Model(&model.IPAccessBlock{}).
			Where("id = ? AND trigger_username = ?", row.ID, "").
			UpdateColumns(map[string]any{
				"trigger_username":   candidate.Username,
				"trigger_fail_count": candidate.FailureCount,
				"trigger_window":     "1小时",
				"reason":             reason,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureAdminUserStatusEnum(db *gorm.DB) error {
	// GORM AutoMigrate 不会可靠地把 MySQL ENUM 旧定义修正为新定义，
	// 这里显式兜底，避免线上仍保留旧 status 枚举导致 banned 更新失败。
	return db.Exec(`
		ALTER TABLE admin_users
		MODIFY COLUMN status ENUM('active', 'banned') NOT NULL DEFAULT 'active' COMMENT '账号状态'
	`).Error
}

type tableColumn struct {
	ColumnName string `gorm:"column:column_name"`
}

type tableIndex struct {
	IndexName  string `gorm:"column:index_name"`
	ColumnName string `gorm:"column:column_name"`
	SeqInIndex int    `gorm:"column:seq_in_index"`
}

func ensureAdminUserIPWhitelistCIDRColumn(db *gorm.DB) error {
	var columns []tableColumn
	if err := db.Raw(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'admin_user_ip_whitelists'
		  AND column_name IN ('cidr', 'c_id_r')
	`).Scan(&columns).Error; err != nil {
		return err
	}

	hasCIDR := false
	hasLegacyCIDR := false
	for _, column := range columns {
		switch column.ColumnName {
		case "cidr":
			hasCIDR = true
		case "c_id_r":
			hasLegacyCIDR = true
		}
	}

	switch {
	case hasCIDR && hasLegacyCIDR:
		if err := db.Exec(`
			UPDATE admin_user_ip_whitelists
			SET cidr = c_id_r
			WHERE (cidr IS NULL OR cidr = '')
			  AND c_id_r IS NOT NULL
			  AND c_id_r <> ''
		`).Error; err != nil {
			return err
		}
		if err := dropAdminUserIPWhitelistIndex(db, "uk_admin_user_cidr"); err != nil {
			return err
		}
		if err := db.Exec(`
			ALTER TABLE admin_user_ip_whitelists
			DROP COLUMN c_id_r
		`).Error; err != nil {
			return err
		}
		logger.Infof("normalized admin_user_ip_whitelists.cidr from legacy c_id_r column")
	case !hasCIDR && hasLegacyCIDR:
		if err := db.Exec(`
			ALTER TABLE admin_user_ip_whitelists
			CHANGE COLUMN c_id_r cidr VARCHAR(64) NOT NULL COMMENT '标准化后的IP/CIDR'
		`).Error; err != nil {
			return err
		}
		logger.Infof("renamed admin_user_ip_whitelists.c_id_r to cidr")
	}

	return ensureAdminUserIPWhitelistCIDRIndex(db)
}

func ensureAdminUserIPWhitelistCIDRIndex(db *gorm.DB) error {
	var indexes []tableIndex
	if err := db.Raw(`
		SELECT index_name, column_name, seq_in_index
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'admin_user_ip_whitelists'
		  AND index_name = 'uk_admin_user_cidr'
		ORDER BY seq_in_index ASC
	`).Scan(&indexes).Error; err != nil {
		return err
	}

	if len(indexes) == 2 &&
		indexes[0].ColumnName == "admin_user_id" &&
		indexes[1].ColumnName == "cidr" {
		return nil
	}

	if err := dropAdminUserIPWhitelistIndex(db, "uk_admin_user_cidr"); err != nil {
		return err
	}
	return db.Exec(`
		ALTER TABLE admin_user_ip_whitelists
		ADD UNIQUE KEY uk_admin_user_cidr (admin_user_id, cidr)
	`).Error
}

func dropAdminUserIPWhitelistIndex(db *gorm.DB, indexName string) error {
	var indexes []tableIndex
	if err := db.Raw(`
		SELECT index_name, column_name, seq_in_index
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'admin_user_ip_whitelists'
		  AND index_name = ?
	`, indexName).Scan(&indexes).Error; err != nil {
		return err
	}
	if len(indexes) == 0 {
		return nil
	}

	return db.Exec(fmt.Sprintf(
		"ALTER TABLE admin_user_ip_whitelists DROP INDEX `%s`",
		indexName,
	)).Error
}

func ensureUserRefreshSessionIPActiveIndex(db *gorm.DB) error {
	const indexName = "idx_user_refresh_sessions_ip_active_user"
	var indexes []tableIndex
	if err := db.Raw(`
		SELECT index_name, column_name, seq_in_index
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'user_refresh_sessions'
		  AND index_name = ?
		ORDER BY seq_in_index ASC
	`, indexName).Scan(&indexes).Error; err != nil {
		return err
	}

	if len(indexes) == 4 &&
		indexes[0].ColumnName == "ip" &&
		indexes[1].ColumnName == "revoked_at" &&
		indexes[2].ColumnName == "expires_at" &&
		indexes[3].ColumnName == "user_id" {
		return nil
	}

	if len(indexes) > 0 {
		if err := db.Exec(fmt.Sprintf(
			"ALTER TABLE user_refresh_sessions DROP INDEX `%s`",
			indexName,
		)).Error; err != nil {
			return err
		}
	}

	return db.Exec(`
		CREATE INDEX idx_user_refresh_sessions_ip_active_user
		ON user_refresh_sessions (ip, revoked_at, expires_at, user_id)
	`).Error
}

func ensureAdminLoginAuditRiskSyncIndex(db *gorm.DB) error {
	const indexName = "idx_admin_login_audits_risk_sync"
	var indexes []tableIndex
	if err := db.Raw(`
		SELECT index_name, column_name, seq_in_index
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'admin_login_audits'
		  AND index_name = ?
		ORDER BY seq_in_index ASC
	`, indexName).Scan(&indexes).Error; err != nil {
		return err
	}

	if len(indexes) == 4 &&
		indexes[0].ColumnName == "result" &&
		indexes[1].ColumnName == "stage" &&
		indexes[2].ColumnName == "created_at" &&
		indexes[3].ColumnName == "ip_address" {
		return nil
	}

	if len(indexes) > 0 {
		if err := db.Exec(fmt.Sprintf(
			"ALTER TABLE admin_login_audits DROP INDEX `%s`",
			indexName,
		)).Error; err != nil {
			return err
		}
	}

	return db.Exec(`
		CREATE INDEX idx_admin_login_audits_risk_sync
		ON admin_login_audits (result, stage, created_at, ip_address)
	`).Error
}

type userNicknameUniqueIndex struct {
	IndexName string `gorm:"column:index_name"`
}

func ensureUserNicknameNonUnique(db *gorm.DB) error {
	var indexes []userNicknameUniqueIndex
	if err := db.Raw(`
		SELECT DISTINCT index_name
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'users'
		  AND column_name = 'nickname'
		  AND non_unique = 0
		  AND index_name <> 'PRIMARY'
	`).Scan(&indexes).Error; err != nil {
		return err
	}

	for _, idx := range indexes {
		if idx.IndexName == "" {
			continue
		}
		if err := db.Exec(fmt.Sprintf("ALTER TABLE users DROP INDEX `%s`", idx.IndexName)).Error; err != nil {
			return err
		}
		logger.Infof("dropped unique index %q on users.nickname", idx.IndexName)
	}
	return nil
}

func ensureUserIPWhitelistColumn(db *gorm.DB) error {
	if err := db.Exec(`
		UPDATE users
		SET ip_whitelist_ip = ''
		WHERE ip_whitelist_ip IS NULL
	`).Error; err != nil {
		return err
	}

	return db.Exec(`
		ALTER TABLE users
		MODIFY COLUMN ip_whitelist_ip TEXT NOT NULL COMMENT '用户登录 IP 白名单，多条以换行分隔'
	`).Error
}

func ensureSystemSettingValueColumn(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE system_settings
		MODIFY COLUMN value TEXT NOT NULL
	`).Error
}

// seedAdmin 仅在「系统里还没有任何活跃超管」时创建初始超管。
// 由 ADMIN_USERNAME + ADMIN_PASSWORD 控制；不再把已存在账号强行提升/解禁，
// 避免每次部署把已删除的 admin/888 又「复活」成超管。
func seedAdmin(db *gorm.DB) {
	username := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" || password == "" {
		return
	}
	if isBootstrapAdminUsername(username) {
		logger.Warnf("skip bootstrap admin %q: username is in purge list (admin/888)", username)
		return
	}

	var activeSuperCount int64
	if err := db.Model(&model.AdminUser{}).
		Where("role = ? AND status = ?", "super_admin", "active").
		Count(&activeSuperCount).Error; err != nil {
		logger.Panicf("count active super_admin failed: %v", err)
	}
	if activeSuperCount > 0 {
		return
	}

	var admin model.AdminUser
	err := db.Where("username = ?", username).First(&admin).Error
	switch {
	case err == nil:
		logger.Infof("bootstrap admin %q already exists; skip recreate (active_super_admin=0)", username)
		return
	case !errors.Is(err, gorm.ErrRecordNotFound):
		logger.Panicf("query bootstrap admin failed: %v", err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Panicf("bcrypt failed: %v", err)
	}
	if err := db.Create(&model.AdminUser{
		Username:     username,
		PasswordHash: string(hash),
		Role:         "super_admin",
		Status:       "active",
	}).Error; err != nil {
		logger.Panicf("seed admin failed: %v", err)
	}
	logger.Infof("super admin user %q created (first active super_admin)", username)
}

// purgeBootstrapAdmins 彻底禁用历史写死/误种的超管账号 admin、888。
// 不物理删行（避免 users.owner_id 悬空）；禁用后不会再出现在「分配给」活跃列表。
// 可用环境变量 ADMIN_PURGE_USERNAMES 覆盖，逗号分隔；设为 "-" 可关闭清理。
func purgeBootstrapAdmins(db *gorm.DB) {
	names := bootstrapAdminPurgeUsernames()
	if len(names) == 0 {
		return
	}
	res := db.Model(&model.AdminUser{}).
		Where("username IN ?", names).
		Where("status <> ?", "banned").
		Updates(map[string]any{
			"status": "banned",
			"role":   "assistant",
		})
	if res.Error != nil {
		logger.Warnf("purge bootstrap admins failed: %v", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		logger.Infof("purged bootstrap admins %v (banned, rows=%d)", names, res.RowsAffected)
	}
}

func bootstrapAdminPurgeUsernames() []string {
	raw := strings.TrimSpace(os.Getenv("ADMIN_PURGE_USERNAMES"))
	if raw == "-" {
		return nil
	}
	if raw == "" {
		raw = "admin,888"
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

func isBootstrapAdminUsername(username string) bool {
	u := strings.ToLower(strings.TrimSpace(username))
	for _, name := range bootstrapAdminPurgeUsernames() {
		if strings.ToLower(name) == u {
			return true
		}
	}
	return false
}

// seedTestUser 写入调试测试账号（TEST_USER_PHONE + TEST_USER_PASSWORD 环境变量控制）。
// 仅在开发/测试环境设置，生产环境不设置即不创建。
func seedTestUser(db *gorm.DB) {
	phone := os.Getenv("TEST_USER_PHONE")
	password := os.Getenv("TEST_USER_PASSWORD")
	if phone == "" || password == "" {
		return
	}

	nickname := os.Getenv("TEST_USER_NICKNAME")
	if nickname == "" {
		nickname = "测试用户"
	}

	ownerID := firstActiveSuperAdminID(db)

	var existing model.User
	err := db.Where("phone = ?", phone).First(&existing).Error
	switch {
	case err == nil:
		if existing.OwnerID == 0 && ownerID > 0 {
			if err := repository.NewOwnershipRepository().RepairTestOwner(context.Background(), existing.ID, ownerID); err != nil {
				logger.Warnf("repair test user owner failed")
			}
		}

		return
	case !errors.Is(err, gorm.ErrRecordNotFound):
		logger.Warnf("query test user failed: %v", err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Panicf("bcrypt failed: %v", err)
	}

	// 固定 IM UserID，方便调试时追踪
	const testIMUserID = "u_test000001"

	user := &model.User{
		BizID:        idgs.Global.UserGenerate().Int64(),
		Phone:        phone,
		PasswordHash: string(hash),
		IMUserID:     testIMUserID,
		Nickname:     nickname,
		OwnerID:      ownerID,
		Status:       "active",
		CreatedAt:    time.Now(),
	}
	if err := db.Create(user).Error; err != nil {
		logger.Warnf("seed test user failed: %v", err)
		return
	}

	// 同步注册到腾讯 IM（未配置 SDKAppID 时跳过）
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		if err := imClient.AccountImport(testIMUserID, nickname, ""); err != nil {
			logger.Warnf("test user IM import failed (non-fatal): %v", err)
		} else if err := imClient.SetAllowType(testIMUserID, 0); err != nil {
			logger.Warnf("test user IM allowType sync failed (non-fatal): %v", err)
		}
	} else {
		logger.Warnf("TencentIM not configured, skipping IM import for test user")
	}

	logger.Infof("test user %q created (im_user_id=%s owner_id=%d)", nickname, testIMUserID, ownerID)
}

func firstActiveSuperAdminID(db *gorm.DB) uint {
	var admin model.AdminUser
	err := db.Where("role = ? AND status = ?", "super_admin", "active").
		Order("id ASC").
		First(&admin).Error
	if err != nil {
		return 0
	}
	return admin.ID
}

func ensureUserBizIDs(db *gorm.DB) error {
	var users []model.User
	if err := db.Where("biz_id = 0").Find(&users).Error; err != nil {
		return err
	}
	for i := range users {
		if err := db.Model(&users[i]).Update("biz_id", idgs.Global.UserGenerate().Int64()).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureGroupBizIDs(db *gorm.DB) error {
	var groups []model.Group
	if err := db.Where("biz_id = 0").Find(&groups).Error; err != nil {
		return err
	}
	for i := range groups {
		if err := db.Model(&groups[i]).Update("biz_id", idgs.Global.GroupGenerate().Int64()).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureGroupSlowModeColumns(db *gorm.DB) error {
	var columns []tableColumn
	if err := db.Raw(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'groups'
		  AND column_name IN ('slow_mode_enabled', 'slow_mode_seconds', 'slow_mode_version')
	`).Scan(&columns).Error; err != nil {
		return err
	}

	existing := make(map[string]bool, len(columns))
	for _, column := range columns {
		existing[column.ColumnName] = true
	}

	alterParts := make([]string, 0, 3)
	if !existing["slow_mode_enabled"] {
		alterParts = append(alterParts, "ADD COLUMN slow_mode_enabled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否开启群发言频率限制'")
	}
	if !existing["slow_mode_seconds"] {
		alterParts = append(alterParts, "ADD COLUMN slow_mode_seconds INT NOT NULL DEFAULT 0 COMMENT '普通成员两次发言间隔秒数'")
	}
	if !existing["slow_mode_version"] {
		alterParts = append(alterParts, "ADD COLUMN slow_mode_version BIGINT NOT NULL DEFAULT 0 COMMENT '慢速模式版本'")
	}
	if len(alterParts) == 0 {
		return nil
	}

	if err := db.Exec("ALTER TABLE groups " + strings.Join(alterParts, ", ")).Error; err != nil {
		return err
	}
	logger.Infof("ensured groups slow mode columns")
	return nil
}

func ensureUserAuthVersions(db *gorm.DB) error {
	return db.Model(&model.User{}).
		Where("auth_version <= 0").
		Update("auth_version", 1).Error
}

func ensureAdminAuthVersions(db *gorm.DB) error {
	return db.Model(&model.AdminUser{}).
		Where("auth_version <= 0").
		Update("auth_version", 1).Error
}
