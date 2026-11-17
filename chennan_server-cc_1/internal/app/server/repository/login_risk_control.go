package repository

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

const (
	LoginRiskScopeAccount = "account"
	LoginRiskScopeIP      = "ip"

	loginRiskStatusWatching = "watching"
	loginRiskStatusCooldown = "cooldown"
	loginRiskStatusResolved = "resolved"

	loginRiskLevelLow    = "low"
	loginRiskLevelMedium = "medium"
	loginRiskLevelHigh   = "high"

	adminIPAutoBlockWindow    = time.Hour
	adminIPAutoBlockThreshold = 10

	adminPasswordFirstCooldownThreshold = 3
	adminPasswordLaterCooldownThreshold = 2
)

var accountPasswordCooldownDurations = []time.Duration{
	3 * time.Minute,
	10 * time.Minute,
	5 * time.Hour,
}

var adminPasswordCooldownDurations = []time.Duration{
	time.Hour,
	time.Hour,
	3 * time.Hour,
}

// LoginRiskControlRepository writes user login risk states for admin review.
type LoginRiskControlRepository struct {
	db *gorm.DB
}

type adminIPFailureAggregate struct {
	IPAddress       string
	Username        string
	FailureCount    int
	WindowStartedAt time.Time
	LatestAt        time.Time
}

func NewLoginRiskControlRepository() *LoginRiskControlRepository {
	return &LoginRiskControlRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func AccountLoginRiskKey(phone string) string {
	return "account:" + strings.TrimSpace(phone)
}

func AdminLoginRiskKey(username string) string {
	return "admin:" + strings.TrimSpace(username)
}

func IPLoginRiskKey(ipAddress string) string {
	return "ip:" + strings.TrimSpace(ipAddress)
}

// ActiveAccountCooldown returns an active account cooldown row, if present.
func (r *LoginRiskControlRepository) ActiveAccountCooldown(phone string, now time.Time) (*model.LoginRiskControl, bool, error) {
	return r.activeCooldown(AccountLoginRiskKey(phone), now)
}

// ActiveAccountCooldownsByPhones returns active account cooldown rows keyed by phone.
func (r *LoginRiskControlRepository) ActiveAccountCooldownsByPhones(phones []string, now time.Time) (map[string]model.LoginRiskControl, error) {
	result := make(map[string]model.LoginRiskControl, len(phones))
	if len(phones) == 0 {
		return result, nil
	}
	riskKeys := make([]string, 0, len(phones))
	phoneByRiskKey := make(map[string]string, len(phones))
	for _, phone := range phones {
		phone = strings.TrimSpace(phone)
		if phone == "" {
			continue
		}
		riskKey := AccountLoginRiskKey(phone)
		if _, exists := phoneByRiskKey[riskKey]; exists {
			continue
		}
		riskKeys = append(riskKeys, riskKey)
		phoneByRiskKey[riskKey] = phone
	}
	if len(riskKeys) == 0 {
		return result, nil
	}

	var rows []model.LoginRiskControl
	if err := r.db.
		Where("risk_key IN ?", riskKeys).
		Where("status = ?", loginRiskStatusCooldown).
		Where("cooldown_until IS NOT NULL AND cooldown_until > ?", now).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if phone, ok := phoneByRiskKey[row.RiskKey]; ok {
			result[phone] = row
		}
	}
	return result, nil
}

// ActiveAdminCooldown returns an active admin-account cooldown row, if present.
func (r *LoginRiskControlRepository) ActiveAdminCooldown(username string, now time.Time) (*model.LoginRiskControl, bool, error) {
	return r.activeCooldown(AdminLoginRiskKey(username), now)
}

