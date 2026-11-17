package tencentim

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// MessageElement 表示 IM 消息体中的单个元素。
type MessageElement struct {
	MsgType    string          `json:"MsgType"`
	MsgContent json.RawMessage `json:"MsgContent"`
}

// BatchSendTextMessageResult 表示批量发送单聊文本消息的结果。
type BatchSendTextMessageResult struct {
	MsgKey    string
	MsgID     string
	ErrorList map[string]int
}

// OfflinePushInfo 离线推送文案。
type OfflinePushInfo struct {
	PushFlag    int                     `json:"PushFlag"` // 0=推送
	Title       string                  `json:"Title,omitempty"`
	Desc        string                  `json:"Desc,omitempty"`
	Ext         string                  `json:"Ext,omitempty"` // 点击通知透传，客户端用来打开会话
	AndroidInfo *OfflinePushAndroidInfo `json:"AndroidInfo,omitempty"`
}

// OfflinePushAndroidInfo 对应腾讯云 IM REST API 的 OfflinePushInfo.AndroidInfo。
// 字段大小写必须与 REST API 保持一致。
type OfflinePushAndroidInfo struct {
	OPPOChannelID                string            `json:"OPPOChannelID,omitempty"`
	OPPOCategory                 string            `json:"OPPOCategory,omitempty"`
	OPPOPrivateMsgTemplateID     string            `json:"OPPOPrivateMsgTemplateId,omitempty"`
	OPPOPrivateTitleParameters   map[string]string `json:"OPPOPrivateTitleParameters,omitempty"`
	OPPOPrivateContentParameters map[string]string `json:"OPPOPrivateContentParameters,omitempty"`
	XiaoMiChannelID              string            `json:"XiaoMiChannelID,omitempty"`
	XiaoMiTemplateID             string            `json:"XiaoMiTemplateId,omitempty"`
	XiaoMiTemplateParam          map[string]string `json:"XiaoMiTemplateParam,omitempty"`
}

// ServerSystemNoticeCloudCustomData 标识由业务服务端生成、仅用于界面展示的系统提示。
// 消息前回调会结合 Operator_Account 校验该标识，避免把系统提示当成用户消息限流。
const ServerSystemNoticeCloudCustomData = "mllx_server_system_notice_v1"

// BatchSendTextMessage 通过 REST API 批量发送单聊文本消息。
func (c *Client) BatchSendTextMessage(fromAccount string, toAccounts []string, text string, syncToSender bool) (*BatchSendTextMessageResult, error) {
	return c.BatchSendTextMessageWithPush(fromAccount, toAccounts, text, syncToSender, nil)
}

// BatchSendTextMessageWithPush 批量发送单聊文本，并可选带离线推送。
func (c *Client) BatchSendTextMessageWithPush(
	fromAccount string,
	toAccounts []string,
	text string,
	syncToSender bool,
	push *OfflinePushInfo,
) (*BatchSendTextMessageResult, error) {
	type textContent struct {
		Text string `json:"Text"`
	}
	type req struct {
		SyncOtherMachine int      `json:"SyncOtherMachine"`
		FromAccount      string   `json:"From_Account,omitempty"`
		ToAccount        []string `json:"To_Account"`
		MsgRandom        uint32   `json:"MsgRandom"`
		MsgBody          []struct {
			MsgType    string      `json:"MsgType"`
			MsgContent textContent `json:"MsgContent"`
		} `json:"MsgBody"`
		OfflinePushInfo *OfflinePushInfo `json:"OfflinePushInfo,omitempty"`
	}
	type respBody struct {
		accountBaseResp
		MsgKey    string `json:"MsgKey"`
		MsgID     string `json:"MsgId"`
		ErrorList []struct {
			ToAccount string `json:"To_Account"`
			ErrorCode int    `json:"ErrorCode"`
		} `json:"ErrorList"`
	}

	syncOtherMachine := 2
	if syncToSender {
		syncOtherMachine = 1
	}

	body := req{
		SyncOtherMachine: syncOtherMachine,
		FromAccount:      fromAccount,
		ToAccount:        toAccounts,
		MsgRandom:        rand.Uint32(),
		MsgBody: []struct {
			MsgType    string      `json:"MsgType"`
			MsgContent textContent `json:"MsgContent"`
		}{
			{
				MsgType:    "TIMTextElem",
				MsgContent: textContent{Text: text},
			},
		},
		OfflinePushInfo: push,
	}

	var resp respBody
	if err := c.post(c.buildURL("openim", "batchsendmsg"), body, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}

	result := &BatchSendTextMessageResult{
		MsgKey:    resp.MsgKey,
		MsgID:     resp.MsgID,
		ErrorList: make(map[string]int, len(resp.ErrorList)),
	}
	for _, item := range resp.ErrorList {
		result.ErrorList[item.ToAccount] = item.ErrorCode
	}
	return result, nil
}

