package tencentim

import (
	"context"
	"errors"
	"strings"
	"time"
)

// GroupCreate 在腾讯云 IM 创建群组，groupID 非空时使用自定义 ID（格式 @TGS#_SUFFIX），返回最终 GroupId。
func (c *Client) GroupCreate(groupID, name, ownerID, groupType string) (string, error) {
	type req struct {
		Owner_Account string `json:"Owner_Account,omitempty"`
		Type          string `json:"Type"`
		GroupId       string `json:"GroupId,omitempty"`
		Name          string `json:"Name"`
	}
	type respBody struct {
		accountBaseResp
		GroupId string `json:"GroupId"`
	}
	var resp respBody
	if groupType == "" {
		groupType = "Work"
	}
	r := req{Type: groupType, Name: name, GroupId: groupID}
	if ownerID != "" {
		r.Owner_Account = ownerID
	}
	if err := c.post(c.buildURL("group_open_http_svc", "create_group"), r, &resp); err != nil {
		return "", err
	}
	if err := resp.err(); err != nil {
		return "", err
	}
	if resp.GroupId != "" {
		return resp.GroupId, nil
	}
	return groupID, nil // TIM 有时不回传 GroupId，用传入值
}

// GroupDestroy 解散群组。
func (c *Client) GroupDestroy(groupID string) error {
	type req struct {
		GroupId string `json:"GroupId"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("group_open_http_svc", "destroy_group"), req{GroupId: groupID}, &resp); err != nil {
		return err
	}
	return resp.err()
}

type GroupAddMemberItemResult struct {
	Account string `json:"account"`
	Result  int    `json:"result"`
	Reason  string `json:"reason,omitempty"`
}

type GroupAddMemberResult struct {
	Added          []string                   `json:"added"`
	AlreadyMembers []string                   `json:"already_members"`
	Failed         []GroupAddMemberItemResult `json:"failed"`
}

// GroupAddMember 批量添加群成员，Silence=0 下发进群系统通知（成员会话列表中出现群）。
func (c *Client) GroupAddMember(groupID string, memberIDs []string) error {
	_, err := c.GroupAddMemberDetailed(groupID, memberIDs, false)
	return err
}

// GroupAddMemberWithSilence 批量添加群成员，并允许业务层关闭腾讯默认的管理员入群提示。
func (c *Client) GroupAddMemberWithSilence(groupID string, memberIDs []string, silence bool) error {
	_, err := c.GroupAddMemberDetailed(groupID, memberIDs, silence)
	return err
}

// GroupAddMemberDetailed 批量添加群成员，并返回腾讯对每个成员的处理结果。
func (c *Client) GroupAddMemberDetailed(groupID string, memberIDs []string, silence bool) (GroupAddMemberResult, error) {
	type memberItem struct {
		Member_Account string `json:"Member_Account"`
	}
	type req struct {
		GroupId    string       `json:"GroupId"`
		MemberList []memberItem `json:"MemberList"`
		Silence    int          `json:"Silence"`
	}
	members := make([]memberItem, 0, len(memberIDs))
	for _, id := range memberIDs {
		members = append(members, memberItem{Member_Account: id})
	}
	type respBody struct {
		accountBaseResp
		MemberList []struct {
			Member_Account string `json:"Member_Account"`
			Result         int    `json:"Result"` // 0=成功 1=已是成员 2=人数超限
		} `json:"MemberList"`
	}
	var resp respBody
	silenceValue := 0
	if silence {
		silenceValue = 1
	}
	if err := c.post(c.buildURL("group_open_http_svc", "add_group_member"), req{
		GroupId:    groupID,
		MemberList: members,
		Silence:    silenceValue,
	}, &resp); err != nil {
		return GroupAddMemberResult{}, err
	}
	if err := resp.err(); err != nil {
		return GroupAddMemberResult{}, err
	}
	result := GroupAddMemberResult{
		Added:          make([]string, 0),
		AlreadyMembers: make([]string, 0),
		Failed:         make([]GroupAddMemberItemResult, 0),
	}
	for _, item := range resp.MemberList {
		switch item.Result {
		case 0:
			result.Added = append(result.Added, item.Member_Account)
		case 1:
			result.AlreadyMembers = append(result.AlreadyMembers, item.Member_Account)
		default:
			result.Failed = append(result.Failed, GroupAddMemberItemResult{
				Account: item.Member_Account,
				Result:  item.Result,
				Reason:  groupAddMemberResultReason(item.Result),
			})
		}
	}
	return result, nil
}

func groupAddMemberResultReason(result int) string {
	switch result {
	case 2:
		return "群成员人数已达上限"
	default:
		return "腾讯IM添加成员失败"
	}
}

// GroupDeleteMember 批量移除群成员，Silence=0 下发退群系统通知。
func (c *Client) GroupDeleteMember(groupID string, memberIDs []string) error {
	type req struct {
		GroupId             string   `json:"GroupId"`
		MemberToDel_Account []string `json:"MemberToDel_Account"`
		Silence             int      `json:"Silence"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("group_open_http_svc", "delete_group_member"), req{
		GroupId:             groupID,
		MemberToDel_Account: memberIDs,
		Silence:             0, // 下发退群系统通知
	}, &resp); err != nil {
		return err
	}
	return resp.err()
}

