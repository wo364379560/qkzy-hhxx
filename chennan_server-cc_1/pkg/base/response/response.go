package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式。
//
//	code 0     = 成功
//	code 400   = 参数/业务错误（客户端可读消息）
//	code 401   = 未授权
//	code 403   = 无权限
//	code 500   = 服务器内部错误
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 返回成功响应（HTTP 200，code 0）。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

// BadRequest 参数或业务校验失败（HTTP 400）。
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{Code: http.StatusBadRequest, Message: msg})
}

// InvalidParams 参数绑定/格式错误，自动拼接前缀（HTTP 400）。
func InvalidParams(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, Response{Code: http.StatusBadRequest, Message: "参数错误: " + detail})
}

// Unauthorized 未登录或凭据无效（HTTP 401）。
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{Code: http.StatusUnauthorized, Message: msg})
}

// Forbidden 无操作权限（HTTP 403）。
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Response{Code: http.StatusForbidden, Message: msg})
}

// NotFound 资源不存在（HTTP 404）。
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{Code: http.StatusNotFound, Message: msg})
}

// InternalError 服务器内部错误，永远不向客户端暴露内部细节（HTTP 500）。
func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, Response{Code: http.StatusInternalServerError, Message: "服务器内部错误，请稍后重试"})
}

// InternalErrorMessage 返回带具体原因的 500 响应。
func InternalErrorMessage(c *gin.Context, msg string) {
	if msg == "" {
		InternalError(c)
		return
	}
	c.JSON(http.StatusInternalServerError, Response{Code: http.StatusInternalServerError, Message: msg})
}