// GroupSendTextMessage 通过 REST API 发送群文本消息。
func (c *Client) GroupSendTextMessage(groupID, text, fromAccount string) error {
	type textContent struct {
		Text string `json:"Text"`
	}
	type req struct {
		GroupID        string `json:"GroupId"`
		FromAccount    string `json:"From_Account,omitempty"`
		Random         uint32 `json:"Random"`
		OnlineOnlyFlag int    `json:"OnlineOnlyFlag"`
		MsgBody        []struct {
			MsgType    string      `json:"MsgType"`
			MsgContent textContent `json:"MsgContent"`
		} `json:"MsgBody"`
	}

	body := req{
		GroupID:        groupID,
		FromAccount:    fromAccount,
		Random:         rand.Uint32(),
		OnlineOnlyFlag: 0,
		MsgBody: []struct {
			MsgType    string      `json:"MsgType"`
			MsgContent textContent `json:"MsgContent"`
		}{
			{
				MsgType:    "TIMTextElem",
				MsgContent: textContent{Text: text},
			},
		},
	}

	var resp accountBaseResp
	if err := c.post(c.buildURL("group_open_http_svc", "send_group_msg"), body, &resp); err != nil {
		return err
	}
	return resp.err()
}

// GroupSendCustomMessage 通过 REST API 发送群自定义消息。
func (c *Client) GroupSendCustomMessage(
	groupID, data, description, extension, fromAccount, cloudCustomData string,
) error {
	return c.GroupSendCustomMessageWithPush(
		groupID, data, description, extension, fromAccount, cloudCustomData, nil,
	)
}

// GroupSendCustomMessageWithPush 发送群自定义消息，并可选带离线推送。
func (c *Client) GroupSendCustomMessageWithPush(
	groupID, data, description, extension, fromAccount, cloudCustomData string,
	push *OfflinePushInfo,
) error {
	type customContent struct {
		Data string `json:"Data"`
		Desc string `json:"Desc,omitempty"`
		Ext  string `json:"Ext,omitempty"`
	}
	type req struct {
		GroupID         string           `json:"GroupId"`
		FromAccount     string           `json:"From_Account,omitempty"`
		Random          uint32           `json:"Random"`
		OnlineOnlyFlag  int              `json:"OnlineOnlyFlag"`
		SendMsgControl  []string         `json:"SendMsgControl,omitempty"`
		CloudCustomData string           `json:"CloudCustomData,omitempty"`
		OfflinePushInfo *OfflinePushInfo `json:"OfflinePushInfo,omitempty"`
		MsgBody         []struct {
			MsgType    string        `json:"MsgType"`
			MsgContent customContent `json:"MsgContent"`
		} `json:"MsgBody"`
	}

	body := req{
		GroupID:         groupID,
		FromAccount:     fromAccount,
		Random:          rand.Uint32(),
		OnlineOnlyFlag:  0,
		SendMsgControl:  []string{"NoUnread"},
		CloudCustomData: cloudCustomData,
		OfflinePushInfo: push,
		MsgBody: []struct {
			MsgType    string        `json:"MsgType"`
			MsgContent customContent `json:"MsgContent"`
		}{
			{
				MsgType: "TIMCustomElem",
				MsgContent: customContent{
					Data: data,
					Desc: description,
					Ext:  extension,
				},
			},
		},
	}

	var resp accountBaseResp
	if err := c.post(c.buildURL("group_open_http_svc", "send_group_msg"), body, &resp); err != nil {
		return err
	}
	return resp.err()
}

