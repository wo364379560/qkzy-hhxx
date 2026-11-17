package controller_chat

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"gorm.io/gorm"
)

type GroupTodoItem struct {
	ID               uint     `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	ImageURLs        []string `json:"image_urls"`
	Status           string   `json:"status"`
	CreatorIMUserID  string   `json:"creator_im_user_id"`
	CreatorName      string   `json:"creator_name"`
	AssigneeIMUserID string   `json:"assignee_im_user_id"`
	AssigneeName     string   `json:"assignee_name"`
	DoneAt           string   `json:"done_at,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type GroupEventItem struct {
	ID              uint   `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	StartAt         string `json:"start_at"`
	EndAt           string `json:"end_at,omitempty"`
	Location        string `json:"location"`
	CreatorIMUserID string `json:"creator_im_user_id"`
	CreatorName     string `json:"creator_name"`
	CreatedAt       string `json:"created_at"`
}

type GroupLedgerItem struct {
	ID              uint    `json:"id"`
	Type            string  `json:"type"`
	Amount          float64 `json:"amount"`
	Category        string  `json:"category"`
	Note            string  `json:"note"`
	RecordDate      string  `json:"record_date"`
	CreatorIMUserID string  `json:"creator_im_user_id"`
	CreatorName     string  `json:"creator_name"`
	CreatedAt       string  `json:"created_at"`
}

type GroupLedgerListResult struct {
	Items        []GroupLedgerItem `json:"items"`
	ExpenseTotal float64           `json:"expense_total"`
	IncomeTotal  float64           `json:"income_total"`
}

func ensureGroupMember(imUserID, groupID string) error {
	groupID = strings.TrimSpace(groupID)
	imUserID = strings.TrimSpace(imUserID)
	if groupID == "" {
		return AuthErr("group_id 不能为空")
	}
	if imUserID == "" {
		return AuthErr("你不是该群成员")
	}
	role, err := tencentim.NewClient().GroupGetRole(groupID, imUserID)
	if err != nil {
		return err
	}
	switch role {
	case groupRoleOwner, groupRoleAdmin, groupRoleMember:
		return nil
	}
	return AuthErr("你不是该群成员")
}

func ensureGroupOwnerOrAdmin(imUserID, groupID string) error {
	groupID = strings.TrimSpace(groupID)
	imUserID = strings.TrimSpace(imUserID)
	if groupID == "" {
		return AuthErr("group_id 不能为空")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return err
	}
	permission, err := canManageRegularMembersFromDB(groupID, imUserID)
	if err != nil {
		return err
	}
	if permission == nil {
		return AuthErr("仅群主或管理员可操作")
	}
	return nil
}

func resolveDisplayName(imUserID string) string {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil || user == nil {
		return imUserID
	}
	if strings.TrimSpace(user.Nickname) != "" {
		return user.Nickname
	}
	return imUserID
}

func ListGroupTodos(imUserID, groupID string) ([]GroupTodoItem, error) {
	if err := ensureGroupMember(imUserID, groupID); err != nil {
		return nil, err
	}
	rows, err := repository.NewGroupCollabRepository().ListTodos(strings.TrimSpace(groupID))
	if err != nil {
		return nil, err
	}
	items := make([]GroupTodoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toGroupTodoItem(row))
	}
	return items, nil
}

type CreateGroupTodoReq struct {
	GroupID          string
	Title            string
	Description      string
	ImageURLs        []string
	AssigneeIMUserID string
	AssigneeName     string
}

func GetGroupTodo(imUserID, groupID string, todoID uint) (*GroupTodoItem, error) {
	if err := ensureGroupMember(imUserID, groupID); err != nil {
		return nil, err
	}
	row, err := repository.NewGroupCollabRepository().FindTodo(todoID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AuthErr("待办不存在")
		}
		return nil, err
	}
	if row.Status == model.GroupCollabStatusDeleted {
		return nil, AuthErr("待办已删除")
	}
	item := toGroupTodoItem(*row)
	return &item, nil
}

func CreateGroupTodo(imUserID string, req CreateGroupTodoReq) (*GroupTodoItem, error) {
	if err := ensureGroupOwnerOrAdmin(imUserID, req.GroupID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, AuthErr("待办标题不能为空")
	}
	if utf8.RuneCountInString(title) > 100 {
		return nil, AuthErr("待办标题不能超过100字")
	}
	desc := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(desc) > 1000 {
		return nil, AuthErr("待办描述不能超过1000字")
	}
	images := normalizeImageURLs(req.ImageURLs)
	row := &model.GroupTodo{
		IMGroupID:        strings.TrimSpace(req.GroupID),
		Title:            title,
		Description:      desc,
		ImageURLs:        encodeImageURLs(images),
		Status:           model.GroupTodoStatusOpen,
		CreatorIMUserID:  imUserID,
		CreatorName:      resolveDisplayName(imUserID),
		AssigneeIMUserID: strings.TrimSpace(req.AssigneeIMUserID),
		AssigneeName:     strings.TrimSpace(req.AssigneeName),
	}
	if row.AssigneeIMUserID != "" && row.AssigneeName == "" {
		row.AssigneeName = resolveDisplayName(row.AssigneeIMUserID)
	}
	if err := repository.NewGroupCollabRepository().CreateTodo(row); err != nil {
		return nil, err
	}
	item := toGroupTodoItem(*row)
	return &item, nil
}

func SetGroupTodoDone(imUserID string, groupID string, todoID uint, done bool) (*GroupTodoItem, error) {
	if err := ensureGroupOwnerOrAdmin(imUserID, groupID); err != nil {
		return nil, err
	}
	repo := repository.NewGroupCollabRepository()
	row, err := repo.FindTodo(todoID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AuthErr("待办不存在")
		}
		return nil, err
	}
	now := time.Now()
	fields := map[string]any{}
	if done {
		fields["status"] = model.GroupTodoStatusDone
		fields["done_at"] = now
	} else {
		fields["status"] = model.GroupTodoStatusOpen
		fields["done_at"] = nil
	}
	if err := repo.UpdateTodo(row.ID, fields); err != nil {
		return nil, err
	}
	row, err = repo.FindTodo(todoID, strings.TrimSpace(groupID))
	if err != nil {
		return nil, err
	}
	item := toGroupTodoItem(*row)
	return &item, nil
}

func DeleteGroupTodo(imUserID, groupID string, todoID uint) error {
	if err := ensureGroupOwnerOrAdmin(imUserID, groupID); err != nil {
		return err
	}
	repo := repository.NewGroupCollabRepository()
	row, err := repo.FindTodo(todoID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthErr("待办不存在")
		}
		return err
	}
	return repo.UpdateTodo(row.ID, map[string]any{"status": model.GroupCollabStatusDeleted})
}

type CreateGroupEventReq struct {
	GroupID     string
	Title       string
	Description string
	StartAt     string
	EndAt       string
	Location    string
}

func ListGroupEvents(imUserID, groupID string) ([]GroupEventItem, error) {
	if err := ensureGroupMember(imUserID, groupID); err != nil {
		return nil, err
	}
	from := time.Now().AddDate(0, 0, -1)
	rows, err := repository.NewGroupCollabRepository().ListEvents(strings.TrimSpace(groupID), from)
	if err != nil {
		return nil, err
	}
	items := make([]GroupEventItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toGroupEventItem(row))
	}
	return items, nil
}

func GetGroupEvent(imUserID, groupID string, eventID uint) (*GroupEventItem, error) {
	if err := ensureGroupMember(imUserID, groupID); err != nil {
		return nil, err
	}
	row, err := repository.NewGroupCollabRepository().FindEvent(eventID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AuthErr("日程不存在")
		}
		return nil, err
	}
	if row.Status == model.GroupCollabStatusDeleted {
		return nil, AuthErr("日程已删除")
	}
	item := toGroupEventItem(*row)
	return &item, nil
}

func CreateGroupEvent(imUserID string, req CreateGroupEventReq) (*GroupEventItem, error) {
	if err := ensureGroupOwnerOrAdmin(imUserID, req.GroupID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, AuthErr("日程标题不能为空")
	}
	desc := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(desc) > 500 {
		return nil, AuthErr("日程描述不能超过500字")
	}
	startAt, ok := parseRFC3339(req.StartAt)
	if !ok {
		return nil, AuthErr("开始时间格式无效")
	}
	var endAt *time.Time
	if strings.TrimSpace(req.EndAt) != "" {
		parsed, ok := parseRFC3339(req.EndAt)
		if !ok {
			return nil, AuthErr("结束时间格式无效")
		}
		endAt = &parsed
	}
	row := &model.GroupSharedEvent{
		IMGroupID:       strings.TrimSpace(req.GroupID),
		Title:           title,
		Description:     desc,
		StartAt:         startAt,
		EndAt:           endAt,
		Location:        strings.TrimSpace(req.Location),
		Status:          model.GroupCollabStatusActive,
		CreatorIMUserID: imUserID,
		CreatorName:     resolveDisplayName(imUserID),
	}
	if err := repository.NewGroupCollabRepository().CreateEvent(row); err != nil {
		return nil, err
	}
	item := toGroupEventItem(*row)
	return &item, nil
}

func DeleteGroupEvent(imUserID, groupID string, eventID uint) error {
	if err := ensureGroupOwnerOrAdmin(imUserID, groupID); err != nil {
		return err
	}
	repo := repository.NewGroupCollabRepository()
	row, err := repo.FindEvent(eventID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthErr("日程不存在")
		}
		return err
	}
	return repo.SoftDeleteEvent(row.ID)
}

type CreateGroupLedgerReq struct {
	GroupID    string
	Type       string
	Amount     float64
	Category   string
	Note       string
	RecordDate string
}

func ListGroupLedgers(imUserID, groupID string) (*GroupLedgerListResult, error) {
	if err := ensureGroupMember(imUserID, groupID); err != nil {
		return nil, err
	}
	gid := strings.TrimSpace(groupID)
	repo := repository.NewGroupCollabRepository()
	rows, err := repo.ListLedgers(gid, 100)
	if err != nil {
		return nil, err
	}
	expense, income, err := repo.SumLedgers(gid)
	if err != nil {
		return nil, err
	}
	items := make([]GroupLedgerItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toGroupLedgerItem(row))
	}
	return &GroupLedgerListResult{
		Items:        items,
		ExpenseTotal: expense,
		IncomeTotal:  income,
	}, nil
}

func GetGroupLedger(imUserID, groupID string, ledgerID uint) (*GroupLedgerItem, error) {
	if err := ensureGroupMember(imUserID, groupID); err != nil {
		return nil, err
	}
	row, err := repository.NewGroupCollabRepository().FindLedger(ledgerID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AuthErr("账本记录不存在")
		}
		return nil, err
	}
	if row.Status == model.GroupCollabStatusDeleted {
		return nil, AuthErr("账本记录已删除")
	}
	item := toGroupLedgerItem(*row)
	return &item, nil
}

func CreateGroupLedger(imUserID string, req CreateGroupLedgerReq) (*GroupLedgerItem, error) {
	if err := ensureGroupOwnerOrAdmin(imUserID, req.GroupID); err != nil {
		return nil, err
	}
	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = model.GroupLedgerTypeExpense
	}
	if typ != model.GroupLedgerTypeExpense && typ != model.GroupLedgerTypeIncome {
		return nil, AuthErr("账本类型无效")
	}
	if req.Amount <= 0 {
		return nil, AuthErr("金额必须大于 0")
	}
	if req.Amount > 1_000_000_000 {
		return nil, AuthErr("金额过大")
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		return nil, AuthErr("分类不能为空")
	}
	if utf8.RuneCountInString(category) > 32 {
		return nil, AuthErr("分类不能超过32字")
	}
	note := strings.TrimSpace(req.Note)
	if utf8.RuneCountInString(note) > 200 {
		return nil, AuthErr("备注不能超过200字")
	}
	recordDate := strings.TrimSpace(req.RecordDate)
	if recordDate == "" {
		recordDate = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", recordDate); err != nil {
		return nil, AuthErr("日期格式无效，需为 yyyy-MM-dd")
	}
	row := &model.GroupSharedLedger{
		IMGroupID:       strings.TrimSpace(req.GroupID),
		Type:            typ,
		Amount:          req.Amount,
		Category:        category,
		Note:            note,
		RecordDate:      recordDate,
		Status:          model.GroupCollabStatusActive,
		CreatorIMUserID: imUserID,
		CreatorName:     resolveDisplayName(imUserID),
	}
	if err := repository.NewGroupCollabRepository().CreateLedger(row); err != nil {
		return nil, err
	}
	item := toGroupLedgerItem(*row)
	return &item, nil
}

func DeleteGroupLedger(imUserID, groupID string, ledgerID uint) error {
	if err := ensureGroupOwnerOrAdmin(imUserID, groupID); err != nil {
		return err
	}
	repo := repository.NewGroupCollabRepository()
	if _, err := repo.FindLedger(ledgerID, strings.TrimSpace(groupID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthErr("账本记录不存在")
		}
		return err
	}
	return repo.SoftDeleteLedger(ledgerID)
}

func toGroupLedgerItem(row model.GroupSharedLedger) GroupLedgerItem {
	return GroupLedgerItem{
		ID:              row.ID,
		Type:            row.Type,
		Amount:          row.Amount,
		Category:        row.Category,
		Note:            row.Note,
		RecordDate:      row.RecordDate,
		CreatorIMUserID: row.CreatorIMUserID,
		CreatorName:     row.CreatorName,
		CreatedAt:       formatRFC3339(row.CreatedAt),
	}
}

func toGroupTodoItem(row model.GroupTodo) GroupTodoItem {
	images := decodeImageURLs(row.ImageURLs)
	if images == nil {
		images = []string{}
	}
	item := GroupTodoItem{
		ID:               row.ID,
		Title:            row.Title,
		Description:      row.Description,
		ImageURLs:        images,
		Status:           row.Status,
		CreatorIMUserID:  row.CreatorIMUserID,
		CreatorName:      row.CreatorName,
		AssigneeIMUserID: row.AssigneeIMUserID,
		AssigneeName:     row.AssigneeName,
		CreatedAt:        formatRFC3339(row.CreatedAt),
		UpdatedAt:        formatRFC3339(row.UpdatedAt),
	}
	if row.DoneAt != nil {
		item.DoneAt = formatRFC3339(*row.DoneAt)
	}
	return item
}

func normalizeImageURLs(urls []string) []string {
	out := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, raw := range urls {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func encodeImageURLs(urls []string) string {
	if len(urls) == 0 {
		return "[]"
	}
	b, err := json.Marshal(urls)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func decodeImageURLs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return nil
	}
	return normalizeImageURLs(urls)
}

func toGroupEventItem(row model.GroupSharedEvent) GroupEventItem {
	item := GroupEventItem{
		ID:              row.ID,
		Title:           row.Title,
		Description:     row.Description,
		StartAt:         formatRFC3339(row.StartAt),
		Location:        row.Location,
		CreatorIMUserID: row.CreatorIMUserID,
		CreatorName:     row.CreatorName,
		CreatedAt:       formatRFC3339(row.CreatedAt),
	}
	if row.EndAt != nil {
		item.EndAt = formatRFC3339(*row.EndAt)
	}
	return item
}
