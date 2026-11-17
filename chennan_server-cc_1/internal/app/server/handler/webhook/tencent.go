package handler_webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/grouprisk"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/quietlog"
	"github.com/ixayda/ix-Go/internal/app/server/quietpolicy"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
	"github.com/ixayda/ix-Go/pkg/idgs"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

const (
	cmdPrevFriendAdd             = "Sns.CallbackPrevFriendAdd"
	cmdPrevFriendResponse        = "Sns.CallbackPrevFriendResponse"
	cmdFriendAdd                 = "Sns.CallbackFriendAdd"
	cmdPrevCreateGroup           = "Group.CallbackBeforeCreateGroup"
	cmdAfterCreateGroup          = "Group.CallbackAfterCreateGroup"
	cmdPrevApplyJoinGroup        = "Group.CallbackBeforeApplyJoinGroup"
	cmdPrevInviteJoin            = "Group.CallbackBeforeInviteJoinGroup"
	cmdStateChange               = "State.StateChange"
	cmdAfterMemberFieldChanged   = "Group.CallbackAfterMemberFieldChanged"
	errCodeCrossOwner            = 38000
	errCodeIMReject              = 10101
	errCodeIMGroupMissing        = 10102
	errCodeAddFriendQuietHours   = 38001
	errCodeCreateGroupQuietHours = 38002
	errCodeJoinGroupQuietHours   = 38003
	errCodeRateLimited           = 38429
)

type friendItem struct {
	ToAccount string `json:"To_Account"`
}

type prevFriendAddReq struct {
	CallbackCommand  string       `json:"CallbackCommand"`
	FromAccount      string       `json:"From_Account"`
	RequesterAccount string       `json:"Requester_Account"`
	FriendItem       []friendItem `json:"FriendItem"`
	AddFriendItem    []friendItem `json:"AddFriendItem"`
}

type friendAddPair struct {
	FromAccount      string `json:"From_Account"`
	ToAccount        string `json:"To_Account"`
	InitiatorAccount string `json:"Initiator_Account"`
}

type friendAddReq struct {
	CallbackCommand string          `json:"CallbackCommand"`
	PairList        []friendAddPair `json:"PairList"`
	ClientCmd       string          `json:"ClientCmd"`
	FromAccount     string          `json:"From_Account"`
	FriendItem      []friendItem    `json:"FriendItem"`
}

type responseFriendItem struct {
	ToAccount      string `json:"To_Account"`
	ResponseAction string `json:"ResponseAction"`
}

type prevFriendResponseReq struct {
	CallbackCommand    string               `json:"CallbackCommand"`
	FromAccount        string               `json:"From_Account"`
	RequesterAccount   string               `json:"Requester_Account"`
	ResponseFriendItem []responseFriendItem `json:"ResponseFriendItem"`
	FriendItem         []responseFriendItem `json:"FriendItem"`
}

type friendItemResult struct {
	ToAccount  string `json:"To_Account"`
	ResultCode int    `json:"ResultCode"`
	ResultInfo string `json:"ResultInfo"`
}

type prevFriendAddResp struct {
	ActionStatus string             `json:"ActionStatus"`
	ErrorCode    int                `json:"ErrorCode"`
	ErrorInfo    string             `json:"ErrorInfo"`
	ResultItem   []friendItemResult `json:"ResultItem"`
}

type groupMemberItem struct {
	MemberAccount string `json:"Member_Account"`
}

type prevCreateGroupReq struct {
	CallbackCommand string            `json:"CallbackCommand"`
	OperatorAccount string            `json:"Operator_Account"`
	MemberList      []groupMemberItem `json:"MemberList"`
}

type afterCreateGroupReq struct {
	CallbackCommand string `json:"CallbackCommand"`
	GroupID         string `json:"GroupId"`
	GroupName       string `json:"Name"`
	GroupType       string `json:"Type"`
	OwnerAccount    string `json:"Owner_Account"`
	OperatorAccount string `json:"Operator_Account"`
	CreateTime      int64  `json:"CreateTime"`
}

type prevApplyJoinGroupReq struct {
	CallbackCommand  string `json:"CallbackCommand"`
	RequestorAccount string `json:"Requestor_Account"`
	GroupID          string `json:"GroupId"`
}

type prevInviteJoinGroupReq struct {
	CallbackCommand    string            `json:"CallbackCommand"`
	OperatorAccount    string            `json:"Operator_Account"`
	GroupID            string            `json:"GroupId"`
	DestinationMembers []groupMemberItem `json:"DestinationMembers"`
}

type prevInviteJoinGroupResp struct {
	ActionStatus          string   `json:"ActionStatus"`
	ErrorCode             int      `json:"ErrorCode"`
	ErrorInfo             string   `json:"ErrorInfo"`
	RefusedMembersAccount []string `json:"RefusedMembers_Account,omitempty"`
}

type stateChangeReq struct {
	CallbackCommand string `json:"CallbackCommand"`
	EventTime       int64  `json:"EventTime"`
	Info            struct {
		Action    string `json:"Action"`
		ToAccount string `json:"To_Account"`
	} `json:"Info"`
}

type callbackResp struct {
	ActionStatus string `json:"ActionStatus"`
	ErrorCode    int    `json:"ErrorCode"`
	ErrorInfo    string `json:"ErrorInfo"`
}