// C2CHistoryMessage 表示单聊历史消息记录。
type C2CHistoryMessage struct {
	FromAccount     string           `json:"From_Account"`
	ToAccount       string           `json:"To_Account"`
	MsgSeq          int64            `json:"MsgSeq"`
	MsgRandom       int64            `json:"MsgRandom"`
	MsgTimeStamp    int64            `json:"MsgTimeStamp"`
	MsgFlagBits     int              `json:"MsgFlagBits"`
	IsPeerRead      int              `json:"IsPeerRead"`
	MsgKey          string           `json:"MsgKey"`
	MsgBody         []MessageElement `json:"MsgBody"`
	CloudCustomData string           `json:"CloudCustomData"`
}

// GetC2CHistoryResult 表示拉取单聊历史消息的结果。
type GetC2CHistoryResult struct {
	Complete    int                 `json:"Complete"`
	MsgCnt      int                 `json:"MsgCnt"`
	LastMsgTime int64               `json:"LastMsgTime"`
	LastMsgKey  string              `json:"LastMsgKey"`
	MsgList     []C2CHistoryMessage `json:"MsgList"`
}

// GetC2CHistory 拉取指定会话的单聊历史消息。
func (c *Client) GetC2CHistory(operatorAccount, peerAccount string, maxCnt int, minTime, maxTime int64, lastMsgKey string) (*GetC2CHistoryResult, error) {
	type req struct {
		OperatorAccount string `json:"Operator_Account"`
		PeerAccount     string `json:"Peer_Account"`
		MaxCnt          int    `json:"MaxCnt"`
		MinTime         int64  `json:"MinTime"`
		MaxTime         int64  `json:"MaxTime"`
		LastMsgKey      string `json:"LastMsgKey,omitempty"`
	}
	type respBody struct {
		accountBaseResp
		Complete    int                 `json:"Complete"`
		MsgCnt      int                 `json:"MsgCnt"`
		LastMsgTime int64               `json:"LastMsgTime"`
		LastMsgKey  string              `json:"LastMsgKey"`
		MsgList     []C2CHistoryMessage `json:"MsgList"`
	}

	var resp respBody
	if err := c.post(c.buildURL("openim", "admin_getroammsg"), req{
		OperatorAccount: operatorAccount,
		PeerAccount:     peerAccount,
		MaxCnt:          maxCnt,
		MinTime:         minTime,
		MaxTime:         maxTime,
		LastMsgKey:      lastMsgKey,
	}, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}

	return &GetC2CHistoryResult{
		Complete:    resp.Complete,
		MsgCnt:      resp.MsgCnt,
		LastMsgTime: resp.LastMsgTime,
		LastMsgKey:  resp.LastMsgKey,
		MsgList:     resp.MsgList,
	}, nil
}

// WithdrawC2CMessage 撤回单聊消息。
func (c *Client) WithdrawC2CMessage(fromAccount, toAccount, msgKey string) error {
	type req struct {
		FromAccount string `json:"From_Account"`
		ToAccount   string `json:"To_Account"`
		MsgKey      string `json:"MsgKey"`
	}

	var resp accountBaseResp
	if err := c.post(c.buildURL("openim", "admin_msgwithdraw"), req{
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		MsgKey:      msgKey,
	}, &resp); err != nil {
		return err
	}
	return resp.err()
}

