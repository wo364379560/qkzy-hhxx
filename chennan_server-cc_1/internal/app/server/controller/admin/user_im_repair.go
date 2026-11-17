package controller_admin

import (
	"errors"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"strings"
	"sync"
)

var userIMRepairLocks sync.Map

func RepairUserIMAccount(actorID, userID uint, expectedSDKAppID int) (map[string]any, error) {
	user, err := findManagedUser(actorID, false, userID)
	if err != nil {
		return nil, err
	}
	if user.Status != "active" || strings.TrimSpace(user.IMUserID) == "" {
		return nil, errors.New("用户已停用或缺少IM账号，不可同步")
	}
	application := config.Cfg.Get().TencentIM.SDKAppID
	if expectedSDKAppID <= 0 || expectedSDKAppID != application {
		return nil, errors.New("当前聊天应用配置已变化，请重新核对SDKAppID")
	}
	lock, _ := userIMRepairLocks.LoadOrStore(user.ID, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	if !mutex.TryLock() {
		return nil, errors.New("该用户正在同步，请稍后查询")
	}
	defer mutex.Unlock()
	client := tencentim.NewClient()
	if !client.IsConfigured() {
		return nil, errors.New("腾讯IM尚未配置")
	}
	if err = client.AccountImport(user.IMUserID, user.Nickname, user.AvatarURL); err != nil {
		return nil, err
	}
	verified, err := client.AccountGetInfo([]string{user.IMUserID})
	if err != nil {
		return nil, err
	}
	if _, ok := verified[user.IMUserID]; !ok {
		return nil, errors.New("同步已提交，但腾讯账号状态未确认，请查询后再操作")
	}
	return map[string]any{"im_user_id": user.IMUserID, "sdk_app_id": application, "confirmed": true}, nil
}
