package controller_admin

import "time"

// PageResult 分页结果，与前端 PageResult<T> 对应。
type PageResult[T any] struct {
	List  []T   `json:"list"`
	Total int64 `json:"total"`
}

// BatchResult 批量操作单条结果。
type BatchResult struct {
	Index   int    `json:"index"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateUserReq 更新用户资料请求。
type UpdateUserReq struct {
	WananID        *string    `json:"wanan_id"`
	Nickname       *string    `json:"nickname"`
	UserTag        *string    `json:"user_tag"`
	AvatarURL      *string    `json:"avatar_url"`
	SelfSignature  *string    `json:"self_signature"`
	Gender         *int8      `json:"gender"`
	Birthday       *time.Time `json:"birthday"`
	CanCreateGroup     *bool      `json:"can_create_group"`
	GloballySearchable *bool      `json:"globally_searchable"`
}

// ResetUserPasswordReq 重置用户密码请求。
type ResetUserPasswordReq struct {
	Mode        string  `json:"mode" binding:"required,oneof=set default"`
	NewPassword *string `json:"new_password"`
	MustChange  bool    `json:"must_change"`
}

// UpdateUserPhoneReq 修改用户手机号请求。
type UpdateUserPhoneReq struct {
	NewPhone string `json:"new_phone" binding:"required"`
}

// SaveUserIPWhitelistReq 保存用户登录白名单请求。
type SaveUserIPWhitelistReq struct {
	Enabled bool    `json:"enabled"`
	IP      *string `json:"ip"`
}

// UserIPWhitelistResult 是用户登录白名单保存结果。
type UserIPWhitelistResult struct {
	Enabled      bool   `json:"enabled"`
	IP           string `json:"ip"`
	ForcedLogout bool   `json:"forced_logout"`
}

// BatchGetUserStatusReq 批量获取用户在线状态请求。
type BatchGetUserStatusReq struct {
	IMUserIDs []string `json:"im_user_ids" binding:"required"`
}