// RecallGroupMessage 管理员撤回群消息（无客户端 2 分钟限制）。
func (c *Client) RecallGroupMessage(groupID string, msgSeq uint64) error {
	type seqItem struct {
		MsgSeq uint64 `json:"MsgSeq"`
	}
	type req struct {
		GroupId    string    `json:"GroupId"`
		MsgSeqList []seqItem `json:"MsgSeqList"`
	}
	type recallRet struct {
		MsgSeq  uint64 `json:"MsgSeq"`
		RetCode int    `json:"RetCode"`
	}
	type respBody struct {
		accountBaseResp
		RecallRetList []recallRet `json:"RecallRetList"`
	}

	var resp respBody
	if err := c.post(c.buildURL("group_open_http_svc", "group_msg_recall"), req{
		GroupId: groupID,
		MsgSeqList: []seqItem{
			{MsgSeq: msgSeq},
		},
	}, &resp); err != nil {
		return err
	}
	if err := resp.err(); err != nil {
		return err
	}
	for _, item := range resp.RecallRetList {
		if item.MsgSeq == msgSeq && item.RetCode != 0 {
			return fmt.Errorf("撤回群消息失败(%d)", item.RetCode)
		}
	}
	return nil
}

// RecentConversationSession 表示会话列表中的单项记录。
type RecentConversationSession struct {
	Type      int    `json:"Type"`
	ToAccount string `json:"To_Account"`
	GroupID   string `json:"GroupId"`
	MsgTime   int64  `json:"MsgTime"`
	TopFlag   int    `json:"TopFlag"`
}

// GetRecentConversationListResult 表示拉取会话列表的结果。
type GetRecentConversationListResult struct {
	SessionItem   []RecentConversationSession `json:"SessionItem"`
	CompleteFlag  int                         `json:"CompleteFlag"`
	TimeStamp     int64                       `json:"TimeStamp"`
	StartIndex    int                         `json:"StartIndex"`
	TopTimeStamp  int64                       `json:"TopTimeStamp"`
	TopStartIndex int                         `json:"TopStartIndex"`
}

// GetRecentConversationList 拉取指定用户的最近会话列表。
func (c *Client) GetRecentConversationList(fromAccount string, limit int) (*GetRecentConversationListResult, error) {
	type req struct {
		FromAccount   string `json:"From_Account"`
		TimeStamp     int64  `json:"TimeStamp"`
		StartIndex    int    `json:"StartIndex"`
		TopTimeStamp  int64  `json:"TopTimeStamp"`
		TopStartIndex int    `json:"TopStartIndex"`
		AssistFlags   int    `json:"AssistFlags"`
	}
	type respBody struct {
		accountBaseResp
		SessionItem   []RecentConversationSession `json:"SessionItem"`
		CompleteFlag  int                         `json:"CompleteFlag"`
		TimeStamp     int64                       `json:"TimeStamp"`
		StartIndex    int                         `json:"StartIndex"`
		TopTimeStamp  int64                       `json:"TopTimeStamp"`
		TopStartIndex int                         `json:"TopStartIndex"`
	}

	if limit <= 0 {
		limit = 50
	}

	result := &GetRecentConversationListResult{
		SessionItem: make([]RecentConversationSession, 0, limit),
	}

	timeStamp := int64(0)
	startIndex := 0
	topTimeStamp := int64(0)
	topStartIndex := 0
	seen := make(map[string]bool)

	for {
		var resp respBody
		if err := c.post(c.buildURL("recentcontact", "get_list"), req{
			FromAccount:   fromAccount,
			TimeStamp:     timeStamp,
			StartIndex:    startIndex,
			TopTimeStamp:  topTimeStamp,
			TopStartIndex: topStartIndex,
			AssistFlags:   15,
		}, &resp); err != nil {
			return nil, err
		}
		if err := resp.err(); err != nil {
			return nil, err
		}

		for _, item := range resp.SessionItem {
			key := item.ToAccount
			if item.Type == 2 {
				key = item.GroupID
			}
			key = itemKey(item.Type, key)
			if seen[key] {
				continue
			}
			seen[key] = true
			result.SessionItem = append(result.SessionItem, item)
			if len(result.SessionItem) >= limit {
				result.CompleteFlag = resp.CompleteFlag
				result.TimeStamp = resp.TimeStamp
				result.StartIndex = resp.StartIndex
				result.TopTimeStamp = resp.TopTimeStamp
				result.TopStartIndex = resp.TopStartIndex
				result.SessionItem = result.SessionItem[:limit]
				return result, nil
			}
		}

		result.CompleteFlag = resp.CompleteFlag
		result.TimeStamp = resp.TimeStamp
		result.StartIndex = resp.StartIndex
		result.TopTimeStamp = resp.TopTimeStamp
		result.TopStartIndex = resp.TopStartIndex

		if resp.CompleteFlag == 1 {
			return result, nil
		}

		timeStamp = resp.TimeStamp
		startIndex = resp.StartIndex
		topTimeStamp = resp.TopTimeStamp
		topStartIndex = resp.TopStartIndex
	}
}

