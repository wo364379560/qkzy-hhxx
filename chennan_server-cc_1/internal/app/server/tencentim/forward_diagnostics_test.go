package tencentim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestForwardDiagnosticsPreserveTencentStageAndCode(t *testing.T) {
	for _, tc := range []struct {
		name, command, response string
		code                    int
	}{
		{"group top-level", "get_group_info", `{"ActionStatus":"FAIL","ErrorCode":10004,"ErrorInfo":"invalid GroupIdList","RequestId":"req-group-top"}`, 10004},
		{"group per-item", "get_group_info", `{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"g","ErrorCode":10004,"ErrorInfo":"invalid group field"}],"RequestId":"req-group-item"}`, 10004},
		{"group per-item rate", "get_group_info", `{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"g","ErrorCode":10023,"ErrorInfo":"rate limited"}]}`, 10023},
		{"member", "get_specified_group_member_info", `{"ActionStatus":"FAIL","ErrorCode":10004,"ErrorInfo":"invalid member parameter","RequestId":"req-member"}`, 10004},
		{"group send", "send_group_msg", `{"ActionStatus":"FAIL","ErrorCode":10004,"ErrorInfo":"invalid message parameter","RequestId":"req-send-group"}`, 10004},
		{"C2C send", "sendmsg", `{"ActionStatus":"FAIL","ErrorCode":20009,"ErrorInfo":"friend check rejected","RequestId":"req-send-c2c"}`, 20009},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if strings.HasSuffix(r.URL.Path, "/"+tc.command) {
					return forwardReply(tc.response), nil
				}
				if tc.command == "get_specified_group_member_info" && strings.HasSuffix(r.URL.Path, "/get_group_info") {
					return forwardReply(`{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"g","ErrorCode":0,"Name":"group","MuteAllMember":"Off"}]}`), nil
				}
				t.Fatalf("unexpected request command %s", r.URL.Path)
				return nil, nil
			})
			var err error
			if strings.HasPrefix(tc.command, "get_") {
				_, err = client.CheckForwardGroup(context.Background(), "g", "alice")
			} else {
				target := "GROUPg"
				if tc.command == "sendmsg" {
					target = "C2Cbob"
				}
				_, err = client.SendForward(context.Background(), "alice", target, []MessageElement{{MsgType: "TIMTextElem", MsgContent: json.RawMessage(`{"Text":"hello"}`)}}, 1, 1, nil)
			}
			var detail *DeliveryError
			if !errors.As(err, &detail) || detail.Code != tc.code || detail.Unknown || detail.Command != tc.command || detail.ErrorInfo == "" {
				t.Fatalf("lost Tencent diagnostics: %+v", err)
			}
			if !strings.HasPrefix(detail.Message, forwardStage(tc.command)+"：") || !strings.Contains(detail.Message, fmt.Sprint(tc.code)) || !strings.Contains(detail.Message, detail.ErrorInfo) {
				t.Fatalf("user-facing error lost stage/code/detail: %q", detail.Message)
			}
			wantCalls := 1
			if tc.command == "get_specified_group_member_info" {
				wantCalls = 2
			}
			if calls != wantCalls {
				t.Fatal("a failed preflight continued to another REST request")
			}
		})
	}
}

func TestForwardDiagnosticsRedactBeforeTruncating(t *testing.T) {
	client := forwardTestClient(nil)
	for _, unsafe := range []string{
		`invalid parameter; https://example.invalid/v4/send?usersig=url-user-sig&secret_key=url-secret`,
		`invalid parameter; https%3A%2F%2Fexample.invalid%2Fpath%3Fusersig%3Dencoded-signature`,
		`invalid parameter; {"UserSig":"json-signature","SecretKey":"json-secret","token":"json-token"}`,
		`invalid parameter; Authorization: Bearer bearer-signature; access_token=access-signature; refresh_token=refresh-signature`,
		`invalid parameter; appKey=test-only secret_id=secret-identity password=password-value`,
		"invalid parameter; unlabelled signing key test-only\n\x00\u202e",
	} {
		detail := client.forwardError("get_group_info", 10004, false, "腾讯拒绝请求", unsafe, "req-safe-123")
		for _, value := range []string{"example.invalid", "url-user-sig", "url-secret", "encoded-signature", "json-signature", "json-secret", "json-token", "bearer-signature", "access-signature", "refresh-signature", "test-only", "secret-identity", "password-value", "\n", "\x00", "\u202e"} {
			if strings.Contains(detail.Message, value) || strings.Contains(detail.ErrorInfo, value) {
				t.Fatalf("sensitive diagnostic escaped redaction (%q)", value)
			}
		}
		if !strings.Contains(detail.Message, "invalid parameter") || detail.RequestID != "req-safe-123" {
			t.Fatal("redaction removed the safe diagnosis or internal request correlation")
		}
		encoded, err := json.Marshal(detail)
		if err != nil || strings.Contains(string(encoded), detail.RequestID) || strings.Contains(detail.Message, detail.RequestID) {
			t.Fatal("internal Tencent request ID was exposed")
		}
	}
	long := client.forwardError("send_group_msg", 10004, false, "腾讯拒绝请求", strings.Repeat("详细说明", 100)+" usersig=tail-secret", "https://example.invalid/?token=private")
	if len([]rune(long.ErrorInfo)) > 160 || len([]rune(long.Message)) > 200 || !utf8.ValidString(long.Message) || long.RequestID != "" {
		t.Fatal("diagnostic bounds/request correlation validation failed")
	}
}

