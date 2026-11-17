package controller_admin

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// charset 邀请码字符集（去掉易混淆的 O/0/I/1）。
const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateCode() (string, error) {
	const length = 8
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// InviteCodeResult 邀请码响应。
type InviteCodeResult struct {
	ID        uint    `json:"id"`
	Code      string  `json:"code"`
	OwnerID   uint    `json:"ownerID"`
	OwnerName string  `json:"ownerName"`
	MaxUses   int     `json:"maxUses"`
	UsedCount int     `json:"usedCount"`
	ExpiredAt *string `json:"expiredAt"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"createdAt"`
}

func toInviteCodeResult(ic *model.InviteCode, ownerName string) InviteCodeResult {
	var expiredAt *string
	if ic.ExpiredAt != nil {
		s := ic.ExpiredAt.Format("2006-01-02 15:04:05")
		expiredAt = &s
	}
	return InviteCodeResult{
		ID:        ic.ID,
		Code:      ic.Code,
		OwnerID:   ic.OwnerID,
		OwnerName: ownerName,
		MaxUses:   ic.MaxUses,
		UsedCount: ic.UsedCount,
		ExpiredAt: expiredAt,
		Status:    ic.Status,
		CreatedAt: ic.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListInviteCodesReq 查询参数。
type ListInviteCodesReq struct {
	Page          int
	PageSize      int
	ScopeOwnerIDs []uint
	OwnerID       uint // 0 表示不限制（管理员）
}

// ListInviteCodes 分页列出邀请码。
func ListInviteCodes(req ListInviteCodesReq) (PageResult[InviteCodeResult], error) {
	repo := repository.NewInviteCodeRepository()
	rows, total, err := repo.ListPagedWithOwner(req.Page, req.PageSize, req.ScopeOwnerIDs, req.OwnerID)
	if err != nil {
		return PageResult[InviteCodeResult]{}, err
	}
	list := make([]InviteCodeResult, 0, len(rows))
	for i := range rows {
		list = append(list, toInviteCodeResult(&rows[i].InviteCode, rows[i].OwnerName))
	}
	return PageResult[InviteCodeResult]{List: list, Total: total}, nil
}

// CreateInviteCodeReq 创建邀请码参数。
type CreateInviteCodeReq struct {
	OwnerID   uint
	MaxUses   int
	ExpiredAt *time.Time
}

// CreateInviteCode 生成单个邀请码。
func CreateInviteCode(req CreateInviteCodeReq) (*InviteCodeResult, error) {
	code, err := generateCode()
	if err != nil {
		return nil, err
	}
	if req.MaxUses < 0 {
		req.MaxUses = 0
	}
	ic := &model.InviteCode{
		Code:      code,
		OwnerID:   req.OwnerID,
		MaxUses:   req.MaxUses,
		ExpiredAt: req.ExpiredAt,
		Status:    "active",
		CreatedAt: time.Now(),
	}
	repo := repository.NewInviteCodeRepository()
	if err = repo.Create(ic); err != nil {
		return nil, err
	}
	result := toInviteCodeResult(ic, "")
	return &result, nil
}

// BatchCreateInviteCodes 批量生成邀请码。
func BatchCreateInviteCodes(actorID uint, isAdmin bool, requestedOwnerID uint, count, maxUses int, expiredAt *time.Time) ([]InviteCodeResult, error) {
	if !isAdmin {
		return nil, errors.New("助理无权管理邀请码")
	}
	if count <= 0 || count > 200 {
		return nil, errors.New("批量数量需在 1~200 之间")
	}
	if maxUses < 0 {
		maxUses = 0
	}
	ownerID, err := resolveManagedOwnerID(actorID, isAdmin, requestedOwnerID)
	if err != nil {
		return nil, err
	}
	repo := repository.NewInviteCodeRepository()
	results := make([]InviteCodeResult, 0, count)
	for i := 0; i < count; i++ {
		code, err := generateCode()
		if err != nil {
			return nil, err
		}
		ic := &model.InviteCode{
			Code:      code,
			OwnerID:   ownerID,
			MaxUses:   maxUses,
			ExpiredAt: expiredAt,
			Status:    "active",
			CreatedAt: time.Now(),
		}
		if err = repo.Create(ic); err != nil {
			// 跳过重复码，继续生成
			continue
		}
		results = append(results, toInviteCodeResult(ic, ""))
	}
	return results, nil
}

// UpdateInviteCodeReq 更新邀请码参数（maxUses=-1 跳过，status/code="" 跳过）。
type UpdateInviteCodeReq struct {
	CodeID  uint
	MaxUses int
	Status  string
	Code    string // 非空时替换码字符串
}

// UpdateInviteCode 更新邀请码的 maxUses、启用状态，以及可选的码字符串。
func UpdateInviteCode(actorID uint, isAdmin bool, req UpdateInviteCodeReq) (*InviteCodeResult, error) {
	if !isAdmin {
		return nil, errors.New("助理无权管理邀请码")
	}
	repo := repository.NewInviteCodeRepository()
	ic, err := findManagedInviteCode(actorID, isAdmin, req.CodeID)
	if err != nil {
		return nil, err
	}
	// 更新码字符串（手动设置）
	if req.Code != "" {
		if err = repo.UpdateCode(ic.ID, req.Code); err != nil {
			return nil, err
		}
	}
	fields := map[string]interface{}{}
	if req.MaxUses >= 0 {
		fields["max_uses"] = req.MaxUses
	}
	if req.Status == "active" || req.Status == "expired" {
		fields["status"] = req.Status
	}
	if len(fields) > 0 {
		if err = repo.UpdateFields(ic.ID, fields); err != nil {
			return nil, err
		}
	}
	ic, err = repo.FindByID(req.CodeID)
	if err != nil {
		return nil, err
	}
	result := toInviteCodeResult(ic, "")
	return &result, nil
}

// RegenerateInviteCode 为指定邀请码重新生成码字符串。
func RegenerateInviteCode(actorID uint, isAdmin bool, codeID uint) (*InviteCodeResult, error) {
	if !isAdmin {
		return nil, errors.New("助理无权管理邀请码")
	}
	repo := repository.NewInviteCodeRepository()
	if _, err := findManagedInviteCode(actorID, isAdmin, codeID); err != nil {
		return nil, err
	}
	newCode, err := generateCode()
	if err != nil {
		return nil, err
	}
	if err = repo.UpdateCode(codeID, newCode); err != nil {
		return nil, err
	}
	ic, err := repo.FindByID(codeID)
	if err != nil {
		return nil, err
	}
	result := toInviteCodeResult(ic, "")
	return &result, nil
}

// DeleteInviteCode 删除邀请码（仅允许删除自己的或管理员全量删除）。
func DeleteInviteCode(adminID uint, isAdmin bool, codeID uint) error {
	if !isAdmin {
		return errors.New("助理无权管理邀请码")
	}
	repo := repository.NewInviteCodeRepository()
	if _, err := findManagedInviteCode(adminID, isAdmin, codeID); err != nil {
		if err.Error() == "无权操作此邀请码" {
			return errors.New("无权删除此邀请码")
		}
		return err
	}
	return repo.Delete(codeID)
}