// GroupInfo 群组基本信息（用于批量查询）。
type GroupInfo struct {
	GroupID         string
	Name            string
	Type            string
	MemberNum       int
	OwnerAccount    string // TIM 群主 IM 账号
	ShutUpAllMember bool   // 是否全员禁言
	CreateTime      int64  // TIM 真实创建时间，Unix 秒
}

// GroupListItem 群组列表项。
type GroupListItem struct {
	GroupID   string
	GroupName string
}

// GroupGetInfo 批量获取群组信息（一次 API 最多 50 个）。
func (c *Client) GroupGetInfo(groupIDs []string) (map[string]GroupInfo, error) {
	return c.GroupGetInfoContext(context.Background(), groupIDs)
}

func (c *Client) GroupGetInfoContext(ctx context.Context, groupIDs []string) (map[string]GroupInfo, error) {
	type req struct {
		GroupIdList         []string `json:"GroupIdList"`
		GroupBaseInfoFilter []string `json:"GroupBaseInfoFilter"`
	}
	type groupItem struct {
		GroupId         string `json:"GroupId"`
		ErrorCode       int    `json:"ErrorCode"`
		Name            string `json:"Name"`
		Type            string `json:"Type"`
		MemberNum       int    `json:"MemberNum"`
		Owner_Account   string `json:"Owner_Account"`
		ShutUpAllMember string `json:"ShutUpAllMember"` // "On" / "Off"
		CreateTime      int64  `json:"CreateTime"`
	}
	type respBody struct {
		accountBaseResp
		GroupInfoList []groupItem `json:"GroupInfo"`
	}
	var resp respBody
	if err := c.postContext(ctx, "group_open_http_svc", "get_group_info", req{
		GroupIdList:         groupIDs,
		GroupBaseInfoFilter: []string{"Name", "Type", "MemberNum", "Owner_Account", "ShutUpAllMember", "CreateTime"},
	}, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	result := make(map[string]GroupInfo, len(resp.GroupInfoList))
	for _, item := range resp.GroupInfoList {
		if item.ErrorCode != 0 {
			continue
		}
		result[item.GroupId] = GroupInfo{
			GroupID:         item.GroupId,
			Name:            item.Name,
			Type:            item.Type,
			MemberNum:       item.MemberNum,
			OwnerAccount:    item.Owner_Account,
			ShutUpAllMember: item.ShutUpAllMember == "On",
			CreateTime:      item.CreateTime,
		}
	}
	return result, nil
}

// GroupMuteAll 开启或关闭全员禁言（ShutUpAllMember On/Off）。
func (c *Client) GroupMuteAll(groupID string, mute bool) error {
	type req struct {
		GroupId         string `json:"GroupId"`
		ShutUpAllMember string `json:"ShutUpAllMember"`
	}
	var resp accountBaseResp
	flag := "Off"
	if mute {
		flag = "On"
	}
	if err := c.post(c.buildURL("group_open_http_svc", "modify_group_base_info"), req{
		GroupId:         groupID,
		ShutUpAllMember: flag,
	}, &resp); err != nil {
		return err
	}
	return resp.err()
}

// GroupMember 群成员信息。
type GroupMember struct {
	Account   string
	Role      string // Owner / Admin / Member
	JoinTime  int64
	MuteUntil int64
}

// GroupGetRoles 批量查询指定用户在群内的角色。
func (c *Client) GroupGetRoles(groupID string, userIDs []string) (map[string]string, error) {
	type req struct {
		GroupId     string   `json:"GroupId"`
		UserAccount []string `json:"User_Account"`
	}
	type roleItem struct {
		Member_Account string `json:"Member_Account"`
		Role           string `json:"Role"`
	}
	type respBody struct {
		accountBaseResp
		UserIDList []roleItem `json:"UserIdList"`
	}
	var resp respBody
	if err := c.post(c.buildURL("group_open_http_svc", "get_role_in_group"), req{
		GroupId:     groupID,
		UserAccount: userIDs,
	}, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	roles := make(map[string]string, len(resp.UserIDList))
	for _, item := range resp.UserIDList {
		if item.Member_Account == "" {
			continue
		}
		roles[item.Member_Account] = item.Role
	}
	return roles, nil
}

// GroupGetRole 查询单个用户在群内的角色。
func (c *Client) GroupGetRole(groupID, userID string) (string, error) {
	roles, err := c.GroupGetRoles(groupID, []string{userID})
	if err != nil {
		return "", err
	}
	return roles[userID], nil
}

// GroupGetMembers 获取群成员列表（最多 200 人，单次调用）。
func (c *Client) GroupGetMembers(groupID string) ([]GroupMember, error) {
	type req struct {
		GroupId          string   `json:"GroupId"`
		MemberInfoFilter []string `json:"MemberInfoFilter"`
		Limit            int      `json:"Limit"`
		Offset           int      `json:"Offset"`
	}
	type memberItem struct {
		Member_Account string `json:"Member_Account"`
		Role           string `json:"Role"`
		JoinTime       int64  `json:"JoinTime"`
		MuteUntil      int64  `json:"MuteUntil"`
	}
	type respBody struct {
		accountBaseResp
		MemberList []memberItem `json:"MemberList"`
		MemberNum  int          `json:"MemberNum"`
	}
	var resp respBody
	if err := c.post(c.buildURL("group_open_http_svc", "get_group_member_info"), req{
		GroupId:          groupID,
		MemberInfoFilter: []string{"Account", "Role", "JoinTime", "MuteUntil"},
		Limit:            200,
		Offset:           0,
	}, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	members := make([]GroupMember, 0, len(resp.MemberList))
	for _, m := range resp.MemberList {
		members = append(members, GroupMember{
			Account:   m.Member_Account,
			Role:      m.Role,
			JoinTime:  m.JoinTime,
			MuteUntil: m.MuteUntil,
		})
	}
	return members, nil
}

// GroupGetAllMembers always obtains a fresh complete snapshot; write-side
// authorization must never consume the display request coalescer's state.
func (c *Client) GroupGetAllMembers(groupID string) ([]GroupMember, error) {
	return c.GroupGetAllMembersContext(context.Background(), groupID)
}

func (c *Client) GroupGetAllMembersContext(ctx context.Context, groupID string) ([]GroupMember, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	info, err := c.GroupGetInfoContext(ctx, []string{groupID})
	if err != nil {
		return nil, err
	}
	group, ok := info[groupID]
	if !ok {
		return nil, errors.New("missing tencent group metadata")
	}
	switch group.Type {
	case "Community", "Public", "Private", "Work", "ChatRoom", "Meeting":
	default:
		return nil, errors.New("unsupported tencent group member pagination")
	}
	return c.groupGetAllMembersForType(ctx, groupID, group.Type)
}

func (c *Client) groupGetAllMembersForType(ctx context.Context, groupID, groupType string) ([]GroupMember, error) {
	type req struct {
		GroupId          string   `json:"GroupId"`
		MemberInfoFilter []string `json:"MemberInfoFilter"`
		Limit            int      `json:"Limit"`
		Offset           *int     `json:"Offset,omitempty"`
		Next             *string  `json:"Next,omitempty"`
	}
	type memberItem struct {
		Account   string `json:"Member_Account"`
		Role      string `json:"Role"`
		JoinTime  int64  `json:"JoinTime"`
		MuteUntil int64  `json:"MuteUntil"`
	}
	type respBody struct {
		accountBaseResp
		MemberList []memberItem `json:"MemberList"`
		MemberNum  *int         `json:"MemberNum"`
		Next       *string      `json:"Next"`
	}
	const limit, maxPages = 100, 1000
	community := groupType == "Community"
	members := make([]GroupMember, 0)
	seenMembers := make(map[string]struct{})
	seenNext := make(map[string]struct{})
	next, offset, expected := "", 0, -1
	for page := 0; page < maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var resp respBody
		request := req{GroupId: groupID, MemberInfoFilter: []string{"Role", "JoinTime", "MuteUntil"}, Limit: limit}
		if community {
			request.Next = &next
		} else {
			request.Offset = &offset
		}
		if err := c.postContext(ctx, "group_open_http_svc", "get_group_member_info", request, &resp); err != nil {
			return nil, err
		}
		if err := resp.err(); err != nil {
			return nil, err
		}
		if resp.MemberList == nil {
			return nil, errors.New("missing tencent group member snapshot")
		}
		if resp.MemberNum != nil {
			if *resp.MemberNum < 0 || (expected >= 0 && expected != *resp.MemberNum) {
				return nil, errors.New("changed tencent group member snapshot")
			}
			expected = *resp.MemberNum
		}
		for _, m := range resp.MemberList {
			m.Account, m.Role = strings.TrimSpace(m.Account), strings.TrimSpace(m.Role)
			if m.Account == "" || (m.Role != "Owner" && m.Role != "Admin" && m.Role != "Member") {
				return nil, errors.New("invalid tencent group member snapshot")
			}
			if _, duplicate := seenMembers[m.Account]; duplicate {
				return nil, errors.New("duplicate tencent group member snapshot")
			}
			seenMembers[m.Account] = struct{}{}
			members = append(members, GroupMember{Account: m.Account, Role: m.Role, JoinTime: m.JoinTime, MuteUntil: m.MuteUntil})
		}
		if expected >= 0 && len(members) > expected {
			return nil, errors.New("changed tencent group member snapshot")
		}
		if community {
			// Tencent requires Next even on the first request. Empty Next ends the
			// scan regardless of page length; it must NEVER switch to Offset.
			if resp.Next == nil {
				return nil, errors.New("missing tencent group member cursor")
			}
			if *resp.Next == "" {
				if expected >= 0 && len(members) != expected {
					return nil, errors.New("incomplete tencent group member snapshot")
				}
				return members, nil
			}
			if len(resp.MemberList) == 0 {
				return nil, errors.New("incomplete tencent group member snapshot")
			}
			if _, repeated := seenNext[*resp.Next]; repeated {
				return nil, errors.New("repeated tencent group member cursor")
			}
			seenNext[*resp.Next] = struct{}{}
			next = *resp.Next
		} else {
			if resp.Next != nil && *resp.Next != "" {
				return nil, errors.New("unexpected tencent group pagination mode")
			}
			if (expected >= 0 && len(members) == expected) || len(resp.MemberList) < limit {
				if expected >= 0 && len(members) != expected {
					return nil, errors.New("incomplete tencent group member snapshot")
				}
				return members, nil
			}
			offset += len(resp.MemberList)
		}
	}
	return nil, errors.New("tencent group member page limit exceeded")
}

// GroupSetMemberRole 修改群成员角色，role 可为 Admin 或 Member。
func (c *Client) GroupSetMemberRole(groupID, userID, role string) error {
	type req struct {
		GroupId       string `json:"GroupId"`
		MemberAccount string `json:"Member_Account"`
		Role          string `json:"Role"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("group_open_http_svc", "modify_group_member_info"), req{
		GroupId:       groupID,
		MemberAccount: userID,
		Role:          role,
	}, &resp); err != nil {
		return err
	}
	if err := resp.err(); err != nil {
		return err
	}
	confirmed, err := c.GroupGetRole(groupID, userID)
	if err != nil {
		return errors.New("腾讯群成员角色修改结果暂未确认，请刷新后重试")
	}
	if confirmed != role {
		return errors.New("腾讯群成员角色修改未生效，请刷新后重试")
	}
	return nil
}

// GroupSetMemberMuteTime 设置群成员禁言时长，muteTime 为 0 表示解除禁言。
func (c *Client) GroupSetMemberMuteTime(groupID, userID string, muteTime int64) error {
	// MuteTime belongs to forbid_send_msg. modify_group_member_info may accept
	// the request while ignoring that unknown field, leaving admins unmuted.
	type req struct {
		GroupId        string   `json:"GroupId"`
		MembersAccount []string `json:"Members_Account"`
		MuteTime       int64    `json:"MuteTime"`
	}
	var resp accountBaseResp
	if err := c.post(c.buildURL("group_open_http_svc", "forbid_send_msg"), req{
		GroupId:        groupID,
		MembersAccount: []string{userID},
		MuteTime:       muteTime,
	}, &resp); err != nil {
		return err
	}
	if err := resp.err(); err != nil {
		return err
	}
	var checked struct {
		accountBaseResp
		MemberList []struct {
			Account   string `json:"Member_Account"`
			MuteUntil *int64
		}
	}
	if err := c.post(c.buildURL("group_open_http_svc", "get_specified_group_member_info"), map[string]any{"GroupId": groupID, "Member_List_Account": []string{userID}, "MemberInfoFilter": []string{"MuteUntil"}}, &checked); err != nil {
		return err
	}
	if err := checked.err(); err != nil {
		return err
	}
	if len(checked.MemberList) != 1 || checked.MemberList[0].Account != userID || checked.MemberList[0].MuteUntil == nil {
		return errors.New("禁言已提交，但无法确认生效；请刷新成员状态")
	}
	until := *checked.MemberList[0].MuteUntil
	if muteTime > 0 && until <= time.Now().Unix() || muteTime == 0 && until > time.Now().Unix() {
		return errors.New("腾讯返回的禁言状态未生效，请重试")
	}
	return nil
}

// GroupGetAllGroups 获取应用下所有群组列表。
func (c *Client) GroupGetAllGroups() ([]GroupListItem, error) {
	type req struct {
		Limit int    `json:"Limit"`
		Next  string `json:"Next,omitempty"`
	}
	type respBody struct {
		accountBaseResp
		TotalCount  int `json:"TotalCount"`
		GroupIdList []struct {
			GroupId string `json:"GroupId"`
		} `json:"GroupIdList"`
		Next string `json:"Next"`
	}

	allGroups := make([]GroupListItem, 0)
	next := ""

	for {
		var resp respBody
		if err := c.post(c.buildURL("group_open_http_svc", "get_appid_group_list"), req{
			Limit: 100,
			Next:  next,
		}, &resp); err != nil {
			return nil, err
		}
		if err := resp.err(); err != nil {
			return nil, err
		}

		for _, item := range resp.GroupIdList {
			allGroups = append(allGroups, GroupListItem{
				GroupID:   item.GroupId,
				GroupName: "", // 需要后续调用 GroupGetInfo 获取
			})
		}

		if resp.Next == "" {
			break
		}
		next = resp.Next
	}

	return allGroups, nil
}