func TestForwardTransportDiagnosticsKeepUnknownAndCommand(t *testing.T) {
	for _, command := range []string{"get_group_info", "get_specified_group_member_info", "send_group_msg", "sendmsg"} {
		for _, failure := range []string{"transport", "http", "json", "marshal"} {
			t.Run(command+"/"+failure, func(t *testing.T) {
				client := forwardTestClient(func(*http.Request) (*http.Response, error) {
					switch failure {
					case "transport":
						return nil, errors.New("failed https://example.invalid/?usersig=private-signature")
					case "http":
						resp := forwardReply(`{"ErrorInfo":"usersig=private-signature"}`)
						resp.StatusCode = http.StatusBadGateway
						return resp, nil
					default:
						return forwardReply(`not json usersig=private-signature`), nil
					}
				})
				var body any = map[string]any{}
				if failure == "marshal" {
					body = make(chan int)
				}
				var response any
				err := client.forwardPost(context.Background(), "service", command, body, &response)
				var detail *DeliveryError
				if !errors.As(err, &detail) || !detail.Unknown || detail.Code != 0 || detail.Command != command || !strings.HasPrefix(detail.Message, forwardStage(command)+"：") || strings.Contains(detail.Message, "private-signature") || strings.Contains(detail.Message, "example.invalid") {
					t.Fatalf("transport error changed uncertainty or leaked details: %v", err)
				}
			})
		}
	}
}

func TestForwardDiagnosticGroupResponseStillFailsClosed(t *testing.T) {
	for _, response := range []string{
		`{"ActionStatus":"FAIL","ErrorCode":0,"ErrorInfo":"invalid response","GroupInfo":[{"GroupId":"g","ErrorCode":0,"MuteAllMember":"Off"}]}`,
		`{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"other","ErrorCode":60018,"ErrorInfo":"wrong group"}]}`,
		`{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[]}`,
	} {
		calls := 0
		client := forwardTestClient(func(*http.Request) (*http.Response, error) { calls++; return forwardReply(response), nil })
		_, err := client.CheckForwardGroup(context.Background(), "g", "alice")
		var detail *DeliveryError
		if !errors.As(err, &detail) || detail.Command != "get_group_info" || calls != 1 {
			t.Fatal("mismatched/top-level error permitted member query or send")
		}
		if strings.Contains(response, `"ActionStatus":"FAIL"`) && !detail.Unknown {
			t.Fatal("missing rejection code must remain unknown")
		}
	}
}

func TestForwardDurableDiagnosticIsBoundedAndPrivate(t *testing.T) {
	client := forwardTestClient(nil)
	err := client.forwardError("send_group_msg", 10004, false, "腾讯拒绝请求", "invalid Width https://media.invalid/a?token=secret", "req-123")
	detail, ok := ForwardFailureDetails(err)
	if !ok || detail.Command != "send_group_msg" || detail.Code != 10004 || detail.RequestID != "req-123" || !strings.Contains(detail.Info, "invalid Width") || strings.Contains(detail.Info, "secret") {
		t.Fatal("durable diagnostic lost correlation or exposed URL")
	}
	_, invalid := PrepareForwardBody(nil)
	detail, ok = ForwardFailureDetails(invalid)
	if !ok || detail.Code != 400 || detail.Command != "validate_forward_message" || !strings.Contains(detail.Info, "不能为空") || detail.RequestID != "" {
		t.Fatal("local validation cause must also survive a successful retry")
	}
	detail, ok = ForwardFailureDetails(&DeliveryError{Command: "https://invalid/?token=private", RequestID: "https://invalid/?token=private", ErrorInfo: "token=private"})
	if !ok || detail.Command != "" || detail.RequestID != "" || strings.Contains(detail.Info, "private") {
		t.Fatal("malformed correlation escaped persistence sanitization")
	}
	if _, ok := ForwardFailureDetails(errors.New("raw URL with private token")); ok {
		t.Fatal("unclassified raw error must not enter persistent diagnostics")
	}
}