// TencentCallback 处理腾讯云 IM 回调。
// POST /api/v1/webhook/tencent
func TencentCallback(c *gin.Context) {
	if !authenticateTencentCallback(c) {
		return
	}
	command := resolveCallbackCommand(c)

	switch command {
	case cmdPrevFriendAdd:
		handlePrevFriendAdd(c)
	case cmdPrevFriendResponse:
		handlePrevFriendResponse(c)
	case cmdFriendAdd:
		handleFriendAdd(c)
	case cmdPrevCreateGroup:
		handlePrevCreateGroup(c)
	case cmdAfterCreateGroup:
		handleAfterCreateGroup(c)
	case cmdPrevApplyJoinGroup:
		handlePrevApplyJoinGroup(c)
	case cmdPrevInviteJoin:
		handlePrevInviteJoinGroup(c)
	case cmdC2CBeforeSendMsg, cmdGroupBeforeSendMsg:
		handleBeforeSendMsg(c)
	case cmdC2CAfterSendMsg, cmdGroupAfterSendMsg:
		handleAfterSendMsg(c)
	case cmdStateChange:
		handleStateChange(c)
	case cmdAfterMemberFieldChanged:
		handleGroupMemberRoleChanged(c)
	default:
		logger.Warnf("unsupported tencent callback command=%q path=%s", command, c.FullPath())
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
	}
}

