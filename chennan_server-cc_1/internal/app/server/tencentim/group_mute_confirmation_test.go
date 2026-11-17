package tencentim

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMemberMuteRequiresConfirmedProviderState(t *testing.T) {
	for _, test := range []struct {
		name            string
		duration, until int64
		allowed         bool
	}{
		{"mute confirmed", 3600, time.Now().Unix() + 3600, true},
		{"provider ignored mute", 3600, 0, false},
		{"unmute confirmed", 0, 0, true},
		{"provider ignored unmute", 0, time.Now().Unix() + 3600, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			commands := []string{}
			client := forwardTestClient(func(request *http.Request) (*http.Response, error) {
				commands = append(commands, request.URL.Path)
				if strings.HasSuffix(request.URL.Path, "/forbid_send_msg") {
					return forwardReply(`{"ActionStatus":"OK","ErrorCode":0}`), nil
				}
				if !strings.HasSuffix(request.URL.Path, "/get_specified_group_member_info") {
					t.Fatalf("unexpected endpoint %s", request.URL.Path)
				}
				return forwardReply(fmt.Sprintf(`{"ActionStatus":"OK","ErrorCode":0,"MemberList":[{"Member_Account":"admin-member","MuteUntil":%d}]}`, test.until)), nil
			})
			err := client.GroupSetMemberMuteTime("rui-group", "admin-member", test.duration)
			if (err == nil) != test.allowed {
				t.Fatalf("unconfirmed mute reported success: %v", err)
			}
			if len(commands) != 2 {
				t.Fatalf("missing provider readback: %v", commands)
			}
		})
	}
}
