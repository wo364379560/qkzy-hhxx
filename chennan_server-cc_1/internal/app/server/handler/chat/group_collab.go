package handler_chat

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type createGroupTodoReq struct {
	Title            string   `json:"title" binding:"required"`
	Description      string   `json:"description"`
	ImageURLs        []string `json:"image_urls"`
	AssigneeIMUserID string   `json:"assignee_im_user_id"`
	AssigneeName     string   `json:"assignee_name"`
}

type setGroupTodoDoneReq struct {
	Done bool `json:"done"`
}

type createGroupEventReq struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	StartAt     string `json:"start_at" binding:"required"`
	EndAt       string `json:"end_at"`
	Location    string `json:"location"`
}

type createGroupLedgerReq struct {
	Type       string  `json:"type"`
	Amount     float64 `json:"amount" binding:"required"`
	Category   string  `json:"category" binding:"required"`
	Note       string  `json:"note"`
	RecordDate string  `json:"record_date"`
}

func ListGroupTodos(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	groupID := c.Param("group_id")
	items, err := controller_chat.ListGroupTodos(imUserID, groupID)
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func GetGroupTodo(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	todoID, err := strconv.ParseUint(c.Param("todo_id"), 10, 64)
	if err != nil || todoID == 0 {
		response.InvalidParams(c, "无效的待办 ID")
		return
	}
	item, err := controller_chat.GetGroupTodo(imUserID, c.Param("group_id"), uint(todoID))
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func CreateGroupTodo(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req createGroupTodoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	item, err := controller_chat.CreateGroupTodo(imUserID, controller_chat.CreateGroupTodoReq{
		GroupID:          c.Param("group_id"),
		Title:            req.Title,
		Description:      req.Description,
		ImageURLs:        req.ImageURLs,
		AssigneeIMUserID: req.AssigneeIMUserID,
		AssigneeName:     req.AssigneeName,
	})
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func SetGroupTodoDone(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	todoID, err := strconv.ParseUint(c.Param("todo_id"), 10, 64)
	if err != nil || todoID == 0 {
		response.InvalidParams(c, "无效的待办 ID")
		return
	}
	var req setGroupTodoDoneReq
	_ = c.ShouldBindJSON(&req)
	item, err := controller_chat.SetGroupTodoDone(imUserID, c.Param("group_id"), uint(todoID), req.Done)
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func DeleteGroupTodo(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	todoID, err := strconv.ParseUint(c.Param("todo_id"), 10, 64)
	if err != nil || todoID == 0 {
		response.InvalidParams(c, "无效的待办 ID")
		return
	}
	if err := controller_chat.DeleteGroupTodo(imUserID, c.Param("group_id"), uint(todoID)); err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, nil)
}

func ListGroupEvents(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	items, err := controller_chat.ListGroupEvents(imUserID, c.Param("group_id"))
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func GetGroupEvent(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	eventID, err := strconv.ParseUint(c.Param("event_id"), 10, 64)
	if err != nil || eventID == 0 {
		response.InvalidParams(c, "无效的日程 ID")
		return
	}
	item, err := controller_chat.GetGroupEvent(imUserID, c.Param("group_id"), uint(eventID))
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func CreateGroupEvent(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req createGroupEventReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	item, err := controller_chat.CreateGroupEvent(imUserID, controller_chat.CreateGroupEventReq{
		GroupID:     c.Param("group_id"),
		Title:       req.Title,
		Description: req.Description,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		Location:    req.Location,
	})
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func DeleteGroupEvent(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	eventID, err := strconv.ParseUint(c.Param("event_id"), 10, 64)
	if err != nil || eventID == 0 {
		response.InvalidParams(c, "无效的日程 ID")
		return
	}
	if err := controller_chat.DeleteGroupEvent(imUserID, c.Param("group_id"), uint(eventID)); err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, nil)
}

func ListGroupLedgers(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	result, err := controller_chat.ListGroupLedgers(imUserID, c.Param("group_id"))
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, result)
}

func GetGroupLedger(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	ledgerID, err := strconv.ParseUint(c.Param("ledger_id"), 10, 64)
	if err != nil || ledgerID == 0 {
		response.InvalidParams(c, "无效的账本 ID")
		return
	}
	item, err := controller_chat.GetGroupLedger(imUserID, c.Param("group_id"), uint(ledgerID))
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func CreateGroupLedger(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req createGroupLedgerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	item, err := controller_chat.CreateGroupLedger(imUserID, controller_chat.CreateGroupLedgerReq{
		GroupID:    c.Param("group_id"),
		Type:       req.Type,
		Amount:     req.Amount,
		Category:   req.Category,
		Note:       req.Note,
		RecordDate: req.RecordDate,
	})
	if err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, item)
}

func DeleteGroupLedger(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	ledgerID, err := strconv.ParseUint(c.Param("ledger_id"), 10, 64)
	if err != nil || ledgerID == 0 {
		response.InvalidParams(c, "无效的账本 ID")
		return
	}
	if err := controller_chat.DeleteGroupLedger(imUserID, c.Param("group_id"), uint(ledgerID)); err != nil {
		writeCollabErr(c, err)
		return
	}
	response.OK(c, nil)
}

func writeCollabErr(c *gin.Context, err error) {
	var ae controller_chat.AuthErr
	if errors.As(err, &ae) {
		response.BadRequest(c, ae.Error())
		return
	}
	c.Error(err)
	response.InternalError(c)
}
