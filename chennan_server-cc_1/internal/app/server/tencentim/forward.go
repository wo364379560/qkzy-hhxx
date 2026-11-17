package tencentim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeliveryError distinguishes explicit rejection from an unknown delivery.
type DeliveryError struct {
	Code      int
	Unknown   bool
	Message   string
	Command   string // REST query/send command, used for independent queue cooldowns.
	ErrorInfo string `json:"-"` // Sanitized Tencent detail; Message is safe for progress UI.
	RequestID string `json:"-"` // Internal correlation only; never part of user-facing errors.
	cause     error
}

func (e *DeliveryError) Error() string { return e.Message }
func (e *DeliveryError) Unwrap() error { return e.cause }

func (c *Client) forwardPost(ctx context.Context, service, command string, body, result any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return c.forwardError(command, 0, true, "请求内容编码失败", "", "")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(service, command), bytes.NewReader(b))
	if err != nil {
		return c.forwardError(command, 0, true, "请求创建失败", "", "")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	// Do not return URLs: they contain the administrator signature.
	if err != nil {
		return c.forwardError(command, 0, true, "网络请求失败，结果待确认", "", "")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.forwardError(command, 0, true, fmt.Sprintf("HTTP %d，结果待确认", resp.StatusCode), "", "")
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result); err != nil {
		return c.forwardError(command, 0, true, "响应不完整，结果待确认", "", "")
	}
	return nil
}

func (c *Client) forwardResponse(command, status string, code int, info, requestID string) error {
	if status == "OK" && code == 0 {
		return nil
	}
	if code == 0 {
		return c.forwardError(command, 0, true, "返回状态异常，结果待确认", info, requestID)
	}
	return c.forwardError(command, code, false, "腾讯拒绝请求", info, requestID)
}

// Each permission REST command gets its own shared rate gate. Other Tencent
// clients are unaffected; only forwarding workers supply this hook.
type ForwardQueryGate func(context.Context, string) error

func (c *Client) forwardQuery(ctx context.Context, command string, body, result any, gates []ForwardQueryGate) error {
	if len(gates) > 0 && gates[0] != nil {
		if err := gates[0](ctx, command); err != nil {
			return c.forwardGateError(command, err)
		}
	}
	return c.forwardPost(ctx, "group_open_http_svc", command, body, result)
}