func itemKey(conversationType int, peerID string) string {
	return string(rune(conversationType)) + ":" + peerID
}

// DeleteRecentConversation 删除指定会话，并可选清理漫游消息。
func (c *Client) DeleteRecentConversation(fromAccount string, conversationType int, peerID string, clearRamble bool) error {
	type req struct {
		FromAccount string `json:"From_Account"`
		Type        int    `json:"Type"`
		ToAccount   string `json:"To_Account,omitempty"`
		ToGroupID   string `json:"ToGroupid,omitempty"`
		ClearRamble int    `json:"ClearRamble,omitempty"`
	}

	clearFlag := 0
	if clearRamble {
		clearFlag = 1
	}

	body := req{
		FromAccount: fromAccount,
		Type:        conversationType,
		ClearRamble: clearFlag,
	}
	if conversationType == 1 {
		body.ToAccount = peerID
	} else {
		body.ToGroupID = peerID
	}

	var resp accountBaseResp
	if err := c.post(c.buildURL("recentcontact", "delete"), body, &resp); err != nil {
		return err
	}
	return resp.err()
}

// GroupHistoryMessage 表示群聊历史消息记录。
type GroupHistoryMessage struct {
	FromAccount     string          `json:"From_Account"`
	IsPlaceMsg      int             `json:"IsPlaceMsg"`
	MsgBody         json.RawMessage `json:"MsgBody"`
	MsgPriority     int             `json:"MsgPriority"`
	MsgRandom       int64           `json:"MsgRandom"`
	MsgSeq          int64           `json:"MsgSeq"`
	MsgTimeStamp    int64           `json:"MsgTimeStamp"`
	CloudCustomData string          `json:"CloudCustomData"`
	IsSystemMsg     int             `json:"IsSystemMsg"`
}

// GetGroupHistoryResult 表示拉取群聊历史消息的结果。
type GetGroupHistoryResult struct {
	GroupID    string                `json:"GroupId"`
	IsFinished int                   `json:"IsFinished"`
	RspMsgList []GroupHistoryMessage `json:"RspMsgList"`
}

// GetGroupHistory 拉取指定群组的历史消息。
func (c *Client) GetGroupHistory(groupID string, reqMsgNumber int, reqMsgSeq int64) (*GetGroupHistoryResult, error) {
	type req struct {
		GroupID         string `json:"GroupId"`
		ReqMsgNumber    int    `json:"ReqMsgNumber"`
		ReqMsgSeq       int64  `json:"ReqMsgSeq,omitempty"`
		WithRecalledMsg int    `json:"WithRecalledMsg"`
	}
	type respBody struct {
		accountBaseResp
		GroupID    string                `json:"GroupId"`
		IsFinished int                   `json:"IsFinished"`
		RspMsgList []GroupHistoryMessage `json:"RspMsgList"`
	}

	var resp respBody
	if err := c.post(c.buildURL("group_open_http_svc", "group_msg_get_simple"), req{
		GroupID:         groupID,
		ReqMsgNumber:    reqMsgNumber,
		ReqMsgSeq:       reqMsgSeq,
		WithRecalledMsg: 1,
	}, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}

	return &GetGroupHistoryResult{
		GroupID:    resp.GroupID,
		IsFinished: resp.IsFinished,
		RspMsgList: resp.RspMsgList,
	}, nil
}
