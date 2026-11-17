package controller_admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/idgs"
)

const (
	CleanupStatusWhitelisted  = "whitelisted"
	CleanupStatusManualBanned = "manual_banned"
	CleanupStatusRestored     = "restored"

	quietGroupStatusPending         = "pending"
	quietGroupStatusAutoDismissed   = "auto_dismissed"
	quietGroupStatusManualDismissed = "manual_dismissed"
	quietGroupStatusResolved        = "resolved"
	quietGroupStatusRiskOnly        = "risk_only"
	quietGroupStatusAPIFailed       = "api_failed"
	quietGroupActionManualDismiss   = "manual_dismiss_group"
)

// GovernanceOverviewResult is the operation governance dashboard summary.
type GovernanceOverviewResult struct {
	UserCount             int64 `json:"userCount"`
	GroupCount            int64 `json:"groupCount"`
	CleanupCandidateCount int64 `json:"cleanupCandidateCount"`
	LoginRiskCount        int64 `json:"loginRiskCount"`
	IPBlockCount          int64 `json:"ipBlockCount"`
	PendingReportCount    int64 `json:"pendingReportCount"`
	WhitelistCount        int64 `json:"whitelistCount"`
	ManualBannedCount     int64 `json:"manualBannedCount"`
	RestoredCount         int64 `json:"restoredCount"`
}

// CleanupStatsResult summarizes cleanup risk control state.
type CleanupStatsResult struct {
	CandidateCount     int64 `json:"candidateCount"`
	BannedCount        int64 `json:"bannedCount"`
	WhitelistCount     int64 `json:"whitelistCount"`
	ManualBannedCount  int64 `json:"manualBannedCount"`
	RestoredCount      int64 `json:"restoredCount"`
	ProtectedAAccounts int64 `json:"protectedAAccounts"`
	CreatedByNonZero   int64 `json:"createdByNonZero"`
	WithInviteCode     int64 `json:"withInviteCode"`
}

// SecurityRiskSummaryResult summarizes server-security counters with clear
// processing-oriented buckets.
type SecurityRiskSummaryResult struct {
	AccountPending       int64 `json:"accountPending"`
	AccountWatching      int64 `json:"accountWatching"`
	AccountCooldown      int64 `json:"accountCooldown"`
	AccountAlerted       int64 `json:"accountAlerted"`
	IPOpen               int64 `json:"ipOpen"`
	IPWatching           int64 `json:"ipWatching"`
	IPAlerted            int64 `json:"ipAlerted"`
	IPBlocked            int64 `json:"ipBlocked"`
	ManualBannedAccounts int64 `json:"manualBannedAccounts"`
	QuietGroupPending    int64 `json:"quietGroupPending"`
}

// CleanupCandidateResult is one cleanup workbench row.
type CleanupCandidateResult struct {
	UserResult
	Type          string `json:"type"`
	CleanupStatus string `json:"cleanupStatus"`
	Reason        string `json:"reason"`
}

// LoginRiskResult is a login risk row.
type LoginRiskResult struct {
	ID            uint   `json:"id"`
	Scope         string `json:"scope"`
	RiskKey       string `json:"riskKey"`
	Username      string `json:"username"`
	IMUserID      string `json:"imUserID"`
	UserID        *uint  `json:"userId"`
	UserStatus    string `json:"userStatus"`
	IPAddress     string `json:"ipAddress"`
	Status        string `json:"status"`
	Level         string `json:"level"`
	Reason        string `json:"reason"`
	FailureCount  int    `json:"failureCount"`
	SuccessCount  int    `json:"successCount"`
	CooldownUntil string `json:"cooldownUntil"`
	UpdatedAt     string `json:"updatedAt"`
}

// ListLoginRisksReq contains login-risk paging and filters.
type ListLoginRisksReq struct {
	Page     int
	PageSize int
	Scope    string
	Status   string
	Keyword  string
	Source   string
}

