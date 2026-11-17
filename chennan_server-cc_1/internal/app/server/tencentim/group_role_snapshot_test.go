package tencentim

import (
	"net/http"
	"testing"
)

func TestGroupRoleMirrorRequiresCompleteMemberSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name      string
		pages     []string
		wantCount int
		wantError bool
	}{
		{"complete", []string{
			`{"ErrorCode":0,"MemberNum":2,"Next":"page2","MemberList":[{"Member_Account":"owner","Role":"Owner"}]}`,
			`{"ErrorCode":0,"MemberNum":2,"MemberList":[{"Member_Account":"admin","Role":"Admin"}]}`,
		}, 2, false},
		{"empty", []string{`{"ErrorCode":0,"MemberNum":0,"MemberList":[]}`}, 0, false},
		{"missing_list", []string{`{"ErrorCode":0}`}, 0, true},
		{"truncated", []string{`{"ErrorCode":0,"MemberNum":2,"MemberList":[{"Member_Account":"owner","Role":"Owner"}]}`}, 0, true},
		{"empty_page_before_end", []string{`{"ErrorCode":0,"MemberNum":2,"MemberList":[]}`}, 0, true},
		{"repeated_cursor", []string{
			`{"ErrorCode":0,"MemberNum":3,"Next":"page2","MemberList":[{"Member_Account":"owner","Role":"Owner"}]}`,
			`{"ErrorCode":0,"MemberNum":3,"Next":"page2","MemberList":[{"Member_Account":"admin","Role":"Admin"}]}`,
		}, 0, true},
		{"duplicate_member", []string{
			`{"ErrorCode":0,"MemberNum":2,"Next":"page2","MemberList":[{"Member_Account":"owner","Role":"Owner"}]}`,
			`{"ErrorCode":0,"MemberNum":2,"MemberList":[{"Member_Account":"owner","Role":"Owner"}]}`,
		}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := 0
			client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
				if page >= len(tc.pages) {
					t.Fatal("unexpected extra page request")
				}
				body := tc.pages[page]
				page++
				return forwardReply(body), nil
			})
			members, err := client.GroupGetAllMembers("test")
			if (err != nil) != tc.wantError || len(members) != tc.wantCount {
				t.Fatalf("incomplete snapshot accepted: members=%v err=%v", members, err)
			}
		})
	}
}