// ActiveAdminCooldownsByUsernames returns active admin-account cooldown rows keyed by username.
func (r *LoginRiskControlRepository) ActiveAdminCooldownsByUsernames(usernames []string, now time.Time) (map[string]model.LoginRiskControl, error) {
	result := make(map[string]model.LoginRiskControl, len(usernames))
	if len(usernames) == 0 {
		return result, nil
	}
	riskKeys := make([]string, 0, len(usernames))
	usernameByRiskKey := make(map[string]string, len(usernames))
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		riskKey := AdminLoginRiskKey(username)
		if _, exists := usernameByRiskKey[riskKey]; exists {
			continue
		}
		riskKeys = append(riskKeys, riskKey)
		usernameByRiskKey[riskKey] = username
	}
	if len(riskKeys) == 0 {
		return result, nil
	}

	var rows []model.LoginRiskControl
	if err := r.db.
		Where("risk_key IN ?", riskKeys).
		Where("status = ?", loginRiskStatusCooldown).
		Where("cooldown_until IS NOT NULL AND cooldown_until > ?", now).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if username, ok := usernameByRiskKey[row.RiskKey]; ok {
			result[username] = row
		}
	}
	return result, nil
}

func (r *LoginRiskControlRepository) activeCooldown(riskKey string, now time.Time) (*model.LoginRiskControl, bool, error) {
	var row model.LoginRiskControl
	err := r.db.Where("risk_key = ?", riskKey).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if row.CooldownUntil != nil && row.CooldownUntil.After(now) && row.Status == loginRiskStatusCooldown {
		return &row, true, nil
	}
	return &row, false, nil
}