// IPAccessBlockResult is an IP block row.
type IPAccessBlockResult struct {
	ID               uint   `json:"id"`
	IPAddress        string `json:"ipAddress"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	TriggerUsername  string `json:"triggerUsername"`
	TriggerFailCount int    `json:"triggerFailCount"`
	TriggerWindow    string `json:"triggerWindow"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

// QuietHoursGroupRiskResult is a quiet-hours group risk workbench row.
type QuietHoursGroupRiskResult struct {
	ID                uint   `json:"id"`
	IMGroupID         string `json:"imGroupId"`
	GroupName         string `json:"groupName"`
	GroupType         string `json:"groupType"`
	OwnerIMUserID     string `json:"ownerImUserId"`
	OwnerID           *uint  `json:"ownerId"`
	OwnerName         string `json:"ownerName"`
	PolicyScope       string `json:"policyScope"`
	PolicyStart       string `json:"policyStart"`
	PolicyEnd         string `json:"policyEnd"`
	TencentCreateTime string `json:"tencentCreateTime"`
	TriggerSource     string `json:"triggerSource"`
	Action            string `json:"action"`
	Status            string `json:"status"`
	Reason            string `json:"reason"`
	TencentResult     string `json:"tencentResult"`
	DryRun            bool   `json:"dryRun"`
	LastCheckedAt     string `json:"lastCheckedAt"`
	DismissedAt       string `json:"dismissedAt"`
	ResolvedAt        string `json:"resolvedAt"`
	ResolveReason     string `json:"resolveReason"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// GetGovernanceOverview returns super-admin only operation overview.
func GetGovernanceOverview() (GovernanceOverviewResult, error) {
	db := datasource.Datasource.Mysql.GetClient()
	result := GovernanceOverviewResult{}
	if err := db.Model(&model.User{}).Count(&result.UserCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.Group{}).Count(&result.GroupCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.Report{}).Where("status = ?", "pending").Count(&result.PendingReportCount).Error; err != nil {
		return result, err
	}
	cleanupStats, err := GetCleanupStats()
	if err != nil {
		return result, err
	}
	result.CleanupCandidateCount = cleanupStats.CandidateCount
	result.WhitelistCount = cleanupStats.WhitelistCount
	result.ManualBannedCount = cleanupStats.ManualBannedCount
	result.RestoredCount = cleanupStats.RestoredCount
	if err := db.Model(&model.LoginRiskControl{}).
		Where("status <> ?", "resolved").
		Count(&result.LoginRiskCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.IPAccessBlock{}).
		Where("status = ?", "active").
		Count(&result.IPBlockCount).Error; err != nil {
		return result, err
	}
	return result, nil
}

// GetSecurityRiskSummary returns focused counters for the server-security page.
func GetSecurityRiskSummary() (SecurityRiskSummaryResult, error) {
	db := datasource.Datasource.Mysql.GetClient()
	result := SecurityRiskSummaryResult{}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status IN ?", repository.LoginRiskScopeAccount, []string{"cooldown", "alerted", "review_pending"}).
		Count(&result.AccountPending).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status = ?", repository.LoginRiskScopeAccount, "watching").
		Count(&result.AccountWatching).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status = ?", repository.LoginRiskScopeAccount, "cooldown").
		Count(&result.AccountCooldown).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status = ?", repository.LoginRiskScopeAccount, "alerted").
		Count(&result.AccountAlerted).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status <> ?", repository.LoginRiskScopeIP, "resolved").
		Count(&result.IPOpen).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status = ?", repository.LoginRiskScopeIP, "watching").
		Count(&result.IPWatching).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.LoginRiskControl{}).
		Where("scope = ? AND status = ?", repository.LoginRiskScopeIP, "alerted").
		Count(&result.IPAlerted).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.IPAccessBlock{}).
		Where("status = ?", "active").
		Count(&result.IPBlocked).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.CleanupUserDisposition{}).
		Where("cleanup_status = ?", CleanupStatusManualBanned).
		Count(&result.ManualBannedAccounts).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.QuietHoursGroupRisk{}).
		Where("status IN ?", []string{quietGroupStatusPending, quietGroupStatusAPIFailed, quietGroupStatusRiskOnly}).
		Count(&result.QuietGroupPending).Error; err != nil {
		return result, err
	}
	return result, nil
}

// GetCleanupStats returns cleanup risk control counters.
func GetCleanupStats() (CleanupStatsResult, error) {
	db := datasource.Datasource.Mysql.GetClient()
	result := CleanupStatsResult{}
	if err := cleanupCandidateQuery(db.Model(&model.User{})).Count(&result.CandidateCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.User{}).Where("status = ?", "banned").Count(&result.BannedCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.CleanupUserWhitelist{}).Where("status = ?", "active").Count(&result.WhitelistCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.CleanupUserDisposition{}).Where("cleanup_status = ?", CleanupStatusManualBanned).Count(&result.ManualBannedCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.CleanupUserDisposition{}).Where("cleanup_status = ?", CleanupStatusRestored).Count(&result.RestoredCount).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.User{}).Where("status = ? AND im_user_id LIKE ?", "active", "a\\_%").Count(&result.ProtectedAAccounts).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.User{}).Where("status = ? AND created_by <> 0", "active").Count(&result.CreatedByNonZero).Error; err != nil {
		return result, err
	}
	if err := db.Model(&model.User{}).Where("status = ? AND invite_code <> ''", "active").Count(&result.WithInviteCode).Error; err != nil {
		return result, err
	}
	return result, nil
}

// ListCleanupCandidates lists users matching the final cleanup policy.
func ListCleanupCandidates(page, pageSize int, keyword, currentLoginIP string) (PageResult[CleanupCandidateResult], error) {
	db := datasource.Datasource.Mysql.GetClient()
	q := cleanupCandidateQuery(db.Model(&model.User{}))
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("nickname LIKE ? OR phone LIKE ? OR im_user_id LIKE ? OR invite_code LIKE ?", like, like, like, like)
	}
	if currentLoginIP = strings.TrimSpace(currentLoginIP); currentLoginIP != "" {
		q = q.Where(`EXISTS (
			SELECT 1 FROM user_refresh_sessions urs
			WHERE urs.user_id = users.id
			  AND urs.revoked_at IS NULL
			  AND urs.ip = ?
		)`, currentLoginIP)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[CleanupCandidateResult]{}, err
	}
	var users []model.User
	if err := q.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return PageResult[CleanupCandidateResult]{}, err
	}
	userResults, err := buildUserResultList(users)
	if err != nil {
		return PageResult[CleanupCandidateResult]{}, err
	}
	items := make([]CleanupCandidateResult, 0, len(userResults))
	for _, user := range userResults {
		items = append(items, CleanupCandidateResult{
			UserResult:    user,
			Type:          "no_invite_expired",
			CleanupStatus: user.Status,
			Reason:        "普通注册无邀请码超过30分钟，非批量/后台创建且不在白名单",
		})
	}
	return PageResult[CleanupCandidateResult]{List: items, Total: total}, nil
}

func cleanupCandidateQuery(q *gorm.DB) *gorm.DB {
	return q.Where("status = ?", "active").
		Where("(invite_code = '' OR invite_code IS NULL)").
		Where("created_at < ?", time.Now().Add(-30*time.Minute)).
		Where("created_by = 0").
		Where("(im_user_id NOT LIKE ? OR im_user_id IS NULL)", "a\\_%").
		Where(`NOT EXISTS (
			SELECT 1
			FROM cleanup_user_whitelists cw
			WHERE cw.status = 'active'
			  AND (
			    cw.user_id = users.id
			    OR (cw.phone <> '' AND cw.phone = COALESCE(users.phone, ''))
			    OR (cw.im_user_id <> '' AND cw.im_user_id = COALESCE(users.im_user_id, ''))
			  )
		)`)
}

// AddCleanupWhitelist protects a user from cleanup.
func AddCleanupWhitelist(actorID, userID uint, reason string) error {
	db := datasource.Datasource.Mysql.GetClient()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return err
	}
	ownerID := user.OwnerID
	return db.Transaction(func(tx *gorm.DB) error {
		adminID := actorID
		item := model.CleanupUserWhitelist{
			UserID:           &user.ID,
			Phone:            strings.TrimSpace(user.Phone),
			IMUserID:         strings.TrimSpace(user.IMUserID),
			Nickname:         strings.TrimSpace(user.Nickname),
			OwnerID:          &ownerID,
			Reason:           strings.TrimSpace(reason),
			Status:           "active",
			CreatedByAdminID: &adminID,
		}
		var existing model.CleanupUserWhitelist
		err := tx.Where("user_id = ?", user.ID).First(&existing).Error
		switch {
		case err == nil:
			if err := tx.Model(&existing).Updates(map[string]any{
				"phone":               item.Phone,
				"im_user_id":          item.IMUserID,
				"nickname":            item.Nickname,
				"owner_id":            item.OwnerID,
				"reason":              item.Reason,
				"status":              item.Status,
				"created_by_admin_id": item.CreatedByAdminID,
			}).Error; err != nil {
				return err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		default:
			return err
		}
		return upsertCleanupDisposition(tx, &user, CleanupStatusWhitelisted, item.Reason, actorID)
	})
}

// BanCleanupUser manually bans a cleanup candidate. It is recoverable.
func BanCleanupUser(ctx context.Context, actorID, userID uint, reason string) error {
	db := datasource.Datasource.Mysql.GetClient()
	var user model.User
	if err := cleanupCandidateQuery(db.Model(&model.User{})).Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("该用户当前不符合自动清理候选条件，未执行封禁")
		}
		return err
	}
	if err := BanUser(ctx, actorID, true, userID); err != nil {
		return err
	}
	return upsertCleanupDisposition(db, &user, CleanupStatusManualBanned, strings.TrimSpace(reason), actorID)
}

// RestoreCleanupUser restores a banned user.
func RestoreCleanupUser(ctx context.Context, actorID, userID uint, reason string) error {
	db := datasource.Datasource.Mysql.GetClient()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return err
	}
	if err := UnbanUser(ctx, actorID, true, userID); err != nil {
		return err
	}
	return upsertCleanupDisposition(db, &user, CleanupStatusRestored, strings.TrimSpace(reason), actorID)
}

func upsertCleanupDisposition(db *gorm.DB, user *model.User, status, reason string, actorID uint) error {
	ownerID := user.OwnerID
	adminID := actorID
	row := model.CleanupUserDisposition{
		UserID:           user.ID,
		IMUserID:         strings.TrimSpace(user.IMUserID),
		Phone:            strings.TrimSpace(user.Phone),
		Nickname:         strings.TrimSpace(user.Nickname),
		OwnerID:          &ownerID,
		CleanupStatus:    status,
		Reason:           reason,
		CreatedByAdminID: &adminID,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"im_user_id", "phone", "nickname", "owner_id", "cleanup_status", "reason", "created_by_admin_id", "updated_at",
		}),
	}).Create(&row).Error
}

// ListLoginRisks returns login risks.
func ListLoginRisks(req ListLoginRisksReq) (PageResult[LoginRiskResult], error) {
	db := datasource.Datasource.Mysql.GetClient()
	page := req.Page
	pageSize := req.PageSize
	scope := strings.TrimSpace(req.Scope)
	status := strings.TrimSpace(req.Status)
	if strings.TrimSpace(scope) == repository.LoginRiskScopeIP && page <= 1 {
		if err := repository.NewLoginRiskControlRepository().SyncRecentAdminIPLoginFailures(24*time.Hour, time.Now()); err != nil {
			return PageResult[LoginRiskResult]{}, err
		}
	}
	q := db.Model(&model.LoginRiskControl{})
	if scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if status == "pending" {
		q = q.Where("status IN ?", []string{"cooldown", "alerted", "review_pending"})
	} else if status != "" && status != "open" {
		q = q.Where("status = ?", status)
	} else {
		q = q.Where("status <> ?", "resolved")
	}
	if source := strings.TrimSpace(req.Source); scope == repository.LoginRiskScopeAccount && source != "" && source != "all" {
		switch source {
		case "admin":
			q = q.Where("risk_key LIKE ?", "admin:%")
		case "user":
			q = q.Where("risk_key NOT LIKE ?", "admin:%")
		}
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where(
			"username LIKE ? OR im_user_id LIKE ? OR ip_address LIKE ? OR risk_key LIKE ? OR CAST(user_id AS CHAR) = ?",
			like, like, like, like, keyword,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[LoginRiskResult]{}, err
	}
	var rows []model.LoginRiskControl
	if err := q.Order(`FIELD(status, 'review_pending', 'cooldown', 'alerted', 'watching', 'resolved'), failure_count DESC, updated_at DESC`).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return PageResult[LoginRiskResult]{}, err
	}
	userIDs := make([]uint, 0, len(rows))
	seenUserIDs := map[uint]bool{}
	for _, row := range rows {
		if row.UserID != nil && !seenUserIDs[*row.UserID] {
			userIDs = append(userIDs, *row.UserID)
			seenUserIDs[*row.UserID] = true
		}
	}
	userStatusMap := map[uint]string{}
	if len(userIDs) > 0 {
		var users []model.User
		if err := db.Select("id", "status").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return PageResult[LoginRiskResult]{}, err
		}
		for _, user := range users {
			userStatusMap[user.ID] = user.Status
		}
	}
	items := make([]LoginRiskResult, 0, len(rows))
	for _, row := range rows {
		cooldownUntil := ""
		if row.CooldownUntil != nil {
			cooldownUntil = row.CooldownUntil.Format(time.RFC3339)
		}
		userStatus := ""
		if row.UserID != nil {
			userStatus = userStatusMap[*row.UserID]
		}
		items = append(items, LoginRiskResult{
			ID: row.ID, Scope: row.Scope, RiskKey: row.RiskKey, Username: row.Username,
			IMUserID: row.IMUserID, UserID: row.UserID, IPAddress: row.IPAddress,
			UserStatus: userStatus, Status: row.Status, Level: row.Level, Reason: row.Reason,
			FailureCount: row.FailureCount, SuccessCount: row.SuccessCount,
			CooldownUntil: cooldownUntil, UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
		})
	}
	return PageResult[LoginRiskResult]{List: items, Total: total}, nil
}

// ResolveLoginRisk marks a risk as resolved.
func ResolveLoginRisk(actorID, id uint, reason string) error {
	now := time.Now()
	return datasource.Datasource.Mysql.GetClient().Model(&model.LoginRiskControl{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":               "resolved",
			"failure_count":        0,
			"cooldown_until":       nil,
			"resolved_by_admin_id": actorID,
			"resolved_at":          &now,
			"resolve_reason":       strings.TrimSpace(reason),
		}).Error
}

// ListIPAccessBlocks returns application IP blocks.
func ListIPAccessBlocks(page, pageSize int, status, ipAddress string) (PageResult[IPAccessBlockResult], error) {
	db := datasource.Datasource.Mysql.GetClient()
	q := db.Model(&model.IPAccessBlock{})
	if status = strings.TrimSpace(status); status != "" {
		q = q.Where("status = ?", status)
	}
	if ipAddress = strings.TrimSpace(ipAddress); ipAddress != "" {
		q = q.Where("ip_address = ?", ipAddress)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[IPAccessBlockResult]{}, err
	}
	var rows []model.IPAccessBlock
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[IPAccessBlockResult]{}, err
	}
	items := make([]IPAccessBlockResult, 0, len(rows))
	for _, row := range rows {
		items = append(items, IPAccessBlockResult{
			ID: row.ID, IPAddress: row.IPAddress, Status: row.Status, Reason: row.Reason,
			TriggerUsername: row.TriggerUsername, TriggerFailCount: row.TriggerFailCount, TriggerWindow: row.TriggerWindow,
			CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
		})
	}
	return PageResult[IPAccessBlockResult]{List: items, Total: total}, nil
}

// BlockIP creates or reactivates an application-layer IP block.
func BlockIP(actorID uint, ipAddress, reason string) error {
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return errors.New("IP 不能为空")
	}
	row := model.IPAccessBlock{
		IPAddress:        ipAddress,
		Status:           "active",
		Reason:           strings.TrimSpace(reason),
		CreatedByAdminID: actorID,
	}
	return datasource.Datasource.Mysql.GetClient().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ip_address"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "reason", "created_by_admin_id", "removed_by_admin_id", "removed_at", "remove_reason", "updated_at",
		}),
	}).Create(&row).Error
}

// UnblockIP disables an application-layer IP block.
func UnblockIP(actorID, id uint, reason string) error {
	now := time.Now()
	return datasource.Datasource.Mysql.GetClient().Model(&model.IPAccessBlock{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":              "removed",
			"removed_by_admin_id": actorID,
			"removed_at":          &now,
			"remove_reason":       strings.TrimSpace(reason),
		}).Error
}

// ListQuietHoursGroupRisks returns quiet-hours group risk rows.
func ListQuietHoursGroupRisks(page, pageSize int, keyword, status string, ownerID uint) (PageResult[QuietHoursGroupRiskResult], error) {
	db := datasource.Datasource.Mysql.GetClient()
	q := db.Model(&model.QuietHoursGroupRisk{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("im_group_id LIKE ? OR group_name LIKE ? OR owner_im_user_id LIKE ? OR reason LIKE ?", like, like, like, like)
	}
	if status = strings.TrimSpace(status); status != "" {
		if status == "open" {
			q = q.Where("status IN ?", []string{quietGroupStatusPending, quietGroupStatusAPIFailed, quietGroupStatusRiskOnly})
		} else {
			q = q.Where("status = ?", status)
		}
	}
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[QuietHoursGroupRiskResult]{}, err
	}
	var rows []model.QuietHoursGroupRisk
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[QuietHoursGroupRiskResult]{}, err
	}

	ownerIDs := make([]uint, 0)
	seenOwners := map[uint]bool{}
	for _, row := range rows {
		if row.OwnerID != nil && !seenOwners[*row.OwnerID] {
			ownerIDs = append(ownerIDs, *row.OwnerID)
			seenOwners[*row.OwnerID] = true
		}
	}
	ownerNameMap := map[uint]string{}
	if len(ownerIDs) > 0 {
		admins, _ := repository.NewAdminUserRepository().FindByIDs(ownerIDs)
		for _, admin := range admins {
			ownerNameMap[admin.ID] = admin.Username
		}
	}

	items := make([]QuietHoursGroupRiskResult, 0, len(rows))
	for _, row := range rows {
		result := QuietHoursGroupRiskResult{
			ID: row.ID, IMGroupID: row.IMGroupID, GroupName: row.GroupName, GroupType: row.GroupType,
			OwnerIMUserID: row.OwnerIMUserID, OwnerID: row.OwnerID, PolicyScope: row.PolicyScope,
			PolicyStart: row.PolicyStart, PolicyEnd: row.PolicyEnd, TriggerSource: row.TriggerSource,
			Action: row.Action, Status: row.Status, Reason: row.Reason, TencentResult: row.TencentResult,
			DryRun: row.DryRun, ResolveReason: row.ResolveReason,
			CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
		}
		if row.OwnerID != nil {
			result.OwnerName = ownerNameMap[*row.OwnerID]
		}
		if row.TencentCreateTime != nil {
			result.TencentCreateTime = row.TencentCreateTime.Format(time.RFC3339)
		}
		if row.LastCheckedAt != nil {
			result.LastCheckedAt = row.LastCheckedAt.Format(time.RFC3339)
		}
		if row.DismissedAt != nil {
			result.DismissedAt = row.DismissedAt.Format(time.RFC3339)
		}
		if row.ResolvedAt != nil {
			result.ResolvedAt = row.ResolvedAt.Format(time.RFC3339)
		}
		items = append(items, result)
	}
	return PageResult[QuietHoursGroupRiskResult]{List: items, Total: total}, nil
}

// ResolveQuietHoursGroupRisk marks one group risk as resolved without changing Tencent group state.
func ResolveQuietHoursGroupRisk(actorID, id uint, reason string) error {
	now := time.Now()
	return datasource.Datasource.Mysql.GetClient().Model(&model.QuietHoursGroupRisk{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":               quietGroupStatusResolved,
			"resolved_by_admin_id": actorID,
			"resolved_at":          &now,
			"resolve_reason":       strings.TrimSpace(reason),
		}).Error
}

// SyncQuietHoursGroupRisk refreshes local group data for one risk row.
func SyncQuietHoursGroupRisk(ctx context.Context, id uint) error {
	var row model.QuietHoursGroupRisk
	if err := datasource.Datasource.Mysql.GetClient().First(&row, id).Error; err != nil {
		return err
	}
	_ = ctx
	return syncQuietHoursGroupFromTencent(row.IMGroupID)
}

// DismissQuietHoursGroupRisk manually dismisses one risky group.
func DismissQuietHoursGroupRisk(actorID, id uint, reason string) error {
	db := datasource.Datasource.Mysql.GetClient()
	var row model.QuietHoursGroupRisk
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	if row.Status == quietGroupStatusAutoDismissed || row.Status == quietGroupStatusManualDismissed {
		return errors.New("该群已解散")
	}
	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return errors.New("腾讯 IM 未配置，无法解散群")
	}
	if err := imClient.GroupDestroy(row.IMGroupID); err != nil {
		return err
	}
	if row.LocalGroupID != nil {
		_ = db.Model(&model.Group{}).Where("id = ?", *row.LocalGroupID).Update("status", "dismissed").Error
	} else if row.IMGroupID != "" {
		_ = db.Model(&model.Group{}).Where("im_group_id = ?", row.IMGroupID).Update("status", "dismissed").Error
	}
	now := time.Now()
	return db.Model(&row).Updates(map[string]any{
		"status":                 quietGroupStatusManualDismissed,
		"action":                 quietGroupActionManualDismiss,
		"reason":                 strings.TrimSpace(reason),
		"dismissed_at":           &now,
		"manual_action_admin_id": actorID,
		"last_checked_at":        &now,
	}).Error
}

func syncQuietHoursGroupFromTencent(groupID string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("群 ID 不能为空")
	}
	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return errors.New("腾讯 IM 未配置，无法补同步")
	}
	infoMap, err := imClient.GroupGetInfo([]string{groupID})
	if err != nil {
		return err
	}
	info, ok := infoMap[groupID]
	if !ok {
		return errors.New("腾讯 IM 未返回群资料")
	}
	ownerID := uint(0)
	createdBy := uint(0)
	if ownerAccount := strings.TrimSpace(info.OwnerAccount); ownerAccount != "" {
		if ownerUser, findErr := repository.NewUserRepository().FindByIMUserID(ownerAccount); findErr == nil {
			ownerID = ownerUser.OwnerID
			createdBy = ownerUser.ID
		}
	}
	groupName := strings.TrimSpace(info.Name)
	if groupName == "" {
		groupName = groupID
	}
	groupType := strings.TrimSpace(info.Type)
	if groupType == "" {
		groupType = "Work"
	}
	createdAt := time.Now()
	if info.CreateTime > 0 {
		createdAt = time.Unix(info.CreateTime, 0)
	}

	db := datasource.Datasource.Mysql.GetClient()
	var group model.Group
	err = db.Where("im_group_id = ?", groupID).First(&group).Error
	switch {
	case err == nil:
		group.Name = groupName
		group.GroupType = groupType
		if createdBy > 0 {
			group.CreatedBy = createdBy
		}
		return repository.NewGroupRepositoryWithDB(db).SaveMirror(context.Background(), &group)
	case errors.Is(err, gorm.ErrRecordNotFound):
		return repository.NewGroupRepositoryWithDB(db).SaveMirror(context.Background(), &model.Group{
			BizID: idgs.Global.GroupGenerate().Int64(), IMGroupID: groupID, Name: groupName, GroupType: groupType,
			OwnerID: ownerID, CreatedBy: createdBy, Status: "active", CreatedAt: createdAt,
		})
	default:
		return err
	}
}
