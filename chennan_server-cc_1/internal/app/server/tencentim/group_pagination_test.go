package tencentim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGroupPaginationUsesOnlyItsDeclaredMode(t *testing.T) {
	for _, kind := range []string{"Community", "Public"} {
		for _, count := range []int{0, 100, 200, 1224} {
			t.Run(fmt.Sprintf("%s/%d", kind, count), func(t *testing.T) {
				offset, pages := 0, 0
				client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
					if strings.HasSuffix(r.URL.Path, "/get_group_info") {
						return forwardReply(fmt.Sprintf(`{"ErrorCode":0,"GroupInfo":[{"GroupId":"g","Type":%q}]}`, kind)), nil
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if kind == "Community" {
						if _, ok := body["Offset"]; ok {
							t.Fatal("Community fell back to Offset")
						}
						want := ""
						if offset > 0 {
							want = fmt.Sprintf("next-%d", offset)
						}
						if body["Next"] != want {
							t.Fatalf("Next=%v want %q", body["Next"], want)
						}
					} else {
						if _, ok := body["Next"]; ok {
							t.Fatal("ordinary group sent Next")
						}
						if body["Offset"] != float64(offset) {
							t.Fatalf("Offset=%v want %d", body["Offset"], offset)
						}
					}
					end := min(offset+100, count)
					members := make([]map[string]any, 0)
					for i := offset; i < end; i++ {
						members = append(members, map[string]any{"Member_Account": fmt.Sprintf("u%d", i), "Role": "Member"})
					}
					payload := map[string]any{"ErrorCode": 0, "MemberList": members, "MemberNum": count}
					if kind == "Community" {
						next := ""
						if end < count {
							next = fmt.Sprintf("next-%d", end)
						}
						payload["Next"] = next
					}
					offset = end
					pages++
					if pages > max(1, (count+99)/100) {
						t.Fatal("requested extra page after final cursor")
					}
					raw, _ := json.Marshal(payload)
					return forwardReply(string(raw)), nil
				})
				got, err := client.GroupGetAllMembers("g")
				if err != nil || len(got) != count {
					t.Fatalf("count=%d error=%v", len(got), err)
				}
			})
		}
	}
}

func TestCommunityRejectsMissingCursorAndChangingCount(t *testing.T) {
	for _, pages := range [][]string{
		{`{"ErrorCode":0,"MemberNum":1,"MemberList":[{"Member_Account":"a","Role":"Member"}]}`},
		{`{"ErrorCode":0,"Next":"p2","MemberNum":2,"MemberList":[{"Member_Account":"a","Role":"Member"}]}`, `{"ErrorCode":0,"Next":"","MemberNum":1,"MemberList":[{"Member_Account":"b","Role":"Member"}]}`},
	} {
		page := 0
		client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
			body := pages[page]
			page++
			return forwardReply(body), nil
		})
		got, err := client.groupGetAllMembersForType(context.Background(), "g", "Community")
		if err == nil || got != nil {
			t.Fatal("partial/unstable snapshot accepted")
		}
	}
}

func TestCommunityFinalFullPageWithoutTotalStopsAtEmptyCursor(t *testing.T) {
	calls := 0
	client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls > 1 {
			t.Fatal("empty cursor retried")
		}
		members := []map[string]any{}
		for i := 0; i < 100; i++ {
			members = append(members, map[string]any{"Member_Account": fmt.Sprint(i), "Role": "Member"})
		}
		raw, _ := json.Marshal(map[string]any{"ErrorCode": 0, "MemberList": members, "Next": ""})
		return forwardReply(string(raw)), nil
	})
	got, err := client.groupGetAllMembersForType(context.Background(), "g", "Community")
	if err != nil || len(got) != 100 {
		t.Fatalf("members=%d err=%v", len(got), err)
	}
}

func TestGroupMemberReadHonorsTotalDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	calls := 0
	client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
		calls++
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	started := time.Now()
	if _, err := client.GroupGetAllMembersContext(ctx, "g"); err == nil {
		t.Fatal("cancellation ignored")
	}
	if time.Since(started) > time.Second || calls != 1 {
		t.Fatalf("unbounded request or replay: %d", calls)
	}
}
