package controller_admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

const (
	conversationTypeC2C   = 1
	conversationTypeGroup = 2
	c2cExportPageSize     = 100
	c2cExportConvLimit    = 1000
	c2cExportWindow       = 30 * 24 * time.Hour
)

// IMSendC2CReq 表示后台批量发送单聊文本消息的参数。
type IMSendC2CReq struct {
	FromUserID   string
	ToUserIDs    []string
	Content      string
	SyncToSender bool
}

// IMSendResult 表示单个目标账号的发送结果。
type IMSendResult struct {
	ToUserID string `json:"toUserID"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}

// IMSendC2CResult 表示批量发送单聊文本消息的结果。
type IMSendC2CResult struct {
	MsgKey  string         `json:"msgKey"`
	MsgID   string         `json:"msgId"`
	Results []IMSendResult `json:"results"`
}

// IMHistoryMessage 表示前端可直接展示的消息记录。
type IMHistoryMessage struct {
	MsgKey     string `json:"msgKey,omitempty"`
	MsgSeq     int64  `json:"msgSeq"`
	MsgRandom  int64  `json:"msgRandom,omitempty"`
	FromUserID string `json:"fromUserID"`
	ToUserID   string `json:"toUserID,omitempty"`
	GroupID    string `json:"groupID,omitempty"`
	MsgTime    int64  `json:"msgTime"`
	MsgType    string `json:"msgType,omitempty"`
	Content    string `json:"content"`
	IsRecalled bool   `json:"isRecalled"`
	IsPlaceMsg bool   `json:"isPlaceMsg"`
}

// IMQueryC2CHistoryReq 表示查询单聊历史消息的参数。
type IMQueryC2CHistoryReq struct {
	OperatorUserID string
	PeerUserID     string
	PageSize       int
	MinTime        int64
	MaxTime        int64
	LastMsgKey     string
}

// IMQueryC2CHistoryResult 表示查询单聊历史消息的结果。
type IMQueryC2CHistoryResult struct {
	Complete       bool               `json:"complete"`
	NextLastMsgKey string             `json:"nextLastMsgKey,omitempty"`
	NextMaxTime    int64              `json:"nextMaxTime,omitempty"`
	List           []IMHistoryMessage `json:"list"`
}

// IMWithdrawC2CReq 表示撤回单聊消息的参数。
type IMWithdrawC2CReq struct {
	FromUserID string
	ToUserID   string
	MsgKey     string
}

// IMConversation 表示会话列表中的单项。
type IMConversation struct {
	Type        string `json:"type"`
	PeerID      string `json:"peerID"`
	PeerLabel   string `json:"peerLabel"`
	LastMsgTime int64  `json:"lastMsgTime"`
	TopFlag     bool   `json:"topFlag"`
}

// IMUserSearchResult 表示 IM 用户候选。
type IMUserSearchResult struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	IMUserID string `json:"imUserID"`
}

// IMGroupSearchResult 表示 IM 群组候选。
type IMGroupSearchResult struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	IMGroupID string `json:"imGroupId"`
	GroupType string `json:"groupType"`
}

// IMDeleteConversationReq 表示删除单个会话的参数。
type IMDeleteConversationReq struct {
	FromUserID  string
	Type        string
	PeerID      string
	ClearRamble bool
}

// IMQueryGroupHistoryReq 表示查询群聊历史消息的参数。
type IMQueryGroupHistoryReq struct {
	GroupID        string
	OperatorUserID string
	PageSize       int
	LastMsgSeq     int64
}

// IMQueryGroupHistoryResult 表示查询群聊历史消息的结果。
type IMQueryGroupHistoryResult struct {
	IsFinished bool               `json:"isFinished"`
	NextMsgSeq int64              `json:"nextMsgSeq,omitempty"`
	List       []IMHistoryMessage `json:"list"`
}

// IMExportC2CPairReq 表示导出两个用户之间聊天记录的参数。
type IMExportC2CPairReq struct {
	OperatorUserID string
	PeerUserID     string
}

// IMExportUserC2CReq 表示导出单个用户全部单聊记录的参数。
type IMExportUserC2CReq struct {
	UserID string
}

// IMExportC2CRecord 表示导出的单聊记录行。
type IMExportC2CRecord struct {
	ScopeMode     string `json:"scopeMode"`
	OwnerUserID   string `json:"ownerUserID"`
	OwnerNickname string `json:"ownerNickname"`
	OwnerPhone    string `json:"ownerPhone"`
	PeerUserID    string `json:"peerUserID"`
	PeerNickname  string `json:"peerNickname"`
	PeerPhone     string `json:"peerPhone"`
	FromUserID    string `json:"fromUserID"`
	FromNickname  string `json:"fromNickname"`
	ToUserID      string `json:"toUserID"`
	ToNickname    string `json:"toNickname"`
	MsgTime       int64  `json:"msgTime"`
	MsgType       string `json:"msgType"`
	Content       string `json:"content"`
	MsgKey        string `json:"msgKey"`
	MsgSeq        int64  `json:"msgSeq"`
	MsgRandom     int64  `json:"msgRandom"`
	IsRecalled    bool   `json:"isRecalled"`
}

type exportUserMeta struct {
	Nickname string
	Phone    string
}

type c2cExportBundle struct {
	scopeMode   string
	ownerUserID string
	peerUserID  string
	messages    []IMHistoryMessage
}

func ensureManagedIMUsers(actorID uint, isAdmin bool, userIDs []string) error {
	for _, userID := range sanitizeIDs(userIDs) {
		if _, err := findManagedUserByIMUserID(actorID, isAdmin, userID); err != nil {
			return err
		}
	}
	return nil
}

// SendC2CTextMessages 通过腾讯云 IM 批量发送单聊文本消息。
func SendC2CTextMessages(actorID uint, isAdmin bool, req IMSendC2CReq) (*IMSendC2CResult, error) {
	req.FromUserID = strings.TrimSpace(req.FromUserID)
	req.Content = strings.TrimSpace(req.Content)
	if req.FromUserID == "" {
		return nil, errors.New("发送方 IM 用户 ID 不能为空")
	}
	if req.Content == "" {
		return nil, errors.New("消息内容不能为空")
	}

	shouldFilterSender, err := keywordfilter.ShouldFilterSender(req.FromUserID)
	if err != nil {
		return nil, errors.New("关键词规则加载失败")
	}
	if shouldFilterSender {
		maskedContent, _, err := keywordfilter.MaskText(req.Content)
		if err != nil {
			return nil, errors.New("关键词规则加载失败")
		}
		req.Content = maskedContent
	}

	_ = actorID
	toUserIDs := sanitizeIDs(req.ToUserIDs)
	if len(toUserIDs) == 0 {
		return nil, errors.New("接收方 IM 用户 ID 不能为空")
	}
	if len(toUserIDs) > 500 {
		return nil, errors.New("单次最多只能发送给 500 个账号")
	}
	if err := ensureManagedIMUsers(actorID, isAdmin, append([]string{req.FromUserID}, toUserIDs...)); err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	senderName := req.FromUserID
	if sender, findErr := findManagedUserByIMUserID(actorID, isAdmin, req.FromUserID); findErr == nil {
		if nickname := strings.TrimSpace(sender.Nickname); nickname != "" {
			senderName = nickname
		}
	}
	push := tencentim.NewC2CTextOfflinePushInfo(
		senderName,
		req.Content,
		"",
		senderName,
		req.Content,
	)
	resp, err := imClient.BatchSendTextMessageWithPush(
		req.FromUserID,
		toUserIDs,
		req.Content,
		req.SyncToSender,
		push,
	)
	if err != nil {
		return nil, err
	}

	results := make([]IMSendResult, 0, len(toUserIDs))
	for _, toUserID := range toUserIDs {
		code, failed := resp.ErrorList[toUserID]
		if failed && code != 0 {
			results = append(results, IMSendResult{
				ToUserID: toUserID,
				Success:  false,
				Message:  "发送失败",
			})
			continue
		}
		results = append(results, IMSendResult{
			ToUserID: toUserID,
			Success:  true,
			Message:  "发送成功",
		})
	}

	return &IMSendC2CResult{
		MsgKey:  resp.MsgKey,
		MsgID:   resp.MsgID,
		Results: results,
	}, nil
}

// QueryC2CHistory 查询单聊历史消息。
func QueryC2CHistory(actorID uint, isAdmin bool, req IMQueryC2CHistoryReq) (*IMQueryC2CHistoryResult, error) {
	req.OperatorUserID = strings.TrimSpace(req.OperatorUserID)
	req.PeerUserID = strings.TrimSpace(req.PeerUserID)
	if req.OperatorUserID == "" || req.PeerUserID == "" {
		return nil, errors.New("请填写完整的双方 IM 用户 ID")
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, req.OperatorUserID); err != nil {
		return nil, err
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, req.PeerUserID); err != nil {
		return nil, err
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 50
	}
	if req.MaxTime <= 0 {
		req.MaxTime = time.Now().Unix()
	}
	if req.MinTime <= 0 {
		req.MinTime = req.MaxTime - 7*24*3600
	}
	if req.MinTime > req.MaxTime {
		return nil, errors.New("开始时间不能大于结束时间")
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	resp, err := imClient.GetC2CHistory(req.OperatorUserID, req.PeerUserID, req.PageSize, req.MinTime, req.MaxTime, req.LastMsgKey)
	if err != nil {
		return nil, err
	}

	list := make([]IMHistoryMessage, 0, len(resp.MsgList))
	for _, item := range resp.MsgList {
		list = append(list, IMHistoryMessage{
			MsgKey:     item.MsgKey,
			MsgSeq:     item.MsgSeq,
			MsgRandom:  item.MsgRandom,
			FromUserID: item.FromAccount,
			ToUserID:   item.ToAccount,
			MsgTime:    item.MsgTimeStamp,
			MsgType:    summarizeMessageType(item.MsgBody),
			Content:    summarizeMessageElements(item.MsgBody),
			IsRecalled: item.MsgFlagBits == 8,
			IsPlaceMsg: false,
		})
	}

	return &IMQueryC2CHistoryResult{
		Complete:       resp.Complete == 1,
		NextLastMsgKey: resp.LastMsgKey,
		NextMaxTime:    resp.LastMsgTime,
		List:           list,
	}, nil
}

// ExportC2CHistoryByPair 导出两个用户之间最近 30 天单聊记录。
func ExportC2CHistoryByPair(actorID uint, isAdmin bool, req IMExportC2CPairReq) ([]IMExportC2CRecord, error) {
	req.OperatorUserID = strings.TrimSpace(req.OperatorUserID)
	req.PeerUserID = strings.TrimSpace(req.PeerUserID)
	if req.OperatorUserID == "" || req.PeerUserID == "" {
		return nil, errors.New("请填写完整的双方 IM 用户 ID")
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, req.OperatorUserID); err != nil {
		return nil, err
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, req.PeerUserID); err != nil {
		return nil, err
	}

	messages, err := collectC2CHistoryWindow(req.OperatorUserID, req.PeerUserID)
	if err != nil {
		return nil, err
	}

	return buildExportRecords([]c2cExportBundle{{
		scopeMode:   "pair",
		ownerUserID: req.OperatorUserID,
		peerUserID:  req.PeerUserID,
		messages:    messages,
	}})
}

// WithdrawC2CMessage 撤回单聊消息。
func WithdrawC2CMessage(actorID uint, isAdmin bool, req IMWithdrawC2CReq) error {
	req.FromUserID = strings.TrimSpace(req.FromUserID)
	req.ToUserID = strings.TrimSpace(req.ToUserID)
	req.MsgKey = strings.TrimSpace(req.MsgKey)
	if req.FromUserID == "" || req.ToUserID == "" || req.MsgKey == "" {
		return errors.New("撤回消息参数不完整")
	}
	if err := ensureManagedIMUsers(actorID, isAdmin, []string{req.FromUserID, req.ToUserID}); err != nil {
		return err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return errors.New("IM 未配置")
	}
	return imClient.WithdrawC2CMessage(req.FromUserID, req.ToUserID, req.MsgKey)
}

// ListRecentConversations 拉取指定用户的会话列表。
func ListRecentConversations(actorID uint, isAdmin bool, fromUserID string) ([]IMConversation, error) {
	fromUserID = strings.TrimSpace(fromUserID)
	if fromUserID == "" {
		return nil, errors.New("IM 用户 ID 不能为空")
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, fromUserID); err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	resp, err := imClient.GetRecentConversationList(fromUserID, 50)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepository()
	groupRepo := repository.NewGroupRepository()
	groupIDs := make([]string, 0)
	userIDs := make([]string, 0)
	seenUserIDs := make(map[string]bool)
	seenGroupIDs := make(map[string]bool)
	for _, item := range resp.SessionItem {
		switch item.Type {
		case conversationTypeC2C:
			if item.ToAccount != "" && !seenUserIDs[item.ToAccount] {
				userIDs = append(userIDs, item.ToAccount)
				seenUserIDs[item.ToAccount] = true
			}
		case conversationTypeGroup:
			if item.GroupID != "" && !seenGroupIDs[item.GroupID] {
				groupIDs = append(groupIDs, item.GroupID)
				seenGroupIDs[item.GroupID] = true
			}
		}
	}

	userMap := map[string]string{}
	if users, err := userRepo.FindByIMUserIDs(userIDs); err == nil {
		for _, user := range users {
			userMap[user.IMUserID] = user.Nickname
		}
	}
	if userInfos, err := imClient.AccountGetInfo(userIDs); err == nil {
		for userID, userInfo := range userInfos {
			if userMap[userID] == "" && strings.TrimSpace(userInfo.Nickname) != "" {
				userMap[userID] = userInfo.Nickname
			}
		}
	}

	groupMap := map[string]string{}
	for _, groupID := range groupIDs {
		group, err := groupRepo.FindByIMGroupID(groupID)
		if err == nil {
			groupMap[group.IMGroupID] = group.Name
		}
	}
	if groupInfos, err := imClient.GroupGetInfo(groupIDs); err == nil {
		for groupID, groupInfo := range groupInfos {
			if groupMap[groupID] == "" && strings.TrimSpace(groupInfo.Name) != "" {
				groupMap[groupID] = groupInfo.Name
			}
		}
	}

	result := make([]IMConversation, 0, len(resp.SessionItem))
	for _, item := range resp.SessionItem {
		convType := "C2C"
		peerID := item.ToAccount
		peerLabel := item.ToAccount
		if item.Type == conversationTypeGroup {
			convType = "Group"
			peerID = item.GroupID
			peerLabel = item.GroupID
			if groupName := groupMap[item.GroupID]; groupName != "" {
				peerLabel = groupName
			}
		} else if nickname := userMap[item.ToAccount]; nickname != "" {
			peerLabel = nickname
		}

		result = append(result, IMConversation{
			Type:        convType,
			PeerID:      peerID,
			PeerLabel:   peerLabel,
			LastMsgTime: item.MsgTime,
			TopFlag:     item.TopFlag == 1,
		})
	}
	return result, nil
}

// ExportUserC2CHistory 导出某个用户最近 30 天全部单聊记录。
func ExportUserC2CHistory(actorID uint, isAdmin bool, req IMExportUserC2CReq) ([]IMExportC2CRecord, error) {
	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		return nil, errors.New("IM 用户 ID 不能为空")
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, req.UserID); err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	conversations, err := imClient.GetRecentConversationList(req.UserID, c2cExportConvLimit)
	if err != nil {
		return nil, err
	}

	minTime, _ := exportWindowRange()
	peerIDs := make([]string, 0, len(conversations.SessionItem))
	seen := make(map[string]struct{}, len(conversations.SessionItem))
	for _, item := range conversations.SessionItem {
		if item.Type != conversationTypeC2C {
			continue
		}
		peerID := strings.TrimSpace(item.ToAccount)
		if peerID == "" {
			continue
		}
		if item.MsgTime > 0 && item.MsgTime < minTime {
			continue
		}
		if _, ok := seen[peerID]; ok {
			continue
		}
		if _, err := findManagedUserByIMUserID(actorID, isAdmin, peerID); err != nil {
			continue
		}
		seen[peerID] = struct{}{}
		peerIDs = append(peerIDs, peerID)
	}

	bundles := make([]c2cExportBundle, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		messages, err := collectC2CHistoryWindow(req.UserID, peerID)
		if err != nil {
			return nil, fmt.Errorf("导出与 %s 的聊天记录失败: %w", peerID, err)
		}
		bundles = append(bundles, c2cExportBundle{
			scopeMode:   "single",
			ownerUserID: req.UserID,
			peerUserID:  peerID,
			messages:    messages,
		})
	}

	return buildExportRecords(bundles)
}

// SearchIMUsers 在后台按昵称搜索可用于 IM 工具的用户候选。
func SearchIMUsers(actorID uint, isAdmin bool, keyword string) ([]IMUserSearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	actor, err := BuildActorContext(actorID)
	if err != nil {
		return nil, err
	}
	scopeOwnerIDs := []uint(nil)
	ownerID := uint(0)
	if !IsSuperAdminRole(actor.Role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
		if IsAssistantRole(actor.Role) {
			ownerID = actorID
		}
	}

	userRepo := repository.NewUserRepository()
	users, err := userRepo.SearchForIMAdmin(scopeOwnerIDs, ownerID, keyword, 50)
	if err != nil {
		return nil, err
	}

	results := make([]IMUserSearchResult, 0, len(users))
	for _, user := range users {
		results = append(results, IMUserSearchResult{
			ID:       user.ID,
			Nickname: user.Nickname,
			Phone:    user.Phone,
			IMUserID: user.IMUserID,
		})
	}
	return results, nil
}

// SearchIMGroups 在后台按群组名称搜索可用于 IM 工具的群组候选。
func SearchIMGroups(actorID uint, isAdmin bool, keyword string) ([]IMGroupSearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	actor, err := BuildActorContext(actorID)
	if err != nil {
		return nil, err
	}
	scopeOwnerIDs := []uint(nil)
	ownerID := uint(0)
	if !IsSuperAdminRole(actor.Role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
		if IsAssistantRole(actor.Role) {
			ownerID = actorID
		}
	}

	groupRepo := repository.NewGroupRepository()
	groups, err := groupRepo.SearchForIMAdmin(scopeOwnerIDs, ownerID, keyword, 50)
	if err != nil {
		return nil, err
	}

	results := make([]IMGroupSearchResult, 0, len(groups))
	for _, group := range groups {
		results = append(results, IMGroupSearchResult{
			ID:        group.ID,
			Name:      group.Name,
			IMGroupID: group.IMGroupID,
			GroupType: group.GroupType,
		})
	}
	return results, nil
}

// DeleteConversation 删除指定会话，并可选清理漫游消息。
func DeleteConversation(actorID uint, isAdmin bool, req IMDeleteConversationReq) error {
	req.FromUserID = strings.TrimSpace(req.FromUserID)
	req.PeerID = strings.TrimSpace(req.PeerID)
	if req.FromUserID == "" || req.PeerID == "" {
		return errors.New("删除会话参数不完整")
	}
	if _, err := findManagedUserByIMUserID(actorID, isAdmin, req.FromUserID); err != nil {
		return err
	}

	conversationType, err := parseConversationType(req.Type)
	if err != nil {
		return err
	}
	if !isAdmin {
		switch conversationType {
		case conversationTypeC2C:
			if _, err := findManagedUserByIMUserID(actorID, false, req.PeerID); err != nil {
				return err
			}
		case conversationTypeGroup:
			if _, err := findManagedGroupByIMGroupID(actorID, false, req.PeerID); err != nil {
				return err
			}
		}
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return errors.New("IM 未配置")
	}
	return imClient.DeleteRecentConversation(req.FromUserID, conversationType, req.PeerID, req.ClearRamble)
}

// QueryGroupHistory 拉取群聊历史消息。
func QueryGroupHistory(actorID uint, isAdmin bool, req IMQueryGroupHistoryReq) (*IMQueryGroupHistoryResult, error) {
	req.GroupID = strings.TrimSpace(req.GroupID)
	req.OperatorUserID = strings.TrimSpace(req.OperatorUserID)
	if req.GroupID == "" {
		return nil, errors.New("群组 ID 不能为空")
	}
	if !isAdmin {
		if _, err := findManagedGroupByIMGroupID(actorID, false, req.GroupID); err != nil {
			return nil, err
		}
		if req.OperatorUserID != "" {
			if _, err := findManagedUserByIMUserID(actorID, false, req.OperatorUserID); err != nil {
				return nil, err
			}
		}
	}
	if req.PageSize <= 0 || req.PageSize > 50 {
		req.PageSize = 50
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	resp, err := imClient.GetGroupHistory(req.GroupID, req.PageSize, req.LastMsgSeq)
	if err != nil {
		return nil, err
	}

	list := make([]IMHistoryMessage, 0, len(resp.RspMsgList))
	var minSeq int64
	for i, item := range resp.RspMsgList {
		content := summarizeRawMessageBody(item.MsgBody)
		if item.IsPlaceMsg == 1 {
			content = "[消息不可用]"
		}
		if i == 0 || item.MsgSeq < minSeq {
			minSeq = item.MsgSeq
		}
		list = append(list, IMHistoryMessage{
			MsgSeq:     item.MsgSeq,
			MsgRandom:  item.MsgRandom,
			FromUserID: item.FromAccount,
			GroupID:    resp.GroupID,
			MsgTime:    item.MsgTimeStamp,
			MsgType:    summarizeRawMessageType(item.MsgBody),
			Content:    content,
			IsRecalled: item.IsPlaceMsg == 1 && strings.Contains(content, "撤回"),
			IsPlaceMsg: item.IsPlaceMsg == 1,
		})
	}

	nextMsgSeq := int64(0)
	if resp.IsFinished == 0 && minSeq > 0 {
		nextMsgSeq = minSeq - 1
	}

	return &IMQueryGroupHistoryResult{
		IsFinished: resp.IsFinished == 1,
		NextMsgSeq: nextMsgSeq,
		List:       list,
	}, nil
}

func sanitizeIDs(ids []string) []string {
	result := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func parseConversationType(value string) (int, error) {
	switch strings.TrimSpace(strings.ToUpper(value)) {
	case "C2C":
		return conversationTypeC2C, nil
	case "GROUP":
		return conversationTypeGroup, nil
	default:
		return 0, errors.New("不支持的会话类型")
	}
}

func summarizeRawMessageBody(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "[空消息]"
	}

	var arrayBody []tencentim.MessageElement
	if err := json.Unmarshal(raw, &arrayBody); err == nil {
		return summarizeMessageElements(arrayBody)
	}

	var singleBody tencentim.MessageElement
	if err := json.Unmarshal(raw, &singleBody); err == nil {
		return summarizeMessageElements([]tencentim.MessageElement{singleBody})
	}

	return "[无法解析的消息]"
}

func summarizeRawMessageType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "Unknown"
	}

	var arrayBody []tencentim.MessageElement
	if err := json.Unmarshal(raw, &arrayBody); err == nil {
		return summarizeMessageType(arrayBody)
	}

	var singleBody tencentim.MessageElement
	if err := json.Unmarshal(raw, &singleBody); err == nil {
		return summarizeMessageType([]tencentim.MessageElement{singleBody})
	}

	return "Unknown"
}

func summarizeMessageElements(elements []tencentim.MessageElement) string {
	if len(elements) == 0 {
		return "[空消息]"
	}

	parts := make([]string, 0, len(elements))
	for _, element := range elements {
		parts = append(parts, summarizeMessageElement(element))
	}
	return strings.Join(parts, " ")
}

func summarizeMessageType(elements []tencentim.MessageElement) string {
	if len(elements) == 0 {
		return "Unknown"
	}
	return elements[0].MsgType
}

func summarizeMessageElement(element tencentim.MessageElement) string {
	switch element.MsgType {
	case "TIMTextElem":
		var content struct {
			Text string `json:"Text"`
		}
		if json.Unmarshal(element.MsgContent, &content) == nil && content.Text != "" {
			return content.Text
		}
		return "[文本消息]"
	case "TIMCustomElem":
		var content struct {
			Desc string `json:"Desc"`
			Data string `json:"Data"`
		}
		if json.Unmarshal(element.MsgContent, &content) == nil {
			if content.Desc != "" {
				return "[自定义] " + content.Desc
			}
			if content.Data != "" {
				return "[自定义消息]"
			}
		}
		return "[自定义消息]"
	case "TIMImageElem":
		return "[图片]"
	case "TIMVideoFileElem":
		return "[视频]"
	case "TIMSoundElem":
		return "[语音]"
	case "TIMFaceElem":
		return "[表情]"
	case "TIMFileElem":
		var content struct {
			FileName string `json:"FileName"`
		}
		if json.Unmarshal(element.MsgContent, &content) == nil && content.FileName != "" {
			return "[文件] " + content.FileName
		}
		return "[文件]"
	case "TIMLocationElem":
		var content struct {
			Desc string `json:"Desc"`
		}
		if json.Unmarshal(element.MsgContent, &content) == nil && content.Desc != "" {
			return "[位置] " + content.Desc
		}
		return "[位置]"
	default:
		return "[" + element.MsgType + "]"
	}
}

func exportWindowRange() (int64, int64) {
	maxTime := time.Now().Unix()
	minTime := maxTime - int64(c2cExportWindow.Seconds())
	return minTime, maxTime
}

func collectC2CHistoryWindow(operatorUserID, peerUserID string) ([]IMHistoryMessage, error) {
	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	minTime, maxTime := exportWindowRange()
	lastMsgKey := ""
	seen := map[string]struct{}{}
	list := make([]IMHistoryMessage, 0, 256)

	for {
		resp, err := imClient.GetC2CHistory(
			operatorUserID,
			peerUserID,
			c2cExportPageSize,
			minTime,
			maxTime,
			lastMsgKey,
		)
		if err != nil {
			return nil, err
		}

		for _, item := range resp.MsgList {
			message := IMHistoryMessage{
				MsgKey:     item.MsgKey,
				MsgSeq:     item.MsgSeq,
				MsgRandom:  item.MsgRandom,
				FromUserID: item.FromAccount,
				ToUserID:   item.ToAccount,
				MsgTime:    item.MsgTimeStamp,
				MsgType:    summarizeMessageType(item.MsgBody),
				Content:    summarizeMessageElements(item.MsgBody),
				IsRecalled: item.MsgFlagBits == 8,
				IsPlaceMsg: false,
			}
			identity := message.MsgKey
			if identity == "" {
				identity = fmt.Sprintf(
					"%d:%d:%d:%s:%s",
					message.MsgSeq,
					message.MsgRandom,
					message.MsgTime,
					message.FromUserID,
					message.ToUserID,
				)
			}
			if _, exists := seen[identity]; exists {
				continue
			}
			seen[identity] = struct{}{}
			list = append(list, message)
		}

		if resp.Complete == 1 || resp.LastMsgKey == "" || len(resp.MsgList) == 0 {
			break
		}

		lastMsgKey = resp.LastMsgKey
		if resp.LastMsgTime > 0 {
			maxTime = resp.LastMsgTime
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].MsgTime == list[j].MsgTime {
			if list[i].MsgSeq == list[j].MsgSeq {
				return list[i].MsgRandom < list[j].MsgRandom
			}
			return list[i].MsgSeq < list[j].MsgSeq
		}
		return list[i].MsgTime < list[j].MsgTime
	})

	return list, nil
}

func buildExportRecords(bundles []c2cExportBundle) ([]IMExportC2CRecord, error) {
	if len(bundles) == 0 {
		return []IMExportC2CRecord{}, nil
	}

	seenUserIDs := make(map[string]struct{})
	userIDs := make([]string, 0)
	for _, bundle := range bundles {
		for _, userID := range []string{bundle.ownerUserID, bundle.peerUserID} {
			if userID == "" {
				continue
			}
			if _, ok := seenUserIDs[userID]; ok {
				continue
			}
			seenUserIDs[userID] = struct{}{}
			userIDs = append(userIDs, userID)
		}
		for _, message := range bundle.messages {
			for _, userID := range []string{message.FromUserID, message.ToUserID} {
				if userID == "" {
					continue
				}
				if _, ok := seenUserIDs[userID]; ok {
					continue
				}
				seenUserIDs[userID] = struct{}{}
				userIDs = append(userIDs, userID)
			}
		}
	}

	users, err := repository.NewUserRepository().FindByIMUserIDs(userIDs)
	if err != nil {
		return nil, err
	}
	metaMap := make(map[string]exportUserMeta, len(users))
	for _, user := range users {
		metaMap[user.IMUserID] = exportUserMeta{
			Nickname: user.Nickname,
			Phone:    user.Phone,
		}
	}

	rows := make([]IMExportC2CRecord, 0)
	for _, bundle := range bundles {
		ownerMeta := metaMap[bundle.ownerUserID]
		peerMeta := metaMap[bundle.peerUserID]
		for _, message := range bundle.messages {
			fromMeta := metaMap[message.FromUserID]
			toMeta := metaMap[message.ToUserID]
			rows = append(rows, IMExportC2CRecord{
				ScopeMode:     bundle.scopeMode,
				OwnerUserID:   bundle.ownerUserID,
				OwnerNickname: ownerMeta.Nickname,
				OwnerPhone:    ownerMeta.Phone,
				PeerUserID:    bundle.peerUserID,
				PeerNickname:  peerMeta.Nickname,
				PeerPhone:     peerMeta.Phone,
				FromUserID:    message.FromUserID,
				FromNickname:  fromMeta.Nickname,
				ToUserID:      message.ToUserID,
				ToNickname:    toMeta.Nickname,
				MsgTime:       message.MsgTime,
				MsgType:       message.MsgType,
				Content:       message.Content,
				MsgKey:        message.MsgKey,
				MsgSeq:        message.MsgSeq,
				MsgRandom:     message.MsgRandom,
				IsRecalled:    message.IsRecalled,
			})
		}
	}

	return rows, nil
}
