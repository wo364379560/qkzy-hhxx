package tencentim

import (
	"fmt"
	"strconv"
	"strings"
)

// accountBaseResp 通用响应结构。
type accountBaseResp struct {
	ActionStatus string `json:"ActionStatus"`
	ErrorCode    int    `json:"ErrorCode"`
	ErrorInfo    string `json:"ErrorInfo"`
}

func (r *accountBaseResp) err() error {
	if r.ErrorCode != 0 {
		return fmt.Errorf("tencent IM [%d]: %s", r.ErrorCode, r.ErrorInfo)
	}
	return nil
}

// AccountImport 导入单个账号到腾讯云 IM。
func (c *Client) AccountImport(userID, nick, faceURL string) error {
	type req struct {
		Identifier string `json:"Identifier"`
		Nick       string `json:"Nick,omitempty"`
		FaceUrl    string `json:"FaceUrl,omitempty"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("im_open_login_svc", "account_import"), req{
		Identifier: userID,
		Nick:       nick,
		FaceUrl:    faceURL,
	}, &resp); err != nil {
		return err
	}
	return resp.err()
}

// AccountDelete 删除账号。
func (c *Client) AccountDelete(userID string) error {
	type deleteItem struct {
		UserID string `json:"UserID"`
	}
	type req struct {
		DeleteItem []deleteItem `json:"DeleteItem"`
	}
	type respBody struct {
		accountBaseResp
		FailAccounts []string `json:"FailAccounts"`
	}
	var resp respBody
	if err := c.post(c.buildURL("im_open_login_svc", "account_delete"), req{
		DeleteItem: []deleteItem{{UserID: userID}},
	}, &resp); err != nil {
		return err
	}
	if len(resp.FailAccounts) > 0 {
		return fmt.Errorf("tencent IM delete account failed: %v", resp.FailAccounts)
	}
	return resp.err()
}

// AccountBan 永久禁止发送 C2C 和群消息。
func (c *Client) AccountBan(userID string) error {
	return c.setNospeaking(userID, 4294967295) // max uint32 = 永久
}

// AccountUnban 解除禁言，恢复发消息权限。
func (c *Client) AccountUnban(userID string) error {
	return c.setNospeaking(userID, 0) // 0 = 解除禁言
}

// setNospeaking 封禁或解封账号（duration=0 表示解封，duration=4294967295 表示永久封禁）。
func (c *Client) setNospeaking(userID string, duration uint32) error {
	type req struct {
		Set_Account            string `json:"Set_Account"`
		C2CmsgNospeakingTime   uint32 `json:"C2CmsgNospeakingTime"`
		GroupmsgNospeakingTime uint32 `json:"GroupmsgNospeakingTime"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("openconfigsvr", "setnospeaking"), req{
		Set_Account:            userID,
		C2CmsgNospeakingTime:   duration,
		GroupmsgNospeakingTime: duration,
	}, &resp); err != nil {
		return err
	}
	return resp.err()
}

// AccountKick 强制下线。
func (c *Client) AccountKick(userID string) error {
	type req struct {
		Identifier string `json:"Identifier"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("im_open_login_svc", "kick"), req{
		Identifier: userID,
	}, &resp); err != nil {
		return err
	}
	return resp.err()
}

// UserInfo 用户信息。
type UserInfo struct {
	UserID        string
	Nickname      string
	FaceURL       string
	SelfSignature string
	Gender        int8
	Birthday      int
	AllowType     int8
}

// AccountGetInfo 批量获取用户信息。
func (c *Client) AccountGetInfo(userIDs []string) (map[string]UserInfo, error) {
	if len(userIDs) == 0 {
		return map[string]UserInfo{}, nil
	}
	type req struct {
		To_Account []string `json:"To_Account"`
		TagList    []string `json:"TagList"`
	}
	type userItem struct {
		To_Account  string `json:"To_Account"`
		ProfileItem []struct {
			Tag   string `json:"Tag"`
			Value any    `json:"Value"`
		} `json:"ProfileItem"`
		ResultCode int    `json:"ResultCode"`
		ResultInfo string `json:"ResultInfo"`
	}
	type respBody struct {
		accountBaseResp
		UserProfileItem []userItem `json:"UserProfileItem"`
	}

	result := make(map[string]UserInfo, len(userIDs))
	for start := 0; start < len(userIDs); start += 100 {
		end := start + 100
		if end > len(userIDs) {
			end = len(userIDs)
		}

		var resp respBody
		if err := c.post(c.buildURL("profile", "portrait_get"), req{
			To_Account: userIDs[start:end],
			TagList: []string{
				TagNick,
				TagImage,
				TagSelfSignature,
				TagGender,
				TagBirthDay,
				TagAllowType,
			},
		}, &resp); err != nil {
			return nil, err
		}
		if err := resp.err(); err != nil {
			return nil, err
		}

		for _, item := range resp.UserProfileItem {
			if item.ResultCode != 0 {
				continue
			}
			info := UserInfo{UserID: item.To_Account}
			for _, profile := range item.ProfileItem {
				switch profile.Tag {
				case TagNick:
					info.Nickname = profileString(profile.Value)
				case TagImage:
					info.FaceURL = profileString(profile.Value)
				case TagSelfSignature:
					info.SelfSignature = profileString(profile.Value)
				case TagGender:
					info.Gender = profileGender(profile.Value)
				case TagBirthDay:
					info.Birthday = profileBirthday(profile.Value)
				case TagAllowType:
					info.AllowType = profileAllowType(profile.Value)
				}
			}
			result[item.To_Account] = info
		}
	}

	return result, nil
}

func profileString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func profileInt(value any) (int, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func profileGender(value any) int8 {
	switch profileString(value) {
	case "Gender_Type_Male", "1":
		return 1
	case "Gender_Type_Female", "2":
		return 2
	default:
		return 0
	}
}

func profileBirthday(value any) int {
	n, ok := profileInt(value)
	if !ok || n <= 0 {
		return 0
	}
	return n
}

func profileAllowType(value any) int8 {
	switch profileString(value) {
	case "AllowType_Type_NeedConfirm", "1":
		return 1
	case "AllowType_Type_DenyAny", "2":
		return 2
	default:
		return 0
	}
}
