package controller_chat

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// DeviceItem 设备列表单项。
type DeviceItem struct {
	ID          string `json:"id"`
	DeviceID    string `json:"device_id"`
	DeviceName  string `json:"device_name"`
	Platform    string `json:"platform"`
	IP          string `json:"ip"`
	LastLoginAt string `json:"last_login_at"`
	LastUsedAt  string `json:"last_used_at"`
	IsCurrent   bool   `json:"is_current"`
	IsOnline    bool   `json:"is_online"`
	Status      string `json:"status"` // online | offline
}

// ListDevicesResult 设备列表响应。
type ListDevicesResult struct {
	Items []DeviceItem `json:"items"`
}

type deviceAgg struct {
	latest   model.UserRefreshSession
	isOnline bool
}

// deviceOnlineFreshness：超过该时长无活跃的有效会话，列表显示为离线。
const deviceOnlineFreshness = 7 * 24 * time.Hour

// ListDevices 优先读 user_devices（一用户一设备一行），再用 session 判定在线。
// 若设备表尚无数据，回退到会话聚合（兼容老数据）。
func ListDevices(imUserID, currentDeviceID string) (*ListDevicesResult, error) {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	currentDeviceID = normalizeDeviceID(currentDeviceID)
	deviceRepo := repository.NewUserDeviceRepository()
	devices, err := deviceRepo.ListByUserID(user.ID)
	if err != nil {
		return nil, err
	}

	sessionRepo := repository.NewUserRefreshSessionRepository()
	sessions, err := sessionRepo.ListForDeviceAggregation(user.ID, 90)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	onlineByDevice := map[string]bool{}
	lastUsedByDevice := map[string]time.Time{}
	for _, session := range sessions {
		deviceID := strings.TrimSpace(session.DeviceID)
		if deviceID == "" {
			continue
		}
		online := session.RevokedAt == nil &&
			session.ExpiresAt.After(now) &&
			!session.LastUsedAt.Before(now.Add(-deviceOnlineFreshness))
		if online {
			onlineByDevice[deviceID] = true
		}
		if prev, ok := lastUsedByDevice[deviceID]; !ok || session.LastUsedAt.After(prev) {
			lastUsedByDevice[deviceID] = session.LastUsedAt
		}
	}

	if len(devices) > 0 {
		items := make([]DeviceItem, 0, len(devices))
		for _, d := range devices {
			if isWebDevice(d.Platform, d.DeviceName) {
				continue
			}
			online := onlineByDevice[d.DeviceID]
			status := model.UserDeviceStatusOffline
			if online {
				status = model.UserDeviceStatusOnline
			}
			lastAt := d.LastLoginAt
			if used, ok := lastUsedByDevice[d.DeviceID]; ok && used.After(lastAt) {
				lastAt = used
			}
			lastText := lastAt.Format("2006-01-02 15:04:05")
			items = append(items, DeviceItem{
				ID:          fmt.Sprintf("dev-%d", d.ID),
				DeviceID:    d.DeviceID,
				DeviceName:  deviceDisplayName(d.DeviceName, d.Platform),
				Platform:    d.Platform,
				IP:          d.LastIP,
				LastLoginAt: lastText,
				LastUsedAt:  lastText,
				IsCurrent:   currentDeviceID != "" && d.DeviceID == currentDeviceID,
				IsOnline:    online,
				Status:      status,
			})
		}
		sortDeviceItems(items)
		return &ListDevicesResult{Items: items}, nil
	}

	// 兼容：尚无 user_devices 行时，仍按会话聚合，并顺带回填设备表。
	return listDevicesFromSessions(user.ID, currentDeviceID, sessions, now)
}