func resolveCallbackCommand(c *gin.Context) string {
	if command := strings.TrimSpace(c.Query("CallbackCommand")); command != "" {
		return command
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Warnf("read tencent callback body failed: path=%s err=%v", c.FullPath(), err)
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) == 0 {
		return ""
	}

	var payload struct {
		CallbackCommand string `json:"CallbackCommand"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		logger.Warnf("parse tencent callback command from body failed: path=%s err=%v", c.FullPath(), err)
		return ""
	}
	return strings.TrimSpace(payload.CallbackCommand)
}

func handleStateChange(c *gin.Context) {
	var req stateChangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warnf("reject tencent state change callback: reason=invalid_request err=%v", err)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}

	action := strings.TrimSpace(req.Info.Action)
	imUserID := strings.TrimSpace(req.Info.ToAccount)
	if imUserID == "" {
		logger.Warnf("ignore tencent state change without account: action=%s", action)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
		return
	}

	if !isTencentOfflineAction(action) {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
		return
	}

	offlineAt := tencentStateEventTime(req.EventTime)
	if err := repository.NewUserRepository().TouchOfflineAtByIMUserID(imUserID, offlineAt); err != nil {
		logger.Warnf("touch last_online_at by tencent offline state failed: im_user_id=%s action=%s err=%v", imUserID, action, err)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}

	c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
}

func isTencentOfflineAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "logout", "disconnect":
		return true
	default:
		return false
	}
}

func tencentStateEventTime(eventTime int64) time.Time {
	if eventTime <= 0 {
		return time.Now()
	}
	if eventTime > 1_000_000_000_000 {
		return time.UnixMilli(eventTime)
	}
	return time.Unix(eventTime, 0)
}

func isTencentActionRateLimited(c *gin.Context, actor string, cost int64, rules []ratelimit.Rule) (bool, ratelimit.Decision) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return false, ratelimit.Decision{Allowed: true}
	}
	decision, err := ratelimit.CheckWithCost(c.Request.Context(), actor, cost, rules)
	if err == ratelimit.ErrLimited || !decision.Allowed {
		return true, decision
	}
	return false, decision
}

func handleAfterCreateGroup(c *gin.Context) {
	var req afterCreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}
	if err := grouprisk.HandleAfterCreateGroup(c.Request.Context(), grouprisk.GroupCreatedInput{
		CallbackCommand: req.CallbackCommand,
		GroupID:         req.GroupID,
		GroupName:       req.GroupName,
		GroupType:       req.GroupType,
		OwnerAccount:    req.OwnerAccount,
		OperatorAccount: req.OperatorAccount,
		CreateTimeUnix:  req.CreateTime,
		TriggerSource:   grouprisk.SourceAfterCreate,
	}); err != nil {
		logger.Warnf("handle after create group risk failed: group=%s owner=%s operator=%s err=%v", req.GroupID, req.OwnerAccount, req.OperatorAccount, err)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
		return
	}
	c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
}

func handlePrevFriendAdd(c *gin.Context) {
	var req prevFriendAddReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, prevFriendAddResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}
	fromAccount := req.fromAccount()
	logger.Infof("received tencent friend add callback: from=%s targets=%s command=%s", fromAccount, strings.Join(toAccountsForFriendItems(req.friendTargets()), ","), req.CallbackCommand)

	repo := repository.NewUserRepository()

	from, err := repo.FindByIMUserID(fromAccount)
	if err != nil {
		logger.Warnf("reject tencent friend add: from=%s reason=from_user_not_found err=%v", req.FromAccount, err)
		results := make([]friendItemResult, 0, len(req.friendTargets()))
		for _, item := range req.friendTargets() {
			results = append(results, friendItemResult{
				ToAccount:  item.ToAccount,
				ResultCode: errCodeCrossOwner,
				ResultInfo: controller_chat.ErrInfoCrossOwnerFriendAdd,
			})
		}
		c.JSON(http.StatusOK, prevFriendAddResp{ActionStatus: "OK", ErrorCode: 0, ResultItem: results})
		return
	}
	if blocked, reason, quietErr := shouldBlockOwnerQuietHours(c.Request.Context(), from.OwnerID, config.ChatQuietHoursActionAddFriend); quietErr != nil {
		c.JSON(http.StatusOK, prevFriendAddResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "evaluate quiet hours policy failed"})
		return
	} else if blocked {
		results := make([]friendItemResult, 0, len(req.friendTargets()))
		for _, item := range req.friendTargets() {
			results = append(results, friendItemResult{
				ToAccount:  item.ToAccount,
				ResultCode: errCodeAddFriendQuietHours,
				ResultInfo: reason,
			})
		}
		_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
			Action:          "add_friend",
			Source:          "im_prev_friend_add",
			CallbackCommand: req.CallbackCommand,
			ActorIMUserID:   req.FromAccount,
			TargetAccount:   strings.Join(toAccountsForFriendItems(req.friendTargets()), ","),
			Reason:          reason,
		})
		logger.Warnf("reject tencent friend add by quiet hours: from=%s targets=%s reason=%s", req.FromAccount, strings.Join(toAccountsForFriendItems(req.friendTargets()), ","), reason)
		c.JSON(http.StatusOK, prevFriendAddResp{
			ActionStatus: "OK",
			ErrorCode:    0,
			ResultItem:   results,
		})
		return
	}

	toAccounts := make([]string, 0, len(req.friendTargets()))
	for _, item := range req.friendTargets() {
		toAccounts = append(toAccounts, item.ToAccount)
	}
	toUsers, err := repo.FindByIMUserIDs(toAccounts)
	if err != nil {
		c.JSON(http.StatusOK, prevFriendAddResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}

	toUserMap := make(map[string]*model.User, len(toUsers))
	for i := range toUsers {
		u := toUsers[i]
		toUserMap[u.IMUserID] = &u
	}

	results := make([]friendItemResult, 0, len(req.friendTargets()))
	rejectedAccounts := make([]string, 0)
	for _, item := range req.friendTargets() {
		toUser, ok := toUserMap[item.ToAccount]
		if !ok || !controller_chat.CanCrossOwnerFriendAdd(from, toUser) {
			toOwnerID := uint(0)
			if toUser != nil {
				toOwnerID = toUser.OwnerID
			}
			logger.Warnf("reject tencent friend add by owner scope: from=%s from_owner=%d to=%s to_owner=%d found=%v",
				req.FromAccount, from.OwnerID, item.ToAccount, toOwnerID, ok)
			results = append(results, friendItemResult{
				ToAccount:  item.ToAccount,
				ResultCode: errCodeCrossOwner,
				ResultInfo: controller_chat.ErrInfoCrossOwnerFriendAdd,
			})
			rejectedAccounts = append(rejectedAccounts, item.ToAccount)
			continue
		}
		results = append(results, friendItemResult{
			ToAccount:  item.ToAccount,
			ResultCode: 0,
		})

	}
	if len(rejectedAccounts) > 0 {
		logger.Warnf("reject tencent friend add by owner scope: from=%s owner=%d rejected=%s",
			req.FromAccount, from.OwnerID, strings.Join(rejectedAccounts, ","))
	}

	c.JSON(http.StatusOK, prevFriendAddResp{
		ActionStatus: "OK",
		ErrorCode:    0,
		ResultItem:   results,
	})
}

// handleFriendAdd 好友关系建立后：把超管名下用户划到对方运营方。
func handleFriendAdd(c *gin.Context) {
	var req friendAddReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warnf("reject tencent friend add after callback: reason=invalid_request err=%v", err)
		c.JSON(http.StatusBadRequest, callbackResp{ActionStatus: "FAIL", ErrorCode: 1})
		return
	}
	logger.Infof("received tencent friend add after callback: pairs=%d command=%s client_cmd=%s", len(req.friendPairs()), req.CallbackCommand, req.ClientCmd)

	seen := make(map[string]struct{}, len(req.friendPairs()))
	for _, pair := range req.friendPairs() {
		initiator, peer, ok := resolveFriendAddPair(pair.InitiatorAccount, pair.FromAccount, pair.ToAccount)
		if !ok {
			continue
		}
		dedupeKey := initiator + "\x00" + peer
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}
		if _, err := controller_chat.QueueFriendOwnership(c.Request.Context(), "callback", c.GetString("owner_callback_event"), initiator, peer); err != nil {
			c.JSON(http.StatusServiceUnavailable, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "ownership task not durably saved"})
			return
		}

	}

	c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
}

func handlePrevFriendResponse(c *gin.Context) {
	var req prevFriendResponseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warnf("reject tencent friend response callback: reason=invalid_request err=%v", err)
		c.JSON(http.StatusOK, prevFriendAddResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}

	fromAccount := strings.TrimSpace(req.FromAccount)
	if fromAccount == "" {
		fromAccount = strings.TrimSpace(req.RequesterAccount)
	}
	items := req.responseItems()
	logger.Infof("received tencent friend response callback: from=%s items=%d command=%s", fromAccount, len(items), req.CallbackCommand)

	results := make([]friendItemResult, 0, len(items))
	for _, item := range items {
		toAccount := strings.TrimSpace(item.ToAccount)
		results = append(results, friendItemResult{ToAccount: toAccount, ResultCode: 0})
		if toAccount == "" || fromAccount == "" || !isFriendAgreeAction(item.ResponseAction) {
			continue
		}
		// From_Account is the accepting user; To_Account is the ORIGINAL applicant.
		repo := repository.NewUserRepository()
		applicant, err := repo.FindByIMUserID(toAccount)
		accepter, otherErr := repo.FindByIMUserID(fromAccount)
		if err != nil || otherErr != nil {
			results[len(results)-1].ResultCode = errCodeCrossOwner
			results[len(results)-1].ResultInfo = "账号归属暂时无法确认"
		} else if reason := controller_chat.FriendAddBlockedReason(applicant, accepter); reason != "" {
			results[len(results)-1].ResultCode = errCodeCrossOwner
			results[len(results)-1].ResultInfo = reason
		}

	}

	c.JSON(http.StatusOK, prevFriendAddResp{
		ActionStatus: "OK",
		ErrorCode:    0,
		ResultItem:   results,
	})
}

func applyFriendOwnerMigrate(initiator, peer string) error {
	return controller_chat.MigrateOwnerByFriendAdd(initiator, peer)
}

func (r prevFriendAddReq) fromAccount() string {
	from := strings.TrimSpace(r.FromAccount)
	if from != "" {
		return from
	}
	return strings.TrimSpace(r.RequesterAccount)
}

func (r prevFriendAddReq) friendTargets() []friendItem {
	if len(r.FriendItem) > 0 {
		return r.FriendItem
	}
	return r.AddFriendItem
}

func (r friendAddReq) friendPairs() []friendAddPair {
	if len(r.PairList) > 0 {
		return r.PairList
	}
	from := strings.TrimSpace(r.FromAccount)
	if from == "" || len(r.FriendItem) == 0 {
		return nil
	}
	pairs := make([]friendAddPair, 0, len(r.FriendItem))
	for _, item := range r.FriendItem {
		to := strings.TrimSpace(item.ToAccount)
		if to == "" {
			continue
		}
		pairs = append(pairs, friendAddPair{
			FromAccount:      from,
			ToAccount:        to,
			InitiatorAccount: from,
		})
	}
	return pairs
}

func (r prevFriendResponseReq) responseItems() []responseFriendItem {
	if len(r.ResponseFriendItem) > 0 {
		return r.ResponseFriendItem
	}
	return r.FriendItem
}

func isFriendAgreeAction(action string) bool {
	normalized := strings.ToLower(strings.TrimSpace(action))
	switch normalized {
	case "response_action_agreeandadd", "response_action_agree", "agreeandadd", "agree":
		return true
	default:
		return strings.Contains(normalized, "agree") && !strings.Contains(normalized, "reject")
	}
}

func resolveFriendAddPair(initiator, from, to string) (string, string, bool) {
	initiator = strings.TrimSpace(initiator)
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if initiator == "" {
		initiator = from
	}
	peer := to
	if initiator == to {
		peer = from
	}
	if initiator == "" || peer == "" || initiator == peer {
		return "", "", false
	}
	return initiator, peer, true
}

func handlePrevCreateGroup(c *gin.Context) {
	var req prevCreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}
	if limited, decision := isTencentActionRateLimited(c, req.OperatorAccount, 1, ratelimit.CreateGroupRules); limited {
		ratelimit.LogLimited("create_group", req.OperatorAccount, clientip.FromContext(c), decision)
		handlers.MarkAccessLogSampled(c)
		c.JSON(http.StatusOK, callbackResp{
			ActionStatus: "FAIL",
			ErrorCode:    errCodeRateLimited,
			ErrorInfo:    ratelimit.DefaultMessage,
		})
		return
	}

	repo := repository.NewUserRepository()

	operator, err := repo.FindByIMUserID(req.OperatorAccount)
	var ownerID uint
	memberAccounts := make([]string, 0, len(req.MemberList))
	for _, m := range req.MemberList {
		if m.MemberAccount != req.OperatorAccount {
			memberAccounts = append(memberAccounts, m.MemberAccount)
		}
	}

	if err == nil {
		if !operator.CanCreateGroup {
			c.JSON(http.StatusOK, callbackResp{
				ActionStatus: "FAIL",
				ErrorCode:    errCodeIMReject,
				ErrorInfo:    "当前账号未开启建群权限，请联系管理员",
			})
			return
		}
		ownerID = operator.OwnerID
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		members, membersErr := repo.FindByIMUserIDs(memberAccounts)
		if membersErr != nil {
			c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
			return
		}
		if len(members) > 0 {
			ownerID = members[0].OwnerID
			for _, m := range members[1:] {
				if m.OwnerID != ownerID {
					c.JSON(http.StatusOK, callbackResp{
						ActionStatus: "FAIL",
						ErrorCode:    errCodeCrossOwner,
						ErrorInfo:    "跨归属建群不允许",
					})
					return
				}
			}
		}
	} else {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}

	if blocked, reason, quietErr := shouldBlockOwnerQuietHours(c.Request.Context(), ownerID, config.ChatQuietHoursActionCreateGroup); quietErr != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "evaluate quiet hours policy failed"})
		return
	} else if blocked {
		_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
			Action:          "create_group",
			Source:          "im_prev_create_group",
			CallbackCommand: req.CallbackCommand,
			ActorIMUserID:   req.OperatorAccount,
			Reason:          reason,
		})
		c.JSON(http.StatusOK, callbackResp{
			ActionStatus: "FAIL",
			ErrorCode:    errCodeCreateGroupQuietHours,
			ErrorInfo:    reason,
		})
		return
	}

	if len(memberAccounts) == 0 {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
		return
	}

	members, err := repo.FindByIMUserIDs(memberAccounts)
	if err != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}

	if ownerID == 0 && len(members) > 0 {
		ownerID = members[0].OwnerID
	}

	for _, m := range members {
		if ownerID > 0 && m.OwnerID != ownerID {
			c.JSON(http.StatusOK, callbackResp{
				ActionStatus: "FAIL",
				ErrorCode:    errCodeCrossOwner,
				ErrorInfo:    "跨归属建群不允许",
			})
			return
		}
	}

	c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
}

func handlePrevApplyJoinGroup(c *gin.Context) {
	var req prevApplyJoinGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}
	logger.Infof("received tencent apply join group callback: requester=%s group=%s command=%s", req.RequestorAccount, req.GroupID, req.CallbackCommand)

	userRepo := repository.NewUserRepository()
	requestor, err := userRepo.FindByIMUserID(req.RequestorAccount)
	if err != nil {
		logger.Warnf("reject tencent apply join group: requester=%s group=%s reason=requestor_not_found", req.RequestorAccount, req.GroupID)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: errCodeIMReject, ErrorInfo: "cross-owner not allowed"})
		return
	}
	if blocked, reason, quietErr := shouldBlockOwnerQuietHours(c.Request.Context(), requestor.OwnerID, config.ChatQuietHoursActionApplyJoinGroup); quietErr != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "evaluate quiet hours policy failed"})
		return
	} else if blocked {
		_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
			Action:          "apply_join_group",
			Source:          "im_prev_apply_join_group",
			CallbackCommand: req.CallbackCommand,
			ActorIMUserID:   req.RequestorAccount,
			TargetGroupID:   req.GroupID,
			Reason:          reason,
		})
		logger.Warnf("reject tencent apply join group by quiet hours: requester=%s group=%s reason=%s", req.RequestorAccount, req.GroupID, reason)
		c.JSON(http.StatusOK, callbackResp{
			ActionStatus: "OK",
			ErrorCode:    errCodeJoinGroupQuietHours,
			ErrorInfo:    reason,
		})
		return
	}

	groupOwnerID, ok, err := findGroupOwnerID(req.GroupID)
	if err != nil {
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}
	if !ok {
		logger.Warnf("reject tencent apply join group: requester=%s group=%s reason=group_not_found", req.RequestorAccount, req.GroupID)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: errCodeIMGroupMissing, ErrorInfo: "group not found"})
		return
	}

	if requestor.OwnerID != groupOwnerID {
		logger.Warnf("reject tencent apply join group by owner scope: requester=%s requestor_owner=%d group=%s group_owner=%d", req.RequestorAccount, requestor.OwnerID, req.GroupID, groupOwnerID)
		c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: errCodeIMReject, ErrorInfo: "cross-owner not allowed"})
		return
	}

	c.JSON(http.StatusOK, callbackResp{ActionStatus: "OK", ErrorCode: 0})
}

func handlePrevInviteJoinGroup(c *gin.Context) {
	var req prevInviteJoinGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warnf("reject tencent invite join group: reason=invalid_request err=%v", err)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}
	memberAccounts := toAccountsForGroupMembers(req.DestinationMembers)
	logger.Infof("received tencent invite join group callback: operator=%s group=%s targets=%s command=%s", req.OperatorAccount, req.GroupID, strings.Join(memberAccounts, ","), req.CallbackCommand)
	if isTencentServerOperator(req.OperatorAccount) {
		handleServerOperatorInviteJoinGroup(c, req, memberAccounts)
		return
	}

	if limited, decision := isTencentActionRateLimited(c, req.OperatorAccount, int64(len(req.DestinationMembers)), ratelimit.InviteJoinGroupRules); limited {
		ratelimit.LogLimited("invite_join_group", req.OperatorAccount, clientip.FromContext(c), decision)
		handlers.MarkAccessLogSampled(c)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{
			ActionStatus:          "OK",
			ErrorCode:             errCodeRateLimited,
			ErrorInfo:             ratelimit.DefaultMessage,
			RefusedMembersAccount: memberAccounts,
		})
		return
	}

	userRepo := repository.NewUserRepository()
	operator, err := userRepo.FindByIMUserID(req.OperatorAccount)
	if err != nil {
		logger.Warnf("reject tencent invite join group: operator=%s group=%s reason=operator_not_found", req.OperatorAccount, req.GroupID)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "OK", ErrorCode: errCodeIMReject, ErrorInfo: "operator not found"})
		return
	}
	if blocked, reason, quietErr := shouldBlockOwnerQuietHours(c.Request.Context(), operator.OwnerID, config.ChatQuietHoursActionInviteJoinGroup); quietErr != nil {
		logger.Errorf("reject tencent invite join group: reason=evaluate_quiet_hours_failed operator=%s group=%s targets=%s err=%v", req.OperatorAccount, req.GroupID, strings.Join(toAccountsForGroupMembers(req.DestinationMembers), ","), quietErr)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "evaluate quiet hours policy failed"})
		return
	} else if blocked {
		_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
			Action:          "invite_join_group",
			Source:          "im_prev_invite_join_group",
			CallbackCommand: req.CallbackCommand,
			ActorIMUserID:   req.OperatorAccount,
			TargetAccount:   strings.Join(toAccountsForGroupMembers(req.DestinationMembers), ","),
			TargetGroupID:   req.GroupID,
			Reason:          reason,
		})
		logger.Warnf("reject tencent invite join group by quiet hours: operator=%s group=%s targets=%s reason=%s", req.OperatorAccount, req.GroupID, strings.Join(toAccountsForGroupMembers(req.DestinationMembers), ","), reason)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{
			ActionStatus: "OK",
			ErrorCode:    errCodeJoinGroupQuietHours,
			ErrorInfo:    reason,
		})
		return
	}

	groupOwnerID, ok, err := findGroupOwnerID(req.GroupID)
	if err != nil {
		logger.Errorf("reject tencent invite join group: reason=find_group_owner_failed operator=%s group=%s targets=%s err=%v", req.OperatorAccount, req.GroupID, strings.Join(toAccountsForGroupMembers(req.DestinationMembers), ","), err)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}
	// 建群后 AfterCreate 未落库 / 落成 owner_id=0 时，按邀请人归属补齐，避免刚建群邀请就 10101/10102
	if !ok || groupOwnerID == 0 || operator.OwnerID != groupOwnerID {
		if repaired, repairErr := ensureInviteGroupOwnerScope(req.GroupID, operator, groupOwnerID, ok); repairErr != nil {
			logger.Errorf("reject tencent invite join group: reason=ensure_group_owner_failed operator=%s group=%s err=%v", req.OperatorAccount, req.GroupID, repairErr)
			c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
			return
		} else if repaired {
			groupOwnerID = operator.OwnerID
			ok = true
			logger.Infof("ensured local group owner scope for invite: operator=%s group=%s owner=%d", req.OperatorAccount, req.GroupID, groupOwnerID)
		}
	}
	if !ok {
		logger.Warnf("reject tencent invite join group: operator=%s group=%s reason=group_not_found", req.OperatorAccount, req.GroupID)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "OK", ErrorCode: errCodeIMGroupMissing, ErrorInfo: "group not found"})
		return
	}
	if operator.OwnerID != groupOwnerID {
		logger.Warnf("reject tencent invite join group by owner scope: operator=%s group=%s group_owner=%d", req.OperatorAccount, req.GroupID, groupOwnerID)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "OK", ErrorCode: errCodeIMReject, ErrorInfo: "cross-owner not allowed"})
		return
	}

	refusedAccounts, err := filterCrossOwnerAccounts(memberAccounts, groupOwnerID)
	if err != nil {
		logger.Errorf("reject tencent invite join group: reason=filter_cross_owner_failed operator=%s group=%s group_owner=%d targets=%s err=%v", req.OperatorAccount, req.GroupID, groupOwnerID, strings.Join(memberAccounts, ","), err)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}
	if len(refusedAccounts) > 0 {
		logger.Warnf("partially reject tencent invite join group by owner scope: operator=%s group=%s group_owner=%d refused=%s", req.OperatorAccount, req.GroupID, groupOwnerID, strings.Join(refusedAccounts, ","))
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{
			ActionStatus:          "OK",
			ErrorCode:             0,
			RefusedMembersAccount: refusedAccounts,
		})
		return
	}

	c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "OK", ErrorCode: 0})
}

func handleServerOperatorInviteJoinGroup(c *gin.Context, req prevInviteJoinGroupReq, memberAccounts []string) {
	groupOwnerID, ok, err := findGroupOwnerID(req.GroupID)
	if err != nil {
		logger.Errorf("reject tencent invite join group: reason=find_group_owner_failed operator=%s group=%s targets=%s err=%v", req.OperatorAccount, req.GroupID, strings.Join(memberAccounts, ","), err)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}
	if !ok {
		logger.Warnf("reject tencent invite join group: operator=%s group=%s reason=group_not_found", req.OperatorAccount, req.GroupID)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "OK", ErrorCode: errCodeIMGroupMissing, ErrorInfo: "group not found"})
		return
	}

	if blocked, reason, quietErr := shouldBlockOwnerQuietHours(c.Request.Context(), groupOwnerID, config.ChatQuietHoursActionInviteJoinGroup); quietErr != nil {
		logger.Errorf("reject tencent invite join group: reason=evaluate_quiet_hours_failed operator=%s group=%s targets=%s err=%v", req.OperatorAccount, req.GroupID, strings.Join(memberAccounts, ","), quietErr)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "evaluate quiet hours policy failed"})
		return
	} else if blocked {
		_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
			Action:          "invite_join_group",
			Source:          "im_prev_invite_join_group",
			CallbackCommand: req.CallbackCommand,
			TargetAccount:   strings.Join(memberAccounts, ","),
			TargetGroupID:   req.GroupID,
			Reason:          reason,
		})
		logger.Warnf("reject tencent invite join group by quiet hours: operator=%s group=%s targets=%s reason=%s", req.OperatorAccount, req.GroupID, strings.Join(memberAccounts, ","), reason)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{
			ActionStatus: "OK",
			ErrorCode:    errCodeJoinGroupQuietHours,
			ErrorInfo:    reason,
		})
		return
	}

	refusedAccounts, err := filterCrossOwnerAccounts(memberAccounts, groupOwnerID)
	if err != nil {
		logger.Errorf("reject tencent invite join group: reason=filter_cross_owner_failed operator=%s group=%s group_owner=%d targets=%s err=%v", req.OperatorAccount, req.GroupID, groupOwnerID, strings.Join(memberAccounts, ","), err)
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "db error"})
		return
	}
	if len(refusedAccounts) > 0 {
		logger.Warnf("partially reject tencent invite join group by owner scope: operator=%s group=%s group_owner=%d refused=%s", req.OperatorAccount, req.GroupID, groupOwnerID, strings.Join(refusedAccounts, ","))
		c.JSON(http.StatusOK, prevInviteJoinGroupResp{
			ActionStatus:          "OK",
			ErrorCode:             0,
			RefusedMembersAccount: refusedAccounts,
		})
		return
	}

	c.JSON(http.StatusOK, prevInviteJoinGroupResp{ActionStatus: "OK", ErrorCode: 0})
}

func findGroupOwnerID(groupID string) (uint, bool, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return 0, false, nil
	}

	groupRepo := repository.NewGroupRepository()
	group, err := groupRepo.FindActiveByIMGroupID(groupID)
	if err == nil {
		return group.OwnerID, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	existingGroup, err := groupRepo.FindByIMGroupID(groupID)
	if err == nil {
		if existingGroup.Status != "active" {
			return 0, false, nil
		}
		return existingGroup.OwnerID, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	return syncMissingActiveGroup(groupID)
}

// ensureInviteGroupOwnerScope 处理「本地群缺失 / owner_id=0 / 归属错位」三类邀请前置问题。
// 优先按腾讯群主校验回写；群主资料暂不可用时，对 owner_id=0 或本地缺失的群按邀请人归属落库。
func ensureInviteGroupOwnerScope(groupID string, operator *model.User, groupOwnerID uint, groupFound bool) (bool, error) {
	if operator == nil || strings.TrimSpace(groupID) == "" {
		return false, nil
	}
	if groupFound && groupOwnerID == operator.OwnerID && groupOwnerID != 0 {
		return true, nil
	}

	if repaired, err := repairGroupOwnerIfTencentOwner(groupID, operator); err != nil {
		return false, err
	} else if repaired {
		return true, nil
	}

	// owner_id=0：建群回调未解析到归属，邀请人通常就是建群者，直接回写
	if groupFound && groupOwnerID == 0 {
		return repairGroupOwnerToOperator(groupID, operator)
	}

	// App 常见路径：先建群再邀人。本地 CreatedBy 已是邀请人，但 OwnerID 错位/竞态未修好时仍允许回写。
	if groupFound {
		if repaired, err := repairGroupOwnerIfLocalCreator(groupID, operator); err != nil {
			return false, err
		} else if repaired {
			return true, nil
		}
	}

	// 本地尚无镜像（AfterCreate 延迟/失败）：按邀请人归属补一条，避免刚建群就邀请失败
	if !groupFound {
		return ensureLocalGroupOwnedByOperator(groupID, operator)
	}
	return false, nil
}

// repairGroupOwnerIfLocalCreator 邀请人即本地建群者时，纠正群归属到邀请人运营方。
func repairGroupOwnerIfLocalCreator(groupID string, operator *model.User) (bool, error) {
	if operator == nil {
		return false, nil
	}
	groupRepo := repository.NewGroupRepository()
	group, err := groupRepo.FindByIMGroupID(strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if group.Status != "active" {
		return false, nil
	}
	if group.CreatedBy != operator.ID {
		return false, nil
	}
	if group.OwnerID == operator.OwnerID && group.OwnerID != 0 {
		return true, nil
	}
	return repairGroupOwnerToOperator(groupID, operator)
}

func repairGroupOwnerToOperator(groupID string, operator *model.User) (bool, error) {
	if operator == nil {
		return false, nil
	}
	groupRepo := repository.NewGroupRepository()
	group, err := groupRepo.FindByIMGroupID(strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if group.Status != "active" {
		return false, nil
	}
	group.OwnerID = operator.OwnerID
	group.CreatedBy = operator.ID
	if err := groupRepo.SaveMirror(context.Background(), group); err != nil {
		return false, err
	}
	return group.OwnerID == operator.OwnerID && group.CreatedBy == operator.ID, nil
}

func ensureLocalGroupOwnedByOperator(groupID string, operator *model.User) (bool, error) {
	if operator == nil || operator.OwnerID == 0 {
		return false, nil
	}
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return false, nil
	}

	if ownerID, ok, err := syncMissingActiveGroup(groupID); err != nil {
		logger.Warnf("sync missing group for invite failed, fallback to operator ownership: group=%s err=%v", groupID, err)
	} else if ok {
		if ownerID == 0 {
			return repairGroupOwnerToOperator(groupID, operator)
		}
		if ownerID == operator.OwnerID {
			return true, nil
		}
		// 腾讯侧群主归属与邀请人不一致时不强制覆盖
		return false, nil
	}

	groupName := groupID
	groupType := "Community"
	if strings.HasPrefix(groupID, "@TGS#_") || strings.Contains(groupID, "@TGS#_@TGS#") {
		groupType = "Community"
	} else if strings.HasPrefix(groupID, "ga_") {
		groupType = "Work"
	}
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		if infoMap, err := imClient.GroupGetInfo([]string{groupID}); err == nil {
			if info, ok := lookupGroupInfo(infoMap, groupID); ok {
				if name := strings.TrimSpace(info.Name); name != "" {
					groupName = name
				}
				if typ := strings.TrimSpace(info.Type); typ != "" {
					groupType = typ
				}
			}
		}
	}

	groupRepo := repository.NewGroupRepository()
	group := &model.Group{
		BizID:     idgs.Global.GroupGenerate().Int64(),
		IMGroupID: groupID,
		Name:      groupName,
		GroupType: groupType,
		OwnerID:   operator.OwnerID,
		CreatedBy: operator.ID,
		Status:    "active",
		CreatedAt: time.Now(),
	}
	if err := groupRepo.Create(group); err != nil {
		existing, findErr := groupRepo.FindByIMGroupID(groupID)
		if findErr == nil {
			if existing.Status != "active" {
				return false, nil
			}
			if existing.OwnerID == 0 {
				return repairGroupOwnerToOperator(groupID, operator)
			}
			return existing.OwnerID == operator.OwnerID, nil
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return false, findErr
		}
		return false, err
	}
	return group.OwnerID == operator.OwnerID && group.CreatedBy == operator.ID, nil
}

func lookupGroupInfo(infoMap map[string]tencentim.GroupInfo, groupID string) (tencentim.GroupInfo, bool) {
	if info, ok := infoMap[groupID]; ok {
		return info, true
	}
	for _, info := range infoMap {
		return info, true
	}
	return tencentim.GroupInfo{}, false
}

// syncMissingActiveGroup 在回调首次碰到 web 端直建群时，按腾讯 IM 实际群资料补建本地镜像。
func syncMissingActiveGroup(groupID string) (uint, bool, error) {
	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return 0, false, nil
	}

	infoMap, err := imClient.GroupGetInfo([]string{groupID})
	if err != nil {
		return 0, false, err
	}
	info, ok := lookupGroupInfo(infoMap, groupID)
	if !ok {
		return 0, false, nil
	}

	ownerAccount := strings.TrimSpace(info.OwnerAccount)
	if ownerAccount == "" {
		return 0, false, nil
	}

	userRepo := repository.NewUserRepository()
	ownerUser, err := userRepo.FindByIMUserID(ownerAccount)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}

	groupName := strings.TrimSpace(info.Name)
	if groupName == "" {
		groupName = groupID
	}
	groupType := strings.TrimSpace(info.Type)
	if groupType == "" {
		groupType = "Work"
	}

	groupRepo := repository.NewGroupRepository()
	group := &model.Group{
		BizID:     idgs.Global.GroupGenerate().Int64(),
		IMGroupID: groupID,
		Name:      groupName,
		GroupType: groupType,
		OwnerID:   ownerUser.OwnerID,
		CreatedBy: ownerUser.ID,
		Status:    "active",
		CreatedAt: time.Now(),
	}

	if err := groupRepo.Create(group); err != nil {
		existingGroup, findErr := groupRepo.FindByIMGroupID(groupID)
		if findErr == nil {
			if existingGroup.Status != "active" {
				return 0, false, nil
			}
			return existingGroup.OwnerID, true, nil
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return 0, false, findErr
		}
		return 0, false, err
	}

	return group.OwnerID, true, nil
}

func isTencentServerOperator(operatorAccount string) bool {
	adminUser := strings.TrimSpace(config.Cfg.Get().TencentIM.AdminUser)
	if adminUser == "" {
		adminUser = "administrator"
	}
	return strings.EqualFold(strings.TrimSpace(operatorAccount), adminUser)
}

// repairGroupOwnerIfTencentOwner 当操作者是腾讯 IM 群主时，将本地群 OwnerID 纠正为操作者归属。
func repairGroupOwnerIfTencentOwner(groupID string, operator *model.User) (bool, error) {
	if operator == nil {
		return false, nil
	}
	groupID = strings.TrimSpace(groupID)
	if groupID == "" || strings.TrimSpace(operator.IMUserID) == "" {
		return false, nil
	}
	groupRepo := repository.NewGroupRepository()
	group, err := groupRepo.FindByIMGroupID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if group.Status != "active" {
		return false, nil
	}
	// 本地归属为空时，不必等腾讯资料；邀请人可直接回写
	if group.OwnerID == 0 {
		return repairGroupOwnerToOperator(groupID, operator)
	}
	if group.OwnerID == operator.OwnerID && group.CreatedBy == operator.ID {
		return true, nil
	}
	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return false, nil
	}
	infoMap, err := imClient.GroupGetInfo([]string{groupID})
	if err != nil {
		return false, err
	}
	info, ok := lookupGroupInfo(infoMap, groupID)
	if !ok {
		return false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(info.OwnerAccount), strings.TrimSpace(operator.IMUserID)) {
		return false, nil
	}
	group.OwnerID = operator.OwnerID
	group.CreatedBy = operator.ID
	if err := groupRepo.SaveMirror(context.Background(), group); err != nil {
		return false, err
	}
	return group.OwnerID == operator.OwnerID && group.CreatedBy == operator.ID, nil
}

func filterCrossOwnerAccounts(accounts []string, expectedOwnerID uint) ([]string, error) {
	if len(accounts) == 0 {
		return nil, nil
	}

	userRepo := repository.NewUserRepository()
	users, err := userRepo.FindByIMUserIDs(accounts)
	if err != nil {
		return nil, err
	}

	ownerMap := make(map[string]uint, len(users))
	for _, user := range users {
		ownerMap[user.IMUserID] = user.OwnerID
	}

	refusedAccounts := make([]string, 0)
	for _, account := range accounts {
		ownerID, ok := ownerMap[account]
		if !ok || ownerID != expectedOwnerID {
			refusedAccounts = append(refusedAccounts, account)
		}
	}
	return refusedAccounts, nil
}

func toAccountsForFriendItems(items []friendItem) []string {
	accounts := make([]string, 0, len(items))
	for _, item := range items {
		if account := strings.TrimSpace(item.ToAccount); account != "" {
			accounts = append(accounts, account)
		}
	}
	return accounts
}

func hasSuccessfulFriendItem(items []friendItemResult) bool {
	for _, item := range items {
		if item.ResultCode == 0 {
			return true
		}
	}
	return false
}

func toAccountsForGroupMembers(items []groupMemberItem) []string {
	accounts := make([]string, 0, len(items))
	for _, item := range items {
		if account := strings.TrimSpace(item.MemberAccount); account != "" {
			accounts = append(accounts, account)
		}
	}
	return accounts
}

func shouldBlockOwnerQuietHours(ctx context.Context, ownerID uint, action string) (bool, string, error) {
	resolution, err := quietpolicy.ResolveChatQuietHoursPolicyForOwner(ctx, ownerID)
	if err != nil {
		return false, "", err
	}
	if !resolution.HasPolicy {
		return false, "", nil
	}

	blocked, err := config.IsChatQuietHoursActionBlockedNow(resolution.Policy, action, time.Now())
	if err != nil {
		return false, "", err
	}
	if !blocked {
		return false, "", nil
	}
	return true, config.ChatQuietHoursBlockedMessage(action, resolution.Policy), nil
}
