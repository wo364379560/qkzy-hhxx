package handler_chat

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestAvatarStorageErrorsKeepSafeDiagnostics(t *testing.T) {
	for _, test := range []struct {
		status     int
		code, want string
	}{{451, "UnavailableForLegalReasons", "欠费"}, {403, "AccessDenied", "权限"}, {403, "SignatureDoesNotMatch", "签名"}} {
		err := &cos.ErrorResponse{Response: &http.Response{StatusCode: test.status}, Code: test.code, RequestID: "safe-request-id", Message: "do not expose signed URL"}
		message, data := avatarUploadFailure(err, "cos_put")
		if data["provider_status"] != test.status || data["request_id"] != "safe-request-id" || data["cos_code"] != test.code {
			t.Fatal(data)
		}
		if !strings.Contains(message, test.want) || strings.Contains(message, "signed") {
			t.Fatal(message)
		}
	}
	message, _ := avatarUploadFailure(errors.New("https://host/?q-signature=secret"), "cos_put")
	if strings.Contains(message, "secret") {
		t.Fatal("credential leaked")
	}
}