// REST group sends can impersonate non-members and bypass mute. Check the
// authenticated sender against current Tencent state immediately before sending.
func (c *Client) CheckForwardGroup(ctx context.Context, group, sender string, gates ...ForwardQueryGate) (name string, err error) {
	command := "get_group_info"
	var info struct {
		ActionStatus string
		ErrorCode    int
		ErrorInfo    string
		RequestID    string `json:"RequestId"`
		GroupInfo    []struct {
			GroupID                              string `json:"GroupId"`
			ErrorCode                            int
			ErrorInfo                            string
			Name, ShutUpAllMember, MuteAllMember string
		}
	}
	if err := c.forwardQuery(ctx, "get_group_info", map[string]any{
		"GroupIdList": []string{group}, "ResponseFilter": map[string]any{
			"GroupBaseInfoFilter": []string{"Name", "MuteAllMember"},
			"MemberInfoFilter":    []string{},
		},
	}, &info, gates); err != nil {
		return "", err
	}
	if err := c.forwardResponse(command, info.ActionStatus, info.ErrorCode, info.ErrorInfo, info.RequestID); err != nil {
		return "", err
	}
	if len(info.GroupInfo) != 1 || info.GroupInfo[0].GroupID != group {
		return "", c.forwardError(command, 10010, false, "群资料响应与请求不匹配", "", info.RequestID)
	}
	if detail := info.GroupInfo[0]; detail.ErrorCode != 0 {
		return "", c.forwardError(command, detail.ErrorCode, false, "腾讯拒绝查询群资料", detail.ErrorInfo, info.RequestID)
	}
	// get_group_info returns MuteAllMember; older responses may use the setter's
	// ShutUpAllMember spelling. Missing/unknown state must never bypass mute.
	all := info.GroupInfo[0].MuteAllMember
	if all == "" {
		all = info.GroupInfo[0].ShutUpAllMember
	}
	if all != "On" && all != "Off" {
		return "", c.forwardError(command, 0, true, "无法确认群禁言状态", "", info.RequestID)
	}
	command = "get_specified_group_member_info"
	var members struct {
		ActionStatus string
		ErrorCode    int
		ErrorInfo    string
		RequestID    string `json:"RequestId"`
		MemberList   []struct {
			MemberAccount string `json:"Member_Account"`
			Role          string
			MuteUntil     *int64
		}
	}
	if err := c.forwardQuery(ctx, "get_specified_group_member_info", map[string]any{
		"GroupId": group, "Member_List_Account": []string{sender}, "MemberInfoFilter": []string{"Role", "MuteUntil"},
	}, &members, gates); err != nil {
		return "", err
	}
	if err := c.forwardResponse(command, members.ActionStatus, members.ErrorCode, members.ErrorInfo, members.RequestID); err != nil {
		return "", err
	}
	if len(members.MemberList) != 1 || members.MemberList[0].MemberAccount != sender {
		return "", c.forwardError(command, 10007, false, "您已不在该群，无法转发", "", members.RequestID)
	}
	m := members.MemberList[0]
	if m.Role != "Owner" && m.Role != "Admin" && m.Role != "Member" {
		return "", c.forwardError(command, 10007, false, "无法确认您的群成员身份", "", members.RequestID)
	}
	if m.MuteUntil == nil || *m.MuteUntil < 0 {
		return "", c.forwardError(command, 0, true, "无法确认个人禁言状态", "", members.RequestID)
	}
	if *m.MuteUntil > time.Now().Unix() || (all == "On" && m.Role != "Owner" && m.Role != "Admin") {
		return "", c.forwardError(command, 10017, false, "您在该群已被禁言，无法转发", "", members.RequestID)
	}
	return info.GroupInfo[0].Name, nil
}

// Keep callbacks, moderation, offline push, history and sender synchronization.
// Use single C2C sends because batchsendmsg bypasses friend/block/mute checks.
func (c *Client) SendForward(ctx context.Context, sender, conversation string, body []MessageElement, random, sequence uint32, push *OfflinePushInfo) (string, error) {
	normalized, err := PrepareForwardBody(body)
	if err != nil {
		return "", err
	}
	req := map[string]any{"From_Account": sender, "MsgBody": body, "OfflinePushInfo": push}
	service, command := "openim", "sendmsg"
	if strings.HasPrefix(conversation, "GROUP") {
		service, command = "group_open_http_svc", "send_group_msg"
		req["GroupId"], req["Random"] = strings.TrimPrefix(conversation, "GROUP"), random
	} else {
		req["To_Account"], req["MsgRandom"], req["MsgSeq"], req["SyncOtherMachine"] = strings.TrimPrefix(conversation, "C2C"), random, sequence, 1
		req["SendMsgControl"] = []string{"WithMuteNotifications", "WithFriendCheck", "WithBlackListCheck", "WithSpeakingCheck"}
	}
	req["MsgBody"] = normalized
	var resp struct {
		ActionStatus  string
		ErrorCode     int
		ErrorInfo     string
		RequestID     string `json:"RequestId"`
		MsgDropReason string
		MsgID         string `json:"MsgId"`
		MsgSeq        uint64
	}
	if err := c.forwardPost(ctx, service, command, req, &resp); err != nil {
		return "", err
	}
	if err := c.forwardResponse(command, resp.ActionStatus, resp.ErrorCode, resp.ErrorInfo, resp.RequestID); err != nil {
		return "", err
	}
	if resp.MsgDropReason == "MsgFreqCtrl" {
		return "", c.forwardError(command, 10023, false, "腾讯频控丢弃，等待自动重试", "", resp.RequestID)
	}
	if resp.MsgDropReason != "" {
		return "", c.forwardError(command, 80001, false, "腾讯未投递该消息", "", resp.RequestID)
	}
	if resp.MsgID != "" {
		return resp.MsgID, nil
	}
	return fmt.Sprint(resp.MsgSeq), nil
}