// RecordAccountPasswordFailure records one wrong password for review without
// hard-locking the account. A password-only account cooldown lets an attacker
// deny service to any known phone number; request/IP/account rate limits are
// used for enforcement instead.
func (r *LoginRiskControlRepository) RecordAccountPasswordFailure(user *model.User, ip string, now time.Time) (*model.LoginRiskControl, bool, error) {
	var saved model.LoginRiskControl
	err := r.db.Transaction(func(tx *gorm.DB) error {
		row, err := lockOrCreateAccountRisk(tx, user, ip, now)
		if err != nil {
			return err
		}

		row.FailureCount++
		if row.FailureCount > 1000 {
			row.FailureCount = 1000
		}
		row.IPAddress = strings.TrimSpace(ip)
		row.Status = loginRiskStatusWatching
		row.CooldownUntil = nil
		switch {
		case row.FailureCount >= 20:
			row.Level = loginRiskLevelHigh
		case row.FailureCount >= 5:
			row.Level = loginRiskLevelMedium
		default:
			row.Level = loginRiskLevelLow
		}
		row.Reason = "密码登录失败（已限频，未锁定账号）"

		if err := tx.Save(row).Error; err != nil {
			return err
		}
		saved = *row
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &saved, false, nil
}

// RecordAdminPasswordFailure records one wrong admin password and returns
// whether this failure triggered an admin-account cooldown.
func (r *LoginRiskControlRepository) RecordAdminPasswordFailure(username string, adminID *uint, adminRole, ip string, now time.Time) (*model.LoginRiskControl, bool, error) {
	username = strings.TrimSpace(username)
	var saved model.LoginRiskControl
	triggered := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		row, err := lockOrCreateAdminRisk(tx, username, adminID, adminRole, ip, now)
		if err != nil {
			return err
		}

		hadPriorCooldown := row.CooldownUntil != nil
		row.FailureCount++
		row.IPAddress = strings.TrimSpace(ip)
		row.Reason = "后台密码登录失败"
		row.Status = loginRiskStatusWatching

		nextFailureCount, nextLevel, cooldownUntil, isTriggered := nextAdminPasswordCooldown(row.FailureCount, row.Level, hadPriorCooldown, now)
		row.FailureCount = nextFailureCount
		if isTriggered {
			row.Level = nextLevel
			row.Status = loginRiskStatusCooldown
			row.CooldownUntil = &cooldownUntil
			row.Reason = fmt.Sprintf("后台账号密码错误达到阈值，已冷却%s", formatCooldownDuration(cooldownUntil.Sub(now)))
			triggered = true
		} else {
			row.Status, row.Level, row.Reason = nextAdminPasswordRisk(row.FailureCount, row.Level, hadPriorCooldown)
		}

		if err := tx.Save(row).Error; err != nil {
			return err
		}
		saved = *row
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &saved, triggered, nil
}

// RecordAdminIPLoginFailure records a failed admin login source IP for manual review.
func (r *LoginRiskControlRepository) RecordAdminIPLoginFailure(ipAddress, username string, now time.Time) error {
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return nil
	}
	username = strings.TrimSpace(username)
	return r.db.Transaction(func(tx *gorm.DB) error {
		var row model.LoginRiskControl
		riskKey := IPLoginRiskKey(ipAddress)
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("risk_key = ?", riskKey).First(&row).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			row = model.LoginRiskControl{
				Scope:           LoginRiskScopeIP,
				RiskKey:         riskKey,
				Username:        username,
				IPAddress:       ipAddress,
				Status:          loginRiskStatusWatching,
				Level:           loginRiskLevelLow,
				Reason:          "后台登录失败来源",
				FailureCount:    0,
				WindowStartedAt: &now,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		previousStatus := row.Status
		row.Scope = LoginRiskScopeIP
		row.RiskKey = riskKey
		row.Username = username
		row.IPAddress = ipAddress
		row.FailureCount++
		row.Reason = "后台登录失败来源"
		row.Status = loginRiskStatusWatching
		row.Level = loginRiskLevelLow
		if row.FailureCount >= 5 {
			row.Status = "alerted"
			row.Level = loginRiskLevelMedium
			row.Reason = "后台登录失败来源，建议人工复核是否封禁"
		}
		if row.FailureCount >= 20 {
			row.Level = loginRiskLevelHigh
		}
		if row.WindowStartedAt == nil || previousStatus == loginRiskStatusResolved {
			row.WindowStartedAt = &now
		}
		return tx.Save(&row).Error
	})
}

// SyncRecentAdminIPLoginFailures rolls recent admin-login audit failures into IP
// risk rows for manual review and auto-blocks obvious credential stuffing
// sources. Auto-block only applies to public IPs with password-stage failures.
func (r *LoginRiskControlRepository) SyncRecentAdminIPLoginFailures(window time.Duration, now time.Time) error {
	if window <= 0 {
		window = 24 * time.Hour
	}
	startedAt := now.Add(-window)
	var aggregates []adminIPFailureAggregate
	if err := r.db.Model(&model.AdminLoginAudit{}).
		Select("ip_address, MAX(username) AS username, COUNT(*) AS failure_count, MIN(created_at) AS window_started_at, MAX(created_at) AS latest_at").
		Where("result = ? AND stage = ? AND ip_address <> ? AND created_at >= ?", "failed", "password", "", startedAt).
		Group("ip_address").
		Having("COUNT(*) >= ?", 3).
		Scan(&aggregates).Error; err != nil {
		return err
	}
	if len(aggregates) == 0 {
		return nil
	}

	if err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, aggregate := range aggregates {
			ipAddress := strings.TrimSpace(aggregate.IPAddress)
			if ipAddress == "" {
				continue
			}
			riskKey := IPLoginRiskKey(ipAddress)
			var row model.LoginRiskControl
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("risk_key = ?", riskKey).First(&row).Error
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				row = model.LoginRiskControl{
					Scope:           LoginRiskScopeIP,
					RiskKey:         riskKey,
					Username:        strings.TrimSpace(aggregate.Username),
					IPAddress:       ipAddress,
					Status:          loginRiskStatusWatching,
					Level:           loginRiskLevelLow,
					Reason:          "后台登录失败来源",
					FailureCount:    aggregate.FailureCount,
					WindowStartedAt: &aggregate.WindowStartedAt,
					CreatedAt:       now,
					UpdatedAt:       now,
				}
				applyAdminIPFailureRiskLevel(&row, window)
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}

			if row.Status == loginRiskStatusResolved && row.ResolvedAt != nil && !aggregate.LatestAt.After(*row.ResolvedAt) {
				continue
			}
			if row.Status == loginRiskStatusResolved && row.ResolvedAt != nil {
				refreshed, ok, err := r.adminIPFailureAggregateSince(tx, ipAddress, *row.ResolvedAt)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				aggregate = refreshed
			}
			row.Scope = LoginRiskScopeIP
			row.RiskKey = riskKey
			row.Username = strings.TrimSpace(aggregate.Username)
			row.IPAddress = ipAddress
			row.FailureCount = aggregate.FailureCount
			row.WindowStartedAt = &aggregate.WindowStartedAt
			row.CooldownUntil = nil
			applyAdminIPFailureRiskLevel(&row, window)
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return r.AutoBlockRecentAdminIPLoginFailures(adminIPAutoBlockWindow, adminIPAutoBlockThreshold, now)
}

// AutoBlockRecentAdminIPLoginFailures creates application-layer IP blocks for
// public IPs that hit the same admin account too often in a short window.
func (r *LoginRiskControlRepository) AutoBlockRecentAdminIPLoginFailures(window time.Duration, threshold int, now time.Time) error {
	if window <= 0 {
		window = adminIPAutoBlockWindow
	}
	if threshold <= 0 {
		threshold = adminIPAutoBlockThreshold
	}
	startedAt := now.Add(-window)
	var aggregates []adminIPFailureAggregate
	if err := r.db.Model(&model.AdminLoginAudit{}).
		Select("ip_address, username, COUNT(*) AS failure_count, MIN(created_at) AS window_started_at, MAX(created_at) AS latest_at").
		Where("result = ? AND stage = ? AND ip_address <> ? AND created_at >= ?", "failed", "password", "", startedAt).
		Group("ip_address, username").
		Having("COUNT(*) >= ?", threshold).
		Scan(&aggregates).Error; err != nil {
		return err
	}
	if len(aggregates) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, aggregate := range aggregates {
			if err := autoBlockAdminIPFailureAggregate(tx, aggregate, window, threshold, now); err != nil {
				return err
			}
		}
		return nil
	})
}

