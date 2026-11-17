package handler_admin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func setIMAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

// SearchIMUsers 搜索 IM 用户候选。
func SearchIMUsers(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.SearchIMUsers(c.GetUint("admin_id"), isAdmin, c.Query("keyword"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// SearchIMGroups 搜索 IM 群组候选。
func SearchIMGroups(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.SearchIMGroups(c.GetUint("admin_id"), isAdmin, c.Query("keyword"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

type sendC2CReq struct {
	FromUserID   string   `json:"fromUserID" binding:"required"`
	ToUserIDs    []string `json:"toUserIDs" binding:"required,min=1"`
	Content      string   `json:"content" binding:"required"`
	SyncToSender bool     `json:"syncToSender"`
}

// SendC2CMessages 批量发送单聊文本消息。
func SendC2CMessages(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var req sendC2CReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_admin.SendC2CTextMessages(c.GetUint("admin_id"), isAdmin, controller_admin.IMSendC2CReq{
		FromUserID:   req.FromUserID,
		ToUserIDs:    req.ToUserIDs,
		Content:      req.Content,
		SyncToSender: req.SyncToSender,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", nil, req.FromUserID, "IM单聊发消息")
	middleware.SetAdminAccessAction(c, "im.c2c.send", "发送C2C消息")
	setIMAuditChangeSet(c, map[string]any{
		"from_user_id":    map[string]any{"before": "", "after": req.FromUserID},
		"recipient_count": map[string]any{"before": 0, "after": len(req.ToUserIDs)},
		"sync_to_sender":  map[string]any{"before": false, "after": req.SyncToSender},
	})
	response.OK(c, result)
}

// GetC2CHistory 获取单聊历史消息。
func GetC2CHistory(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	pageSize := 50
	if raw := c.Query("pageSize"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			pageSize = value
		}
	}

	var minTime int64
	if raw := c.Query("minTime"); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
			minTime = value
		}
	}

	var maxTime int64
	if raw := c.Query("maxTime"); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
			maxTime = value
		}
	}

	result, err := controller_admin.QueryC2CHistory(c.GetUint("admin_id"), isAdmin, controller_admin.IMQueryC2CHistoryReq{
		OperatorUserID: c.Query("operatorUserID"),
		PeerUserID:     c.Query("peerUserID"),
		PageSize:       pageSize,
		MinTime:        minTime,
		MaxTime:        maxTime,
		LastMsgKey:     c.Query("lastMsgKey"),
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

type exportC2CPairReq struct {
	OperatorUserID string `json:"operatorUserID" binding:"required"`
	PeerUserID     string `json:"peerUserID" binding:"required"`
}

// ExportC2CHistoryPair 导出两个用户之间最近 30 天聊天记录。
func ExportC2CHistoryPair(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var req exportC2CPairReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	rows, err := controller_admin.ExportC2CHistoryByPair(c.GetUint("admin_id"), isAdmin, controller_admin.IMExportC2CPairReq{
		OperatorUserID: req.OperatorUserID,
		PeerUserID:     req.PeerUserID,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	filename := fmt.Sprintf("c2c_pair_export_%s_%s_%s.csv", req.OperatorUserID, req.PeerUserID, time.Now().Format("20060102_150405"))
	if err := writeIMExportCSV(c, filename, rows); err != nil {
		response.InternalError(c)
	}
}

type exportUserC2CReq struct {
	UserID string `json:"userID" binding:"required"`
}

// ExportUserC2CHistory 导出单个用户最近 30 天全部单聊记录。
func ExportUserC2CHistory(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var req exportUserC2CReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	rows, err := controller_admin.ExportUserC2CHistory(c.GetUint("admin_id"), isAdmin, controller_admin.IMExportUserC2CReq{
		UserID: req.UserID,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	filename := fmt.Sprintf("c2c_user_export_%s_%s.csv", req.UserID, time.Now().Format("20060102_150405"))
	if err := writeIMExportCSV(c, filename, rows); err != nil {
		response.InternalError(c)
	}
}

type withdrawC2CReq struct {
	FromUserID string `json:"fromUserID" binding:"required"`
	ToUserID   string `json:"toUserID" binding:"required"`
	MsgKey     string `json:"msgKey" binding:"required"`
}

// WithdrawC2CMessage 撤回单聊消息。
func WithdrawC2CMessage(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var req withdrawC2CReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_admin.WithdrawC2CMessage(c.GetUint("admin_id"), isAdmin, controller_admin.IMWithdrawC2CReq{
		FromUserID: req.FromUserID,
		ToUserID:   req.ToUserID,
		MsgKey:     req.MsgKey,
	}); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", nil, req.FromUserID, "IM单聊撤回")
	middleware.SetAdminAccessAction(c, "im.c2c.withdraw", "撤回C2C消息")
	setIMAuditChangeSet(c, map[string]any{
		"from_user_id": map[string]any{"before": "", "after": req.FromUserID},
		"to_user_id":   map[string]any{"before": "", "after": req.ToUserID},
		"msg_key":      map[string]any{"before": "", "after": req.MsgKey},
	})
	response.OK(c, nil)
}

// ListRecentConversations 获取指定用户的会话列表。
func ListRecentConversations(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.ListRecentConversations(c.GetUint("admin_id"), isAdmin, c.Query("fromUserID"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

type deleteConversationReq struct {
	FromUserID  string `json:"fromUserID" binding:"required"`
	Type        string `json:"type" binding:"required"`
	PeerID      string `json:"peerID" binding:"required"`
	ClearRamble bool   `json:"clearRamble"`
}

// DeleteConversation 删除单个会话。
func DeleteConversation(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var req deleteConversationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_admin.DeleteConversation(c.GetUint("admin_id"), isAdmin, controller_admin.IMDeleteConversationReq{
		FromUserID:  req.FromUserID,
		Type:        req.Type,
		PeerID:      req.PeerID,
		ClearRamble: req.ClearRamble,
	}); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "system_config", nil, req.PeerID, "IM会话")
	middleware.SetAdminAccessAction(c, "im.conversation.delete", "删除会话")
	setIMAuditChangeSet(c, map[string]any{
		"from_user_id": map[string]any{"before": "", "after": req.FromUserID},
		"type":         map[string]any{"before": "", "after": req.Type},
		"peer_id":      map[string]any{"before": "", "after": req.PeerID},
		"clear_ramble": map[string]any{"before": false, "after": req.ClearRamble},
	})
	response.OK(c, nil)
}

// GetGroupHistory 获取群聊历史消息。
func GetGroupHistory(c *gin.Context) {
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	pageSize := 50
	if raw := c.Query("pageSize"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			pageSize = value
		}
	}

	var lastMsgSeq int64
	if raw := c.Query("lastMsgSeq"); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
			lastMsgSeq = value
		}
	}

	result, err := controller_admin.QueryGroupHistory(c.GetUint("admin_id"), isAdmin, controller_admin.IMQueryGroupHistoryReq{
		GroupID:        c.Query("groupID"),
		OperatorUserID: c.Query("operatorUserID"),
		PageSize:       pageSize,
		LastMsgSeq:     lastMsgSeq,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

func writeIMExportCSV(c *gin.Context, filename string, rows []controller_admin.IMExportC2CRecord) error {
	var buffer bytes.Buffer
	buffer.WriteString("\uFEFF")
	writer := csv.NewWriter(&buffer)

	if err := writer.Write([]string{
		"导出模式",
		"主体用户ID",
		"主体昵称",
		"主体手机号",
		"会话对象ID",
		"会话对象昵称",
		"会话对象手机号",
		"发送方ID",
		"发送方昵称",
		"接收方ID",
		"接收方昵称",
		"消息时间",
		"消息类型",
		"内容摘要",
		"消息Key",
		"消息Seq",
		"消息Random",
		"是否撤回",
	}); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writer.Write([]string{
			safeIMCSVCell(row.ScopeMode),
			safeIMCSVCell(row.OwnerUserID),
			safeIMCSVCell(row.OwnerNickname),
			safeIMCSVCell(row.OwnerPhone),
			safeIMCSVCell(row.PeerUserID),
			safeIMCSVCell(row.PeerNickname),
			safeIMCSVCell(row.PeerPhone),
			safeIMCSVCell(row.FromUserID),
			safeIMCSVCell(row.FromNickname),
			safeIMCSVCell(row.ToUserID),
			safeIMCSVCell(row.ToNickname),
			safeIMCSVCell(formatIMExportUnixTime(row.MsgTime)),
			safeIMCSVCell(row.MsgType),
			safeIMCSVCell(row.Content),
			safeIMCSVCell(row.MsgKey),
			safeIMCSVCell(strconv.FormatInt(row.MsgSeq, 10)),
			safeIMCSVCell(strconv.FormatInt(row.MsgRandom, 10)),
			imBoolToYesNo(row.IsRecalled),
		}); err != nil {
			return err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
	return nil
}

func formatIMExportUnixTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format(time.DateTime)
}

func imBoolToYesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func safeIMCSVCell(value string) string {
	if value == "" {
		return ""
	}
	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}
