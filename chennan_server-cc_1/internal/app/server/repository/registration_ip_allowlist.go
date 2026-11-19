package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const registrationIPAllowlistKey = "registration_ip_allowlist"
const maxRegistrationIPAllowlist = 200

type RegistrationIPEntry struct {
	IP        string    `json:"ip"`
	Note      string    `json:"note"`
	UpdatedBy uint      `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NormalizeRegistrationIP permits a single globally routable address, never a CIDR,
// host name, port, zone, private, documentation, benchmark or shared-carrier range.
func NormalizeRegistrationIP(raw string) (string, error) {
	ip, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil || ip.Zone() != "" {
		return "", errors.New("仅支持精确公网 IPv4 或 IPv6 地址，不支持网段")
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return "", errors.New("仅支持精确公网 IP")
	}
	for _, cidr := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "3fff::/20", "2001::/32", "2001:2::/48", "2001:10::/28", "2001:20::/28", "2002::/16"} {
		if netip.MustParsePrefix(cidr).Contains(ip) {
			return "", errors.New("仅支持精确公网 IP")
		}
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return "", errors.New("仅支持精确公网 IP")
	}
	return ip.String(), nil
}

func decodeRegistrationIPs(value string) ([]RegistrationIPEntry, error) {
	entries := []RegistrationIPEntry{}
	if err := json.Unmarshal([]byte(value), &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []RegistrationIPEntry{}
	}
	return entries, nil
}

func ListRegistrationIPs(ctx context.Context, db *gorm.DB) ([]RegistrationIPEntry, error) {
	var setting model.SystemSetting
	err := db.WithContext(ctx).Where("setting_key = ?", registrationIPAllowlistKey).Take(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []RegistrationIPEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeRegistrationIPs(setting.Value)
}

// CheckRegistrationIP uses no process cache: a completed add/delete is visible to
// the next request on every process. Lookup failure never grants an exception.
func CheckRegistrationIP(ctx context.Context, raw string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	return registrationIPAllowed(ctx, datasource.Datasource.Mysql.GetClient(), raw)
}

func registrationIPAllowed(ctx context.Context, db *gorm.DB, raw string) (bool, error) {
	ip, err := NormalizeRegistrationIP(raw)
	if err != nil {
		return false, nil
	}
	entries, err := ListRegistrationIPs(ctx, db)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IP == ip {
			return true, nil
		}
	}
	return false, nil
}

// ChangeRegistrationIP serializes concurrent edits on the settings row. Existing
// deployments already have system_settings, so this feature requires no migration.
func ChangeRegistrationIP(ctx context.Context, db *gorm.DB, actor uint, raw, note string, remove bool, audit model.AdminAccessAudit) error {
	ip, err := NormalizeRegistrationIP(raw)
	if err != nil {
		return err
	}
	note = strings.TrimSpace(note)
	if len([]rune(note)) > 200 {
		return errors.New("备注不能超过 200 字")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		initial := model.SystemSetting{SettingKey: registrationIPAllowlistKey, Value: "[]", UpdatedBy: &actor}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&initial).Error; err != nil {
			return err
		}
		var setting model.SystemSetting
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("setting_key = ?", registrationIPAllowlistKey).Take(&setting).Error; err != nil {
			return err
		}
		entries, err := decodeRegistrationIPs(setting.Value)
		if err != nil {
			return err
		}
		next := make([]RegistrationIPEntry, 0, len(entries)+1)
		for _, entry := range entries {
			if entry.IP != ip {
				next = append(next, entry)
			}
		}
		if !remove {
			if len(next) >= maxRegistrationIPAllowlist {
				return errors.New("注册 IP 放行最多 200 条")
			}
			next = append(next, RegistrationIPEntry{IP: ip, Note: note, UpdatedBy: actor, UpdatedAt: time.Now()})
		}
		value, err := json.Marshal(next)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.SystemSetting{}).Where("setting_key = ?", registrationIPAllowlistKey).Updates(map[string]any{"value": string(value), "updated_by": actor}).Error; err != nil {
			return err
		}
		audit.ID = 0
		audit.AdminUserID = &actor
		audit.AccessType = "system_user"
		audit.Method, audit.Route = "POST", "/api/v1/admin/security/registration-ips"
		audit.ActionKey, audit.ActionSummary = "registration_ip.add", "添加注册 IP 放行"
		if remove {
			audit.ActionKey, audit.ActionSummary = "registration_ip.remove", "删除注册 IP 放行"
		}
		audit.TargetType, audit.TargetUsername, audit.TargetDisplayName = "registration_ip", ip, ip
		change, err := json.Marshal(map[string]any{"ip": ip, "removed": remove, "note": note})
		if err != nil {
			return err
		}
		audit.ChangeSetJSON, audit.Success, audit.StatusCode = string(change), true, 200
		// An audit storage failure rolls back the exemption change as well.
		return tx.Create(&audit).Error
	})
}