// AutoBlockAdminIPLoginFailure checks the current admin-login source only. It
// avoids scanning all recent audit rows on every wrong password attempt.
func (r *LoginRiskControlRepository) AutoBlockAdminIPLoginFailure(ipAddress, username string, window time.Duration, threshold int, now time.Time) error {
	ipAddress = strings.TrimSpace(ipAddress)
	username = strings.TrimSpace(username)
	if !isPublicIPAddress(ipAddress) || username == "" {
		return nil
	}
	if window <= 0 {
		window = adminIPAutoBlockWindow
	}
	if threshold <= 0 {
		threshold = adminIPAutoBlockThreshold
	}
	var aggregate adminIPFailureAggregate
	if err := r.db.Model(&model.AdminLoginAudit{}).
		Select("ip_address, username, COUNT(*) AS failure_count, MIN(created_at) AS window_started_at, MAX(created_at) AS latest_at").
		Where("result = ? AND stage = ? AND ip_address = ? AND username = ? AND created_at >= ?", "failed", "password", ipAddress, username, now.Add(-window)).
		Group("ip_address, username").
		Having("COUNT(*) >= ?", threshold).
		Scan(&aggregate).Error; err != nil {
		return err
	}
	if aggregate.FailureCount < threshold {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		return autoBlockAdminIPFailureAggregate(tx, aggregate, window, threshold, now)
	})
}

