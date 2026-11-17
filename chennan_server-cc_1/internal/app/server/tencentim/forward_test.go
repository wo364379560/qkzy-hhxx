package tencentim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func forwardTestClient(fn func(*http.Request) (*http.Response, error)) *Client {
	return &Client{sdkAppID: 1, appKey: "test-only", adminUser: "admin", http: &http.Client{Transport: forwardTestTransport(fn)}}
}

type forwardTestTransport func(*http.Request) (*http.Response, error)

func (f forwardTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func forwardReply(s string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(s))}
}

func TestForwardRESTPermissionsAndDedup(t *testing.T) {
	for _, target := range []string{"C2Cbob", "GROUPgroup"} {
		t.Run(target, func(t *testing.T) {
			var initial string
			client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
				b, _ := io.ReadAll(r.Body)
				if initial != "" && initial != string(b) {
					t.Fatal("retry changed deduplication body")
				}
				initial = string(b)
				var body map[string]any
				_ = json.Unmarshal(b, &body)
				if body["From_Account"] != "alice" || body["ForbidCallbackControl"] != nil || body["CloudCustomData"] != nil {
					t.Fatal("sender/callbacks must remain enforced")
				}
				push := body["OfflinePushInfo"].(map[string]any)
				android := push["AndroidInfo"].(map[string]any)
				if android["XiaoMiChannelID"] != xiaoMiPushChannelID || android["XiaoMiTemplateId"] != xiaoMiChatTemplateID {
					t.Fatal("push templates missing")
				}
				if strings.HasPrefix(target, "C2C") {
					if r.URL.Path != "/v4/openim/sendmsg" || body["SyncOtherMachine"] != float64(1) || body["MsgRandom"] != float64(123) || body["MsgSeq"] != float64(2) {
						t.Fatal("C2C synchronization/dedup mismatch")
					}
					for _, flag := range []string{"WithFriendCheck", "WithBlackListCheck", "WithSpeakingCheck", "WithMuteNotifications"} {
						if !strings.Contains(string(b), flag) {
							t.Fatalf("missing %s", flag)
						}
					}
				} else if r.URL.Path != "/v4/group_open_http_svc/send_group_msg" || body["Random"] != float64(123) || body["SendMsgControl"] != nil {
					t.Fatal("group must preserve unread/history/moderation")
				}
				return forwardReply(`{"ActionStatus":"OK","ErrorCode":0,"MsgId":"id","MsgSeq":5}`), nil
			})
			body := []MessageElement{{MsgType: "TIMTextElem", MsgContent: json.RawMessage(`{"Text":"hello"}`)}}
			for i := 0; i < 2; i++ {
				if _, err := client.SendForward(context.Background(), "alice", target, body, 123, 2, NewC2CTextOfflinePushInfo("alice", "hello", "", "alice", "hello")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestForwardUnknownAndDroppedResponses(t *testing.T) {
	for _, tc := range []struct {
		body    string
		unknown bool
		code    int
	}{
		{`{}`, true, 0}, {`not json`, true, 0}, {`{"ActionStatus":"FAIL","ErrorCode":10017}`, false, 10017},
		{`{"ActionStatus":"OK","ErrorCode":0,"MsgDropReason":"MsgFreqCtrl"}`, false, 10023},
		{`{"ActionStatus":"OK","ErrorCode":0,"MsgDropReason":"ContentRejected"}`, false, 80001},
	} {
		t.Run(tc.body, func(t *testing.T) {
			client := forwardTestClient(func(*http.Request) (*http.Response, error) { return forwardReply(tc.body), nil })
			_, err := client.SendForward(context.Background(), "alice", "GROUPg", []MessageElement{{MsgType: "TIMTextElem", MsgContent: json.RawMessage(`{"Text":"hello"}`)}}, 1, 1, nil)
			var e *DeliveryError
			if !errors.As(err, &e) || e.Unknown != tc.unknown || e.Code != tc.code {
				t.Fatalf("wrong outcome: %v", err)
			}
		})
	}
}

func TestForwardGroupCurrentMemberAndMute(t *testing.T) {
	for _, tc := range []struct {
		name, role, all string
		until           int64
		allowed         bool
	}{
		{"normal", "Member", "Off", 0, true}, {"all mute", "Member", "On", 0, false}, {"owner", "Owner", "On", 0, true},
		{"personal mute", "Admin", "Off", time.Now().Unix() + 3600, false}, {"left group", "NotMember", "Off", 0, false},
		{"unknown role", "Other", "Off", 0, false}, {"missing all-mute", "Member", "", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/get_group_info") {
					var request map[string]any
					_ = json.NewDecoder(r.Body).Decode(&request)
					if request["ResponseFilter"] == nil {
						t.Fatal("group fields require ResponseFilter")
					}
					return forwardReply(fmt.Sprintf(`{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"g","ErrorCode":0,"Name":"group","MuteAllMember":%q}]}`, tc.all)), nil
				}
				if !strings.HasSuffix(r.URL.Path, "/get_specified_group_member_info") {
					t.Fatalf("wrong endpoint %s", r.URL.Path)
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["Member_List_Account"].([]any)[0] != "alice" {
					t.Fatal("must check authenticated sender")
				}
				return forwardReply(fmt.Sprintf(`{"ActionStatus":"OK","ErrorCode":0,"MemberList":[{"Member_Account":"alice","Role":%q,"MuteUntil":%d}]}`, tc.role, tc.until)), nil
			})
			_, err := client.CheckForwardGroup(context.Background(), "g", "alice")
			if (err == nil) != tc.allowed {
				t.Fatalf("permission mismatch: %v", err)
			}
		})
	}
}

func TestForwardMissingPersonalMuteFailsClosed(t *testing.T) {
	client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/get_group_info") {
			return forwardReply(`{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"g","ErrorCode":0,"Name":"group","ShutUpAllMember":"Off"}]}`), nil
		}
		return forwardReply(`{"ActionStatus":"OK","ErrorCode":0,"MemberList":[{"Member_Account":"alice","Role":"Member"}]}`), nil
	})
	if _, err := client.CheckForwardGroup(context.Background(), "g", "alice"); err == nil {
		t.Fatal("missing permission must not send")
	}
}

func TestForwardQueryGateRunsBeforeEachActualRequest(t *testing.T) {
	var permit string
	queries := 0
	client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
		if permit == "" || !strings.HasSuffix(r.URL.Path, "/"+permit) {
			t.Fatal("REST query bypassed its gate")
		}
		command := permit
		permit = ""
		queries++
		if command == "get_group_info" {
			return forwardReply(`{"ActionStatus":"OK","ErrorCode":0,"GroupInfo":[{"GroupId":"g","ErrorCode":0,"Name":"group","MuteAllMember":"Off"}]}`), nil
		}
		return forwardReply(`{"ActionStatus":"FAIL","ErrorCode":60018}`), nil
	})
	_, err := client.CheckForwardGroup(context.Background(), "g", "alice", func(ctx context.Context, command string) error {
		permit = command
		return nil
	})
	var delivery *DeliveryError
	if queries != 2 || !errors.As(err, &delivery) || delivery.Command != "get_specified_group_member_info" || delivery.Code != 60018 {
		t.Fatalf("query rate rejection lost its scope: %v (%d queries)", err, queries)
	}
	_, err = client.CheckForwardGroup(context.Background(), "g", "alice", func(context.Context, string) error { return context.Canceled })
	if !errors.Is(err, context.Canceled) || queries != 2 {
		t.Fatal("cancelled gate must not issue a request")
	}
}
