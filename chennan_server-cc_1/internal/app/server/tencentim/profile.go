package tencentim

// IM 资料标签常量
const (
	TagNick          = "Tag_Profile_IM_Nick"
	TagImage         = "Tag_Profile_IM_Image"
	TagGender        = "Tag_Profile_IM_Gender"
	TagBirthDay      = "Tag_Profile_IM_BirthDay"
	TagSelfSignature = "Tag_Profile_IM_SelfSignature"
	TagAllowType     = "Tag_Profile_IM_AllowType"
	TagUserTag       = "Tag_Profile_Custom_UserTag"
)

// TagCustom 构造 IM 用户自定义资料标签。
func TagCustom(key string) string {
	return "Tag_Profile_Custom_" + key
}

// ProfileItemDetail 资料字段。
type ProfileItemDetail struct {
	Tag   string `json:"Tag"`
	Value any    `json:"Value"`
}

// PortraitSetReq 设置用户资料请求。
type PortraitSetReq struct {
	From_Account string              `json:"From_Account"`
	ProfileItem  []ProfileItemDetail `json:"ProfileItem"`
}

type portraitSetResp struct {
	accountBaseResp
}

// PortraitSet 通过 IM REST API 更新用户资料字段。
// items 只传需要变更的字段即可（增量更新）。
func (c *Client) PortraitSet(userID string, items []ProfileItemDetail) error {
	if len(items) == 0 {
		return nil
	}
	req := PortraitSetReq{
		From_Account: userID,
		ProfileItem:  items,
	}
	var resp portraitSetResp
	url := c.buildURL("profile", "portrait_set")
	if err := c.post(url, req, &resp); err != nil {
		return err
	}
	return resp.err()
}

// AllowTypeTag 将本地加好友策略映射为腾讯云 IM 资料值。
// 0=AllowAny, 1=NeedConfirm, 2=DenyAny。
func AllowTypeTag(allowType int8) string {
	switch allowType {
	case 1:
		return "AllowType_Type_NeedConfirm"
	case 2:
		return "AllowType_Type_DenyAny"
	default:
		return "AllowType_Type_AllowAny"
	}
}

// SetAllowType 设置用户加好友验证方式（建号后必须调用，IM 默认是 AllowAny）。
func (c *Client) SetAllowType(userID string, allowType int8) error {
	return c.PortraitSet(userID, []ProfileItemDetail{
		{Tag: TagAllowType, Value: AllowTypeTag(allowType)},
	})
}
