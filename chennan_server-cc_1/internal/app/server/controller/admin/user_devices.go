package controller_admin

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// UserLoginDeviceItem 后台用户历史登录设备。
type UserLoginDeviceItem struct {
	ID          string `json:"id"`
	DeviceID    string `json:"device_id"`
	DeviceName  string `json:"device_name"`
	Platform    string `json:"platform"`
	IsWeb       bool   `json:"is_web"`
	IP          string `json:"ip"`
	LastLoginAt string `json:"last_login_at"`
	IsOnline    bool   `json:"is_online"`
	Status      string `json:"status"`
}

// ListUserLoginDevices 返回指定用户的全部登录设备（含 Web），供后台编辑页展示。
func ListUserLoginDevices(actorID uint, userID uint) ([]UserLoginDeviceItem, error) {
	actor, err := BuildActorContext(actorID)
	if err != nil {
		return nil, err
	}
	user, err := findManagedUser(actorID, HasManagerPrivileges(actor.Role), userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

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
			!session.LastUsedAt.Before(now.Add(-7*24*time.Hour))
		if online {
			onlineByDevice[deviceID] = true
		}
		if prev, ok := lastUsedByDevice[deviceID]; !ok || session.LastUsedAt.After(prev) {
			lastUsedByDevice[deviceID] = session.LastUsedAt
		}
	}

	items := make([]UserLoginDeviceItem, 0, len(devices))
	if len(devices) > 0 {
		for _, d := range devices {
			online := onlineByDevice[d.DeviceID]
			status := model.UserDeviceStatusOffline
			if online {
				status = model.UserDeviceStatusOnline
			}
			lastAt := d.LastLoginAt
			if used, ok := lastUsedByDevice[d.DeviceID]; ok && used.After(lastAt) {
				lastAt = used
			}
			items = append(items, UserLoginDeviceItem{
				ID:          fmt.Sprintf("dev-%d", d.ID),
				DeviceID:    d.DeviceID,
				DeviceName:  adminDeviceDisplayName(d.DeviceName, d.Platform),
				Platform:    d.Platform,
				IsWeb:       isAdminWebPlatform(d.Platform),
				IP:          d.LastIP,
				LastLoginAt: lastAt.Format("2006-01-02 15:04:05"),
				IsOnline:    online,
				Status:      status,
			})
		}
	} else {
		seen := map[string]struct{}{}
		for _, session := range sessions {
			key := strings.TrimSpace(session.DeviceID)
			if key == "" {
				ipKey := strings.TrimSpace(session.IP)
				if ipKey == "" {
					ipKey = "unknown"
				}
				key = "legacy_ip:" + ipKey
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			online := session.RevokedAt == nil &&
				session.ExpiresAt.After(now) &&
				!session.LastUsedAt.Before(now.Add(-7*24*time.Hour))
			status := model.UserDeviceStatusOffline
			if online {
				status = model.UserDeviceStatusOnline
			}
			items = append(items, UserLoginDeviceItem{
				ID:          fmt.Sprintf("session-%d", session.ID),
				DeviceID:    key,
				DeviceName:  adminDeviceDisplayName(session.DeviceName, session.Platform),
				Platform:    session.Platform,
				IsWeb:       isAdminWebPlatform(session.Platform),
				IP:          session.IP,
				LastLoginAt: session.LastUsedAt.Format("2006-01-02 15:04:05"),
				IsOnline:    online,
				Status:      status,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsWeb != items[j].IsWeb {
			return items[i].IsWeb // Web 排前，方便后台查看
		}
		if items[i].IsOnline != items[j].IsOnline {
			return items[i].IsOnline
		}
		return items[i].LastLoginAt > items[j].LastLoginAt
	})
	return items, nil
}

func isAdminWebPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "web", "h5", "browser", "pc", "desktop":
		return true
	default:
		return false
	}
}

func adminDeviceDisplayName(name, platform string) string {
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
