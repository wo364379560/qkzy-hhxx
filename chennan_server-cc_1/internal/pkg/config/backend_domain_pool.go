package config

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

const backendDomainPoolSettingKey = "backend_domain_pool_settings"
const backendDomainPoolServerCacheTTL = 30 * time.Second

var backendDomainPoolCache = struct {
	sync.RWMutex
	settings  BackendDomainPoolSettings
	expiresAt time.Time
	hasValue  bool
}{}

// BackendDomainPoolSettings 表示下发给 App 的业务域名池配置。
type BackendDomainPoolSettings struct {
	Version    int      `json:"version"`
	TTLSeconds int      `json:"ttl_seconds"`
	APIOrigins []string `json:"api_origins"`
}

func defaultBackendDomainPoolSettings() BackendDomainPoolSettings {
	return BackendDomainPoolSettings{
		Version:    0,
		TTLSeconds: 3600,
		// 新部署且数据库尚未保存域名池时，仍向 App 下发可用主域。
		APIOrigins: []string{
			"https://api.ruizhihe.com",
			"https://api.prfgsu.com",
			"https://api.uktjwg.cn",
		},
	}
}

// GetBackendDomainPoolSettings 获取业务域名池配置。
func GetBackendDomainPoolSettings(ctx context.Context, db *gorm.DB) (BackendDomainPoolSettings, error) {
	now := time.Now()
	backendDomainPoolCache.RLock()
	if backendDomainPoolCache.hasValue && now.Before(backendDomainPoolCache.expiresAt) {
		settings := backendDomainPoolCache.settings
		backendDomainPoolCache.RUnlock()
		return settings, nil
	}
	backendDomainPoolCache.RUnlock()

	backendDomainPoolCache.Lock()
	defer backendDomainPoolCache.Unlock()

	now = time.Now()
	if backendDomainPoolCache.hasValue && now.Before(backendDomainPoolCache.expiresAt) {
		return backendDomainPoolCache.settings, nil
	}

	settings, err := loadBackendDomainPoolSettings(ctx, db)
	if err != nil {
		return BackendDomainPoolSettings{}, err
	}

	backendDomainPoolCache.settings = settings
	backendDomainPoolCache.expiresAt = now.Add(backendDomainPoolServerCacheTTL)
	backendDomainPoolCache.hasValue = true

	return settings, nil
}

func loadBackendDomainPoolSettings(ctx context.Context, db *gorm.DB) (BackendDomainPoolSettings, error) {
	if db == nil {
		return BackendDomainPoolSettings{}, errors.New("mysql datasource not initialized")
	}

	var setting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", backendDomainPoolSettingKey).
		Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultBackendDomainPoolSettings(), nil
		}
		return BackendDomainPoolSettings{}, err
	}

	result := defaultBackendDomainPoolSettings()
	if strings.TrimSpace(setting.Value) == "" {
		return result, nil
	}
	if err := json.Unmarshal([]byte(setting.Value), &result); err != nil {
		return BackendDomainPoolSettings{}, err
	}
	if err := NormalizeBackendDomainPoolSettings(&result); err != nil {
		return BackendDomainPoolSettings{}, err
	}
	if len(result.APIOrigins) == 0 {
		result.APIOrigins = defaultBackendDomainPoolSettings().APIOrigins
	}
	return result, nil
}

func invalidateBackendDomainPoolSettingsCache() {
	backendDomainPoolCache.Lock()
	backendDomainPoolCache.hasValue = false
	backendDomainPoolCache.expiresAt = time.Time{}
	backendDomainPoolCache.Unlock()
}

// SetBackendDomainPoolSettingsTx 在事务中保存业务域名池配置。
func SetBackendDomainPoolSettingsTx(
	ctx context.Context,
	db *gorm.DB,
	settings BackendDomainPoolSettings,
	updatedBy uint,
) (BackendDomainPoolSettings, error) {
	if db == nil {
		return BackendDomainPoolSettings{}, errors.New("mysql datasource not initialized")
	}
	if err := NormalizeBackendDomainPoolSettings(&settings); err != nil {
		return BackendDomainPoolSettings{}, err
	}

	current, err := loadBackendDomainPoolSettings(ctx, db)
	if err != nil {
		return BackendDomainPoolSettings{}, err
	}
	settings.Version = current.Version + 1

	raw, err := json.Marshal(settings)
	if err != nil {
		return BackendDomainPoolSettings{}, err
	}

	if err := upsertSystemSettingTx(
		ctx,
		db,
		backendDomainPoolSettingKey,
		string(raw),
		updatedBy,
	); err != nil {
		return BackendDomainPoolSettings{}, err
	}

	invalidateBackendDomainPoolSettingsCache()

	return settings, nil
}

// NormalizeBackendDomainPoolSettings 规范化业务域名池配置。
func NormalizeBackendDomainPoolSettings(settings *BackendDomainPoolSettings) error {
	if settings == nil {
		return errors.New("backend domain pool settings is nil")
	}

	if settings.TTLSeconds <= 0 {
		settings.TTLSeconds = 3600
	}
	if settings.TTLSeconds < 60 || settings.TTLSeconds > 86400 {
		return errors.New("域名池 TTL 必须在 60 到 86400 秒之间")
	}
	if len(settings.APIOrigins) > 16 {
		return errors.New("业务域名池最多包含 16 个域名")
	}

	normalizedOrigins := make([]string, 0, len(settings.APIOrigins))
	seen := make(map[string]struct{}, len(settings.APIOrigins))
	for _, origin := range settings.APIOrigins {
		normalized, err := normalizeBackendOrigin(origin)
		if err != nil {
			return err
		}
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		normalizedOrigins = append(normalizedOrigins, normalized)
	}
	settings.APIOrigins = normalizedOrigins
	if settings.Version < 0 {
		settings.Version = 0
	}
	return nil
}

func normalizeBackendOrigin(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", errors.New("域名格式无效")
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "https" {
		return "", errors.New("业务域名必须以 https:// 开头")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" || len(host) > 253 || !strings.Contains(host, ".") || net.ParseIP(host) != nil || strings.Trim(host, "0123456789.") == "" || strings.HasSuffix(host, ".local") {
		return "", errors.New("请填写完整业务域名，不能使用 IP 或本地域名")
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" || (parsed.Port() != "" && parsed.Port() != "443") {
		return "", errors.New("只填写 HTTPS 域名，不包含账号、API 路径、查询参数或其他端口")
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("业务域名格式无效")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", errors.New("业务域名格式无效")
			}
		}
	}
	return "https://" + host, nil
}