func autoBlockAdminIPFailureAggregate(tx *gorm.DB, aggregate adminIPFailureAggregate, window time.Duration, threshold int, now time.Time) error {
	ipAddress := strings.TrimSpace(aggregate.IPAddress)
	if !isPublicIPAddress(ipAddress) {
		return nil
	}

	var existing model.IPAccessBlock
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("ip_address = ?", ipAddress).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		if existing.Status == "active" {
			return markAdminIPRiskBlocked(tx, aggregate, window, threshold, now)
		}
		if existing.RemovedAt != nil {
			refreshed, ok, err := adminIPFailureAggregateSince(tx, ipAddress, maxTime(now.Add(-window), *existing.RemovedAt), threshold)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			aggregate = refreshed
		}
	}

	triggerUsername := strings.TrimSpace(aggregate.Username)
	triggerWindow := formatCooldownDuration(window)
	reason := fmt.Sprintf("后台登录疑似撞库：账号 %s 近%s密码错误%d次，来源 IP 已自动封禁，可在服务器防护页解封", triggerUsername, triggerWindow, aggregate.FailureCount)
	row := model.IPAccessBlock{
		IPAddress:        ipAddress,
		Status:           "active",
		Reason:           reason,
		TriggerUsername:  triggerUsername,
		TriggerFailCount: aggregate.FailureCount,
		TriggerWindow:    triggerWindow,
		CreatedByAdminID: 0,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ip_address"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":              "active",
			"reason":              reason,
			"trigger_username":    triggerUsername,
			"trigger_fail_count":  aggregate.FailureCount,
			"trigger_window":      triggerWindow,
			"created_by_admin_id": 0,
			"removed_by_admin_id": nil,
			"removed_at":          nil,
			"remove_reason":       "",
			"updated_at":          now,
		}),
	}).Create(&row).Error; err != nil {
		return err
	}
	return markAdminIPRiskBlocked(tx, aggregate, window, threshold, now)
}

func adminIPFailureAggregateSince(tx *gorm.DB, ipAddress string, since time.Time, threshold int) (adminIPFailureAggregate, bool, error) {
	var aggregate adminIPFailureAggregate
	err := tx.Model(&model.AdminLoginAudit{}).
		Select("ip_address, username, COUNT(*) AS failure_count, MIN(created_at) AS window_started_at, MAX(created_at) AS latest_at").
		Where("result = ? AND stage = ? AND ip_address = ? AND created_at > ?", "failed", "password", ipAddress, since).
		Group("ip_address, username").
		Having("COUNT(*) >= ?", threshold).
		Scan(&aggregate).Error
	if err != nil {
		return aggregate, false, err
	}
	return aggregate, aggregate.FailureCount >= threshold, nil
}

func markAdminIPRiskBlocked(tx *gorm.DB, aggregate adminIPFailureAggregate, window time.Duration, threshold int, now time.Time) error {
	ipAddress := strings.TrimSpace(aggregate.IPAddress)
	if ipAddress == "" {
		return nil
	}
	var row model.LoginRiskControl
	riskKey := IPLoginRiskKey(ipAddress)
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("risk_key = ?", riskKey).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = model.LoginRiskControl{
			Scope:           LoginRiskScopeIP,
			RiskKey:         riskKey,
			Username:        strings.TrimSpace(aggregate.Username),
			IPAddress:       ipAddress,
			CreatedAt:       now,
			UpdatedAt:       now,
			WindowStartedAt: &aggregate.WindowStartedAt,
		}
	}
	row.Scope = LoginRiskScopeIP
	row.RiskKey = riskKey
	row.Username = strings.TrimSpace(aggregate.Username)
	row.IPAddress = ipAddress
	row.Status = "blocked"
	row.Level = loginRiskLevelHigh
	row.Reason = fmt.Sprintf("后台登录疑似撞库：同一账号近%s密码错误%d次，已自动应用层封禁；阈值%d次", formatCooldownDuration(window), aggregate.FailureCount, threshold)
	row.FailureCount = aggregate.FailureCount
	row.CooldownUntil = nil
	row.WindowStartedAt = &aggregate.WindowStartedAt
	return tx.Save(&row).Error
}

