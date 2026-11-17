package tencentim

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	forwardDiagnosticURL        = regexp.MustCompile(`(?i)(?:https?|wss?)://[^\s<>"']+|https?%3a%2f%2f[^\s<>"']+`)
	forwardDiagnosticAuth       = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[a-z0-9+/._~=-]+`)
	forwardDiagnosticCredential = regexp.MustCompile(`(?i)["']?\b(usersig|user[_-]?sig|(?:access|refresh|session)[_-]?token|authorization|secret(?:[_-]?(?:id|key))?|(?:sdk)?app[_-]?key|api[_-]?key|q-signature|q-ak|password|passwd|token|credential|signature)["']?\s*(?:[:=]\s*|\s+)(?:"[^"]*"|'[^']*'|[^\s,;&\]}]+)`)
	forwardDiagnosticRequestID  = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,128}$`)
)

func forwardStage(command string) string {
	switch command {
	case "validate_forward_message":
		return "消息参数校验"
	case "get_group_info":
		return "群资料查询"
	case "get_specified_group_member_info":
		return "群成员检查"
	case "send_group_msg":
		return "群消息发送"
	case "sendmsg":
		return "单聊消息发送"
	default:
		return "腾讯请求"
	}
}

// ForwardFailureDiagnostic contains only bounded, sanitized correlation data.
// It is stored with the delivery so retries/restarts do not erase the cause.
type ForwardFailureDiagnostic struct {
	Command, Info, RequestID string
	Code                     int
}

func ForwardFailureDetails(err error) (ForwardFailureDiagnostic, bool) {
	var failure *DeliveryError
	if !errors.As(err, &failure) {
		return ForwardFailureDiagnostic{}, false
	}
	command := failure.Command
	if forwardStage(command) == "腾讯请求" {
		command = ""
	}
	requestID := failure.RequestID
	if !forwardDiagnosticRequestID.MatchString(requestID) {
		requestID = ""
	}
	info := failure.ErrorInfo
	if info == "" {
		info = failure.Message
	}
	return ForwardFailureDiagnostic{Command: command, Code: failure.Code,
		Info: sanitizeForwardDiagnostic(info, ""), RequestID: requestID}, true
}

func limitForwardDiagnostic(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit-1]) + "…"
	}
	return value
}

// Tencent errors can quote request URLs or credential fields. Redact before
// truncation so a cut-off URL/key does not evade detection; never expose raw
// transport errors, response bodies, or the signing key through progress rows.
func sanitizeForwardDiagnostic(value, signingKey string) string {
	if signingKey != "" {
		value = strings.ReplaceAll(value, signingKey, "[REDACTED]")
	}
	value = forwardDiagnosticURL.ReplaceAllString(value, "[URL REDACTED]")
	value = forwardDiagnosticAuth.ReplaceAllString(value, "[AUTH REDACTED]")
	value = forwardDiagnosticCredential.ReplaceAllString(value, "$1=[REDACTED]")
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	return limitForwardDiagnostic(value, 160)
}

func (c *Client) forwardError(command string, code int, unknown bool, message, info, requestID string) *DeliveryError {
	info = sanitizeForwardDiagnostic(info, c.appKey)
	message = forwardStage(command) + "：" + sanitizeForwardDiagnostic(message, c.appKey)
	if code != 0 {
		message += fmt.Sprintf("（%d）", code)
	}
	if info != "" {
		message += "：" + info
	}
	if !forwardDiagnosticRequestID.MatchString(requestID) || c.appKey != "" && strings.Contains(requestID, c.appKey) {
		requestID = ""
	}
	return &DeliveryError{Code: code, Unknown: unknown, Command: command, Message: limitForwardDiagnostic(message, 200), ErrorInfo: info, RequestID: requestID}
}

func (c *Client) forwardGateError(command string, err error) error {
	var delivery *DeliveryError
	if errors.As(err, &delivery) {
		return c.forwardError(command, delivery.Code, delivery.Unknown, delivery.Message, delivery.ErrorInfo, delivery.RequestID)
	}
	result := c.forwardError(command, 0, true, "发送前检查等待未完成", "", "")
	result.cause = err // Preserve cancellation without returning a raw error string.
	return result
}
