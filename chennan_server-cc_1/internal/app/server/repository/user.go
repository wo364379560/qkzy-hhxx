package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// UserRepository 处理 users 表的数据库操作。
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserRepositoryWithDB(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByPhone(phone string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}

func (r *UserRepository) Update(user *model.User) error {
	if user == nil || user.ID == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.Model(&model.User{}).Where("id = ?", user.ID).Select("*").
		Omit("id", "owner_id", "invite_code", "owner_revision", "created_by", "created_at").Updates(user).Error
}

// UpdateFields 更新用户指定字段。
func (r *UserRepository) UpdateFields(id uint, fields map[string]any) error {
	for key := range fields {
		switch strings.ToLower(strings.ReplaceAll(key, "_", "")) {
		case "ownerid", "invitecode", "ownerrevision", "createdby":
			return ErrOwnershipProtected
		}
	}
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete marks a user as deleted while preserving local business data.
func (r *UserRepository) SoftDelete(id uint) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         "deleted",
			"phone":          gorm.Expr("CASE WHEN phone LIKE 'deleted\\_%' THEN phone ELSE LEFT(CONCAT('deleted_', id, '_', phone), 32) END"),
			"auth_version":   gorm.Expr("GREATEST(auth_version, 1) + 1"),
			"last_online_at": nil,
		}).Error
}

func (r *UserRepository) TouchLastOnlineAt(userID uint, now time.Time) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", userID).
		Where("last_presence_status <> ? OR last_presence_status IS NULL OR last_online_at IS NULL OR last_online_at < ?", "online", now.Add(-5*time.Minute)).
		Updates(map[string]any{
			"last_online_at":           now,
			"last_presence_status":     "online",
			"last_presence_checked_at": now,
		}).Error
}

func (r *UserRepository) TouchLastOnlineAtByIMUserID(imUserID string, now time.Time) error {
	imUserID = strings.TrimSpace(imUserID)
	if imUserID == "" {
		return nil
	}
	return r.db.Model(&model.User{}).
		Where("im_user_id = ? AND status = ?", imUserID, "active").
		Where("last_presence_status <> ? OR last_presence_status IS NULL OR last_online_at IS NULL OR last_online_at < ?", "online", now.Add(-5*time.Minute)).
		Where("last_presence_checked_at IS NULL OR last_presence_checked_at <= ?", now).
		Updates(map[string]any{
			"last_online_at":           now,
			"last_presence_status":     "online",
			"last_presence_checked_at": now,
		}).Error
}