func isPublicIPAddress(ipAddress string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipAddress))
	if ip == nil {
		return false
	}
	return !ip.IsLoopback() &&
		!ip.IsPrivate() &&
		!ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsMulticast()
}

func maxTime(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}

func (r *LoginRiskControlRepository) adminIPFailureAggregateSince(tx *gorm.DB, ipAddress string, since time.Time) (adminIPFailureAggregate, bool, error) {
	var aggregate adminIPFailureAggregate
	err := tx.Model(&model.AdminLoginAudit{}).
		Select("ip_address, MAX(username) AS username, COUNT(*) AS failure_count, MIN(created_at) AS window_started_at, MAX(created_at) AS latest_at").
		Where("result = ? AND stage = ? AND ip_address = ? AND created_at > ?", "failed", "password", ipAddress, since).
		Group("ip_address").
		Having("COUNT(*) >= ?", 3).
		Scan(&aggregate).Error
	if err != nil {
		return aggregate, false, err
	}
	return aggregate, aggregate.FailureCount >= 3, nil
}

// ResolveAccountPasswordRisk resets account cooldown after a verified successful login.
func (r *LoginRiskControlRepository) ResolveAccountPasswordRisk(user *model.User, now time.Time) error {
	return r.db.Model(&model.LoginRiskControl{}).
		Where("risk_key = ?", AccountLoginRiskKey(user.Phone)).
		Updates(map[string]any{
			"status":         loginRiskStatusResolved,
			"level":          loginRiskLevelLow,
			"reason":         "登录成功，账号密码冷却已解除",
			"failure_count":  0,
			"success_count":  gorm.Expr("success_count + 1"),
			"cooldown_until": nil,
			"username":       user.Phone,
			"im_user_id":     user.IMUserID,
			"user_id":        user.ID,
			"updated_at":     now,
		}).Error
}

