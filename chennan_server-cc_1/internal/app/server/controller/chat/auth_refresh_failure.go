package controller_chat

import "errors"

// RefreshFailure is additive response metadata. The HTTP/code 401 convention
// stays compatible with existing clients; temporary infrastructure failures
// must never become a definitive authentication rejection.
type RefreshFailure struct {
	ErrorCode     string `json:"error_code"`
	SessionStatus string `json:"session_status"`
	Retryable     bool   `json:"retryable"`
	Stage         string `json:"-"`
}

type refreshFailureError struct {
	details RefreshFailure
	cause   error
}

func (e *refreshFailureError) Error() string { return e.cause.Error() }
func (e *refreshFailureError) Unwrap() error { return e.cause }

func refreshRejected(code string) error {
	return &refreshFailureError{details: RefreshFailure{
		ErrorCode: code, SessionStatus: "invalid",
	}, cause: AuthErr(refreshSessionInvalidMessage)}
}

func classifyRefreshFailure(err error, stage string) error {
	var known *refreshFailureError
	if errors.As(err, &known) {
		copy := *known
		copy.details.Stage = stage
		return &copy
	}
	details := RefreshFailure{ErrorCode: "SESSION_TEMPORARY", SessionStatus: "temporary", Retryable: true, Stage: stage}
	var authErr AuthErr
	if errors.As(err, &authErr) {
		details.SessionStatus, details.Retryable = "invalid", false
		details.ErrorCode = "SESSION_INVALID"
		switch stage {
		case "rotation_recovery":
			details.ErrorCode = "SESSION_ROTATED"
		case "device_binding":
			details.ErrorCode = "SESSION_DEVICE_MISMATCH"
		case "account_policy":
			details.ErrorCode = "SESSION_POLICY_DENIED"
		}
	}
	return &refreshFailureError{details: details, cause: err}
}

func DescribeRefreshFailure(err error) RefreshFailure {
	var known *refreshFailureError
	if errors.As(err, &known) {
		return known.details
	}
	classified := classifyRefreshFailure(err, "refresh")
	return classified.(*refreshFailureError).details
}
