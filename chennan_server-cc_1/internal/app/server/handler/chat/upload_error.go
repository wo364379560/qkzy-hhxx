package handler_chat

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/pkg/base/response"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var diagnosticID = regexp.MustCompile(`^[a-zA-Z0-9_.:/+=-]{1,160}$`)

// Never serialize err.Error(): COS errors can contain the complete signed URL.
func avatarUploadFailure(err error, stage string) (string, gin.H) {
	details := gin.H{"stage": stage, "retryable": true}
	message := "头像存储失败，请稍后重试"
	var provider *cos.ErrorResponse
	if errors.As(err, &provider) {
		status := 0
		if provider.Response != nil {
			status = provider.Response.StatusCode
		}
		details["provider_status"] = status
		if diagnosticID.MatchString(provider.Code) {
			details["cos_code"] = provider.Code
		}
		requestID := provider.RequestID
		if requestID == "" && provider.Response != nil {
			requestID = provider.Response.Header.Get("X-Cos-Request-Id")
		}
		if diagnosticID.MatchString(requestID) {
			details["request_id"] = requestID
		}
		switch {
		case status == 451:
			message = "头像存储账户欠费或服务被冻结，请联系管理员恢复存储服务"
		case provider.Code == "SignatureDoesNotMatch" || provider.Code == "InvalidAccessKeyId" || provider.Code == "RequestTimeTooSkewed":
			message = "头像存储签名配置异常，请联系管理员核对密钥和服务器时间"
		case status == http.StatusForbidden:
			message = "头像存储权限不足，请联系管理员检查桶和密钥权限"
		}
		details["retryable"] = status == 429 || status >= 500
	} else if isTimeoutError(err) {
		message = "头像上传超时，请检查网络后重试"
	}
	return message, details
}

func writeAvatarUploadFailure(c *gin.Context, err error, stage string) {
	message, details := avatarUploadFailure(err, stage)
	c.JSON(http.StatusBadGateway, response.Response{Code: http.StatusBadGateway, Message: message, Data: details})
}