// ResolveAccountCooldownByUser clears an active account cooldown for admin operations.
func (r *LoginRiskControlRepository) ResolveAccountCooldownByUser(user *model.User, adminID uint, reason string, now time.Time) (bool, error) {
	if user == nil {
		return false, errors.New("用户不存在")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "用户列表解除登录冷却"
	}
	result := r.db.Model(&model.LoginRiskControl{}).
		Where("risk_key = ?", AccountLoginRiskKey(user.Phone)).
		Where("status = ?", loginRiskStatusCooldown).
		Where("cooldown_until IS NOT NULL AND cooldown_until > ?", now).
		Updates(map[string]any{
			"status":               loginRiskStatusResolved,
			"level":                loginRiskLevelLow,
			"reason":               "后台解除账号登录冷却",
			"failure_count":        0,
			"cooldown_until":       nil,
			"resolved_by_admin_id": adminID,
			"resolved_at":          &now,
			"resolve_reason":       reason,
			"username":             user.Phone,
			"im_user_id":           user.IMUserID,
			"user_id":              user.ID,
			"updated_at":           now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ResolveAdminCooldownByUsername clears an active admin-account cooldown for admin operations.
func (r *LoginRiskControlRepository) ResolveAdminCooldownByUsername(username string, adminID uint, reason string, now time.Time) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, errors.New("后台账号不存在")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "后台账号列表解除登录冷却"
	}
	result := r.db.Model(&model.LoginRiskControl{}).
		Where("risk_key = ?", AdminLoginRiskKey(username)).
		Where("status = ?", loginRiskStatusCooldown).
		Where("cooldown_until IS NOT NULL AND cooldown_until > ?", now).
		Updates(map[string]any{
			"status":               loginRiskStatusResolved,
			"level":                loginRiskLevelLow,
			"reason":               "后台解除后台账号登录冷却",
			"failure_count":        0,
			"cooldown_until":       nil,
			"resolved_by_admin_id": adminID,
			"resolved_at":          &now,
			"resolve_reason":       reason,
			"username":             username,
			"updated_at":           now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ResolveAdminPasswordRisk resets admin password risk after a verified successful login.
func (r *LoginRiskControlRepository) ResolveAdminPasswordRisk(username string, adminID uint, adminRole string, now time.Time) error {
	return r.db.Model(&model.LoginRiskControl{}).
		Where("risk_key = ?", AdminLoginRiskKey(username)).
		Updates(map[string]any{
			"status":         loginRiskStatusResolved,
			"level":          loginRiskLevelLow,
			"reason":         "后台登录成功，账号密码风险已解除",
			"failure_count":  0,
			"success_count":  gorm.Expr("success_count + 1"),
			"cooldown_until": nil,
			"username":       strings.TrimSpace(username),
			"im_user_id":     adminRole,
			"user_id":        nil,
			"updated_at":     now,
		}).Error
}

func lockOrCreateAccountRisk(tx *gorm.DB, user *model.User, ip string, now time.Time) (*model.LoginRiskControl, error) {
	var row model.LoginRiskControl
	riskKey := AccountLoginRiskKey(user.Phone)
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("risk_key = ?", riskKey).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		row = model.LoginRiskControl{
			Scope:           LoginRiskScopeAccount,
			RiskKey:         riskKey,
			Username:        user.Phone,
			IMUserID:        user.IMUserID,
			UserID:          &user.ID,
			IPAddress:       strings.TrimSpace(ip),
			Status:          loginRiskStatusWatching,
			Level:           loginRiskLevelLow,
			Reason:          "密码登录失败",
			FailureCount:    0,
			WindowStartedAt: &now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}

	row.Scope = LoginRiskScopeAccount
	row.Username = user.Phone
	row.IMUserID = user.IMUserID
	row.UserID = &user.ID
	row.IPAddress = strings.TrimSpace(ip)
	if row.WindowStartedAt == nil || row.Status == loginRiskStatusResolved {
		row.WindowStartedAt = &now
	}
	if row.Status == loginRiskStatusResolved {
		row.FailureCount = 0
		row.Level = loginRiskLevelLow
		row.CooldownUntil = nil
	}
	if row.Level == "" {
		row.Level = loginRiskLevelLow
	}
	return &row, nil
}

func lockOrCreateAdminRisk(tx *gorm.DB, username string, adminID *uint, adminRole, ip string, now time.Time) (*model.LoginRiskControl, error) {
	var row model.LoginRiskControl
	riskKey := AdminLoginRiskKey(username)
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("risk_key = ?", riskKey).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		row = model.LoginRiskControl{
			Scope:           LoginRiskScopeAccount,
			RiskKey:         riskKey,
			Username:        username,
			IMUserID:        adminRole,
			UserID:          adminID,
			IPAddress:       strings.TrimSpace(ip),
			Status:          loginRiskStatusWatching,
			Level:           loginRiskLevelLow,
			Reason:          "后台密码登录失败",
			FailureCount:    0,
			WindowStartedAt: &now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}

	row.Scope = LoginRiskScopeAccount
	row.Username = username
	row.IMUserID = adminRole
	row.UserID = adminID
	row.IPAddress = strings.TrimSpace(ip)
	if row.WindowStartedAt == nil || row.Status == loginRiskStatusResolved {
		row.WindowStartedAt = &now
	}
	if row.Status == loginRiskStatusResolved {
		row.FailureCount = 0
		row.Level = loginRiskLevelLow
		row.CooldownUntil = nil
	}
	if row.Level == "" {
		row.Level = loginRiskLevelLow
	}
	return &row, nil
}

func applyAdminIPFailureRiskLevel(row *model.LoginRiskControl, window time.Duration) {
	row.Status = loginRiskStatusWatching
	row.Level = loginRiskLevelLow
	row.Reason = fmt.Sprintf("后台登录失败来源，近%s内%d次", formatCooldownDuration(window), row.FailureCount)
	if row.FailureCount >= 5 {
		row.Status = "alerted"
		row.Level = loginRiskLevelMedium
		row.Reason = fmt.Sprintf("后台登录失败来源，近%s内%d次，建议人工复核是否封禁", formatCooldownDuration(window), row.FailureCount)
	}
	if row.FailureCount >= 20 {
		row.Level = loginRiskLevelHigh
	}
}

func nextAdminPasswordRisk(failureCount int, currentLevel string, hadPriorCooldown bool) (string, string, string) {
	threshold := adminPasswordFirstCooldownThreshold
	if hadPriorCooldown {
		threshold = adminPasswordLaterCooldownThreshold
	}
	level := normalizeLoginRiskLevel(currentLevel)
	if failureCount >= 20 {
		return "alerted", loginRiskLevelHigh, "后台账号密码错误达到20次，疑似撞库，请复核并封禁来源IP"
	}
	if failureCount >= threshold {
		if loginRiskLevelIndex(level) < loginRiskLevelIndex(loginRiskLevelMedium) {
			level = loginRiskLevelMedium
		}
		return "alerted", level, "后台账号密码错误接近冷却阈值，请优先复核来源IP"
	}
	return loginRiskStatusWatching, level, "后台密码登录失败，记录账号风险"
}

func nextAccountPasswordCooldown(failureCount int, currentLevel string, hadPriorCooldown bool, now time.Time) (int, string, time.Time, bool) {
	if failureCount < 5 {
		return failureCount, normalizeLoginRiskLevel(currentLevel), time.Time{}, false
	}

	nextLevelIndex := loginRiskLevelIndex(currentLevel)
	if hadPriorCooldown && nextLevelIndex < len(accountPasswordCooldownDurations)-1 {
		nextLevelIndex++
	}
	cooldownUntil := now.Add(accountPasswordCooldownDurations[nextLevelIndex])
	return 0, loginRiskLevelByIndex(nextLevelIndex), cooldownUntil, true
}

func nextAdminPasswordCooldown(failureCount int, currentLevel string, hadPriorCooldown bool, now time.Time) (int, string, time.Time, bool) {
	threshold := adminPasswordFirstCooldownThreshold
	if hadPriorCooldown {
		threshold = adminPasswordLaterCooldownThreshold
	}
	if failureCount < threshold {
		return failureCount, normalizeLoginRiskLevel(currentLevel), time.Time{}, false
	}

	nextLevelIndex := loginRiskLevelIndex(currentLevel)
	if hadPriorCooldown && nextLevelIndex < len(adminPasswordCooldownDurations)-1 {
		nextLevelIndex++
	}
	cooldownUntil := now.Add(adminPasswordCooldownDurations[nextLevelIndex])
	return 0, loginRiskLevelByIndex(nextLevelIndex), cooldownUntil, true
}

func normalizeLoginRiskLevel(level string) string {
	switch level {
	case loginRiskLevelMedium, loginRiskLevelHigh:
		return level
	default:
		return loginRiskLevelLow
	}
}

func loginRiskLevelIndex(level string) int {
	switch level {
	case loginRiskLevelMedium:
		return 1
	case loginRiskLevelHigh:
		return 2
	default:
		return 0
	}
}

func loginRiskLevelByIndex(index int) string {
	switch {
	case index <= 0:
		return loginRiskLevelLow
	case index == 1:
		return loginRiskLevelMedium
	default:
		return loginRiskLevelHigh
	}
}

func formatCooldownDuration(duration time.Duration) string {
	switch {
	case duration >= time.Hour:
		return fmt.Sprintf("%d小时", int(duration.Round(time.Hour)/time.Hour))
	default:
		minutes := int(duration.Round(time.Minute) / time.Minute)
		if minutes < 1 {
			minutes = 1
		}
		return fmt.Sprintf("%d分钟", minutes)
	}
}