func (r *UserRepository) TouchOfflineAtByIMUserID(imUserID string, at time.Time) error {
	imUserID = strings.TrimSpace(imUserID)
	if imUserID == "" {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	return r.db.Model(&model.User{}).
		Where("im_user_id = ? AND status = ?", imUserID, "active").
		Where("last_presence_checked_at IS NULL OR last_presence_checked_at <= ?", at).
		Updates(map[string]any{
			"last_online_at":           at,
			"last_offline_at":          at,
			"last_presence_status":     "offline",
			"last_presence_checked_at": at,
		}).Error
}

func (r *UserRepository) BatchTouchLastOnlineAtByIMUserIDs(imUserIDs []string, now time.Time) error {
	imUserIDs = uniqueNonEmptyStrings(imUserIDs)
	if len(imUserIDs) == 0 {
		return nil
	}
	return r.db.Model(&model.User{}).
		Where("im_user_id IN ?", imUserIDs).
		Where("status = ?", "active").
		Where("last_presence_status <> ? OR last_presence_status IS NULL OR last_online_at IS NULL OR last_online_at < ?", "online", now.Add(-5*time.Minute)).
		Where("last_presence_checked_at IS NULL OR last_presence_checked_at <= ?", now).
		Updates(map[string]any{
			"last_online_at":           now,
			"last_presence_status":     "online",
			"last_presence_checked_at": now,
		}).Error
}

// BatchRecordPresenceStatusByIMUserIDs records the latest queried Tencent IM presence state.
// It intentionally does not update last_online_at during status polling. That
// field is updated by real user activity such as login, refresh token, and
// successful business APIs, and by explicit Tencent offline state callbacks.
// Offline polling writes last_offline_at only for an online -> offline transition
// to avoid high-frequency writes for users that are already offline.
func (r *UserRepository) BatchRecordPresenceStatusByIMUserIDs(onlineIMUserIDs, offlineIMUserIDs []string, now time.Time) error {
	if err := r.batchMarkPresenceOnline(onlineIMUserIDs, now); err != nil {
		return err
	}
	return r.batchMarkPresenceOffline(offlineIMUserIDs, now)
}

func (r *UserRepository) batchMarkPresenceOnline(imUserIDs []string, now time.Time) error {
	imUserIDs = uniqueNonEmptyStrings(imUserIDs)
	if len(imUserIDs) == 0 {
		return nil
	}
	return r.db.Model(&model.User{}).
		Where("im_user_id IN ?", imUserIDs).
		Where("status = ?", "active").
		Where(`
			last_presence_status <> ?
			OR last_presence_status IS NULL
			OR last_presence_checked_at IS NULL
			OR last_presence_checked_at < ?
		`, "online", now.Add(-5*time.Minute)).
		Where("last_presence_checked_at IS NULL OR last_presence_checked_at <= ?", now).
		Updates(map[string]any{
			"last_presence_status":     "online",
			"last_presence_checked_at": now,
		}).Error
}

func (r *UserRepository) batchMarkPresenceOffline(imUserIDs []string, now time.Time) error {
	imUserIDs = uniqueNonEmptyStrings(imUserIDs)
	if len(imUserIDs) == 0 {
		return nil
	}
	return r.db.Model(&model.User{}).
		Where("im_user_id IN ?", imUserIDs).
		Where("status = ?", "active").
		Where("last_presence_status = ?", "online").
		Where("last_presence_checked_at IS NULL OR last_presence_checked_at <= ?", now).
		Updates(map[string]any{
			"last_presence_status":     "offline",
			"last_presence_checked_at": now,
			"last_offline_at":          now,
		}).Error
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (r *UserRepository) buildListQuery(scopeOwnerIDs []uint, ownerID uint, keyword, status, currentLoginIP string, isActivated *bool, ipWhitelistEnabled *bool) *gorm.DB {
	baseQ := r.db.Model(&model.User{})
	baseQ = baseQ.Where("users.status <> ?", "deleted")
	if status == "active" || status == "banned" {
		baseQ = baseQ.Joins("LEFT JOIN admin_users owners ON owners.id = users.owner_id")
	}
	if ownerID > 0 {
		baseQ = baseQ.Where("users.owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		baseQ = baseQ.Where("users.owner_id IN ?", scopeOwnerIDs)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		baseQ = baseQ.Where("users.nickname LIKE ? OR users.phone LIKE ? OR users.im_user_id LIKE ? OR users.wanan_id LIKE ?", like, like, like, like)
	}
	if status != "" {
		switch status {
		case "active":
			baseQ = baseQ.Where("users.status = ? AND (owners.id IS NULL OR owners.status = ?)", "active", "active")
		case "banned":
			baseQ = baseQ.Where("users.status = ? OR (users.status = ? AND owners.status <> ?)", "banned", "active", "active")
		default:
			baseQ = baseQ.Where("users.status = ?", status)
		}
	}
	if isActivated != nil {
		baseQ = baseQ.Where("users.is_activated = ?", *isActivated)
	}
	if ipWhitelistEnabled != nil {
		baseQ = baseQ.Where("users.ip_whitelist_enabled = ?", *ipWhitelistEnabled)
	}
	if currentLoginIP = strings.TrimSpace(currentLoginIP); currentLoginIP != "" {
		baseQ = baseQ.Where(`
			EXISTS (
				SELECT 1
				FROM user_refresh_sessions urs
				WHERE urs.user_id = users.id
				  AND urs.revoked_at IS NULL
				  AND urs.expires_at > ?
				  AND urs.ip = ?
			)
		`, time.Now(), currentLoginIP)
	}
	return baseQ
}

// ListByOwner 不分页（供内部使用）。
func (r *UserRepository) ListByOwner(ownerID uint) ([]model.User, error) {
	var users []model.User
	err := r.db.Where("owner_id = ?", ownerID).Find(&users).Error
	return users, err
}

// ownerUserCount 用于 CountByOwners 的扫描结构。
type ownerUserCount struct {
	OwnerID uint  `gorm:"column:owner_id"`
	Total   int64 `gorm:"column:total"`
}

// CountByOwners 批量统计每个 owner 下的用户总数。
func (r *UserRepository) CountByOwners(ownerIDs []uint) (map[uint]int64, error) {
	if len(ownerIDs) == 0 {
		return map[uint]int64{}, nil
	}
	var rows []ownerUserCount
	err := r.db.Model(&model.User{}).
		Select("owner_id, COUNT(*) AS total").
		Where("owner_id IN ?", ownerIDs).
		Group("owner_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint]int64, len(rows))
	for _, row := range rows {
		m[row.OwnerID] = row.Total
	}
	return m, nil
}

// ListPaged 分页查询用户列表。ownerID=0 时使用 scopeOwnerIDs 过滤；scopeOwnerIDs 为空表示不限制归属。
// status 对 active/banned 会按实际登录状态过滤：若归属人被禁用，则其名下 active 用户也视为 banned。
func (r *UserRepository) ListPaged(page, pageSize int, scopeOwnerIDs []uint, ownerID uint, keyword, status, currentLoginIP string, isActivated *bool, ipWhitelistEnabled *bool) ([]model.User, int64, error) {
	baseQ := r.buildListQuery(scopeOwnerIDs, ownerID, keyword, status, currentLoginIP, isActivated, ipWhitelistEnabled)
	var total int64
	if err := baseQ.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	offset := (page - 1) * pageSize
	err := baseQ.Session(&gorm.Session{}).
		Select("users.*").
		Order("users.id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error
	return users, total, err
}

// ListForExport 查询符合筛选条件的全部用户，用于导出。
func (r *UserRepository) ListForExport(scopeOwnerIDs []uint, ownerID uint, keyword, status, currentLoginIP string, isActivated *bool, ipWhitelistEnabled *bool) ([]model.User, error) {
	baseQ := r.buildListQuery(scopeOwnerIDs, ownerID, keyword, status, currentLoginIP, isActivated, ipWhitelistEnabled)
	var users []model.User
	err := baseQ.Session(&gorm.Session{}).
		Select("users.*").
		Order("users.id DESC").
		Find(&users).Error
	return users, err
}

// FindByIMUserID 通过 IM UserID（如 u_1003ab7k）查找用户。
func (r *UserRepository) FindByIMUserID(imUserID string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("im_user_id = ?", imUserID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ExistsByPhone 检查手机号是否已注册。
func (r *UserRepository) ExistsByPhone(phone string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("phone = ?", phone).Count(&count).Error
	return count > 0, err
}

// ExistsByWananID 检查万安号是否已被占用。
func (r *UserRepository) ExistsByWananID(wananID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("wanan_id = ?", wananID).Count(&count).Error
	return count > 0, err
}

// UpdateStatus 更新用户状态（active / banned / deleted）。
func (r *UserRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

// FindByIMUserIDs 批量通过 IM UserID 查找用户。
func (r *UserRepository) FindByIMUserIDs(imUserIDs []string) ([]model.User, error) {
	if len(imUserIDs) == 0 {
		return nil, nil
	}
	var users []model.User
	err := r.db.Where("im_user_id IN ?", imUserIDs).Find(&users).Error
	return users, err
}

// ListKeywordFilterTargetIMUserIDs 查询需要聊天关键词屏蔽的普通无邀请码用户 IM UserID。
func (r *UserRepository) ListKeywordFilterTargetIMUserIDs() ([]string, error) {
	var imUserIDs []string
	err := r.db.Model(&model.User{}).
		Where("status = ?", "active").
		Where("(invite_code IS NULL OR TRIM(invite_code) = '')").
		Where("(im_user_id NOT LIKE ? OR im_user_id IS NULL)", "a\\_%").
		Where("im_user_id <> ?", "").
		Pluck("im_user_id", &imUserIDs).Error
	return imUserIDs, err
}

// FindAll 查询所有用户（不分页）。
func (r *UserRepository) FindAll() ([]model.User, error) {
	var users []model.User
	err := r.db.Find(&users).Error
	return users, err
}

// FindAllByOwners 查询指定归属人的全部用户（不分页）。
func (r *UserRepository) FindAllByOwners(ownerIDs []uint) ([]model.User, error) {
	if len(ownerIDs) == 0 {
		return []model.User{}, nil
	}
	var users []model.User
	err := r.db.Where("owner_id IN ?", ownerIDs).Find(&users).Error
	return users, err
}

// SearchByOwner 按手机号或弥络蓝序号精确搜索，排除调用者自身，最多返回 20 条。
// 归属与全局搜索可见性由调用方用 CanSearchUser 过滤。
func (r *UserRepository) SearchByOwner(keyword, selfIMUserID string) ([]model.User, error) {
	return r.SearchByOwnerExact(keyword, selfIMUserID)
}

// SearchByOwnerExact 精确搜索手机号或弥络蓝序号。
// 归属与全局搜索可见性由调用方用 CanSearchUser 过滤。
func (r *UserRepository) SearchByOwnerExact(keyword, selfIMUserID string) ([]model.User, error) {
	trimmed := strings.TrimSpace(keyword)
	if trimmed == "" {
		return []model.User{}, nil
	}
	normalized := strings.ToLower(trimmed)
	var users []model.User
	err := r.db.
		Where("status = 'active' AND im_user_id != ?", selfIMUserID).
		Where("phone = ? OR wanan_id = ?", normalized, normalized).
		Limit(20).
		Find(&users).Error
	return users, err
}

// SearchForIMAdmin 在后台按昵称、手机号、万安号或 IM UserID 搜索用户候选。
// ownerID=0 时使用 scopeOwnerIDs 过滤；scopeOwnerIDs 为空表示全量搜索。
// 该接口面向 IM 管理搜索框，约定按昵称做模糊搜索。
func (r *UserRepository) SearchForIMAdmin(scopeOwnerIDs []uint, ownerID uint, keyword string, limit int) ([]model.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	like := "%" + keyword + "%"
	q := r.db.Model(&model.User{}).Where("im_user_id <> ''")
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", scopeOwnerIDs)
	}

	var users []model.User
	if strings.TrimSpace(keyword) == "" {
		err := q.
			Order("nickname ASC, id DESC").
			Limit(limit).
			Find(&users).Error
		return users, err
	}

	err := q.
		Where("nickname LIKE ? OR phone LIKE ? OR im_user_id LIKE ? OR wanan_id LIKE ?", like, like, like, like).
		Order("nickname ASC, id DESC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

// Count 统计用户总数。
func (r *UserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

// CountByOwner 统计某 owner 下的用户数量。
func (r *UserRepository) CountByOwner(ownerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("owner_id = ?", ownerID).Count(&count).Error
	return count, err
}
