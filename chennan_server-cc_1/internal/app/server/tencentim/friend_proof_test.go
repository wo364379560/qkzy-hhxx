package tencentim

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type friendTransport func(*http.Request) (*http.Response, error)

func (f friendTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Uses the official REST response shape (InfoItem), which differs from the
// ResultItem shape used by friend_add and the Web SDK's successUserIDList.
func TestCheckBilateralRESTContract(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		expected   bool
		fail       bool
	}{
		{"both", `{"ActionStatus":"OK","ErrorCode":0,"InfoItem":[{"To_Account":"PeerCase","Relation":"CheckResult_Type_BothWay","ResultCode":0}]}`, true, false},
		{"one_way", `{"ActionStatus":"OK","ErrorCode":0,"InfoItem":[{"To_Account":"PeerCase","Relation":"CheckResult_Type_AWithB","ResultCode":0}]}`, false, false},
		{"no_relationship", `{"ActionStatus":"OK","ErrorCode":0,"InfoItem":[{"To_Account":"PeerCase","Relation":"CheckResult_Type_NoRelation","ResultCode":0}]}`, false, false},
		{"per_user_failure", `{"ActionStatus":"OK","ErrorCode":0,"InfoItem":[{"To_Account":"PeerCase","Relation":"CheckResult_Type_BothWay","ResultCode":30006}]}`, false, true},
		{"wrong_response_field", `{"ActionStatus":"OK","ErrorCode":0,"ResultItem":[{"To_Account":"PeerCase","Relation":"CheckResult_Type_BothWay","ResultCode":0}]}`, false, true},
		{"missing_account", `{"ActionStatus":"OK","ErrorCode":0,"InfoItem":[]}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{sdkAppID: 7, appKey: "fixture", adminUser: "fixture-admin", http: &http.Client{Transport: friendTransport(func(req *http.Request) (*http.Response, error) {
				var body struct {
					From string   `json:"From_Account"`
					Type string   `json:"CheckType"`
					To   []string `json:"To_Account"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.From != "caller" || body.Type != "CheckResult_Type_Both" || len(body.To) != 1 || body.To[0] != "PeerCase" {
					t.Fatal("incorrect bilateral request")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{}}, nil
			})}}
			result, err := c.CheckBilateral(context.Background(), "caller", []string{"PeerCase"})
			if (err != nil) != tc.fail {
				t.Fatalf("error mismatch: %v", err)
			}
			if !tc.fail && result["PeerCase"] != tc.expected {
				t.Fatal("wrong bilateral proof")
			}
		})
	}
}