func listDevicesFromSessions(userID uint, currentDeviceID string, sessions []model.UserRefreshSession, now time.Time) (*ListDevicesResult, error) {
	byDevice := make(map[string]*deviceAgg)
	for _, session := range sessions {
		key := session.DeviceID
		if key == "" {
			ipKey := strings.TrimSpace(session.IP)
			if ipKey == "" {
				ipKey = "unknown"
			}
			key = fmt.Sprintf("legacy_ip:%s", ipKey)
		}

		online := session.RevokedAt == nil &&
			session.ExpiresAt.After(now) &&
			!session.LastUsedAt.Before(now.Add(-deviceOnlineFreshness))
		agg, ok := byDevice[key]
		if !ok {
			byDevice[key] = &deviceAgg{latest: session, isOnline: online}
			continue
		}
		if session.LastUsedAt.After(agg.latest.LastUsedAt) ||
			(session.LastUsedAt.Equal(agg.latest.LastUsedAt) && preferDeviceName(session.DeviceName, agg.latest.DeviceName)) {
			agg.latest = session
		}
		if online {
			agg.isOnline = true
		}
	}

	deviceRepo := repository.NewUserDeviceRepository()
	items := make([]DeviceItem, 0, len(byDevice))
	for aggregationKey, agg := range byDevice {
		s := agg.latest
		if isWebDevice(s.Platform, s.DeviceName) {
			continue
		}
		status := "offline"
		if agg.isOnline {
			status = "online"
		}
		lastUsed := s.LastUsedAt.Format("2006-01-02 15:04:05")
		deviceID := deviceItemID(s.DeviceID, aggregationKey)
		if s.DeviceID != "" {
			_ = deviceRepo.UpsertLogin(userID, s.DeviceID, s.DeviceName, s.Platform, s.AppVersion, s.IP, s.LastUsedAt)
			if !agg.isOnline {
				_ = deviceRepo.MarkOffline(userID, s.DeviceID, now)
			}
		}
		items = append(items, DeviceItem{
			ID:          fmt.Sprintf("%d", s.ID),
			DeviceID:    deviceID,
			DeviceName:  deviceDisplayName(s.DeviceName, s.Platform),
			Platform:    s.Platform,
			IP:          s.IP,
			LastLoginAt: lastUsed,
			LastUsedAt:  lastUsed,
			IsCurrent:   currentDeviceID != "" && s.DeviceID != "" && s.DeviceID == currentDeviceID,
			IsOnline:    agg.isOnline,
			Status:      status,
		})
	}
	sortDeviceItems(items)
	return &ListDevicesResult{Items: items}, nil
}

func sortDeviceItems(items []DeviceItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsCurrent != items[j].IsCurrent {
			return items[i].IsCurrent
		}
		if items[i].IsOnline != items[j].IsOnline {
			return items[i].IsOnline
		}
		return items[i].LastUsedAt > items[j].LastUsedAt
	})
}

func deviceDisplayName(name, platform string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "ios":
		return "iPhone / iPad"
	case "android":
		return "Android 设备"
	case "web", "h5", "browser":
		return "Web 浏览器"
	case "windows":
		return "Windows"
	case "macos", "mac":
		return "Mac"
	default:
		if platform != "" {
			return platform
		}
		return "未知设备"
	}
}

// isWebPlatform Web / H5 / 浏览器登录，不在 App 设备列表与安全提醒中展示。
func isWebPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "web", "h5", "browser", "pc", "desktop":
		return true
	default:
		return false
	}
}

// isWebDevice 按 platform + 设备名综合判断（兼容 Chrome Web / Safari H5 等命名）。
func isWebDevice(platform, deviceName string) bool {
	if isWebPlatform(platform) {
		return true
	}
	blob := strings.ToLower(strings.TrimSpace(platform) + " " + strings.TrimSpace(deviceName))
	for _, key := range []string{"web", "h5", "browser", "chrome", "safari", "firefox", "edge"} {
		if strings.Contains(blob, key) {
			return true
		}
	}
	return false
}

func preferDeviceName(candidate, current string) bool {
	candidate = strings.TrimSpace(candidate)
	current = strings.TrimSpace(current)
	if candidate == "" {
		return false
	}
	if current == "" {
		return true
	}
	generics := map[string]struct{}{
		"android 设备":  {},
		"iphone / ipad": {},
		"未知设备":      {},
		"弥络蓝序":      {},
	}
	_, currentGeneric := generics[strings.ToLower(current)]
	_, candidateGeneric := generics[strings.ToLower(candidate)]
	return currentGeneric && !candidateGeneric
}

// RevokeDevice 撤销指定设备的全部有效 refresh sessions，并更新设备表。
func RevokeDevice(imUserID, deviceID string) error {
	deviceID = normalizeDeviceID(deviceID)
	if deviceID == "" {
		return AuthErr("device_id 不能为空")
	}
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return err
	}
	now := time.Now()
	if strings.HasPrefix(deviceID, "legacy_ip:") {
		ip := strings.TrimPrefix(deviceID, "legacy_ip:")
		if ip == "" || ip == "unknown" {
			return AuthErr("无法识别该历史设备")
		}
		_, err = repository.NewUserRefreshSessionRepository().RevokeLegacyByUserIDAndIP(user.ID, ip)
		return err
	}
	if _, err = repository.NewUserRefreshSessionRepository().RevokeAllByUserIDAndDeviceID(user.ID, deviceID); err != nil {
		return err
	}
	_ = repository.NewUserDeviceRepository().MarkOffline(user.ID, deviceID, now)
	return nil
}

func deviceItemID(deviceID, aggregationKey string) string {
	if deviceID != "" {
		return deviceID
	}
	if strings.HasPrefix(aggregationKey, "legacy_ip:") {
		return aggregationKey
	}
	return ""
}
