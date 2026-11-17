package handler_webhook

import (
	"encoding/json"
	"testing"
)

func TestPrevFriendAddFromAccountFallback(t *testing.T) {
	t.Parallel()

	req := prevFriendAddReq{FromAccount: "u_from"}
	if got := req.fromAccount(); got != "u_from" {
		t.Fatalf("fromAccount=%s", got)
	}
	req = prevFriendAddReq{RequesterAccount: "u_requester"}
	if got := req.fromAccount(); got != "u_requester" {
		t.Fatalf("requester fallback=%s", got)
	}
}

func TestResolveFriendAddPair(t *testing.T) {
	t.Parallel()

	initiator, peer, ok := resolveFriendAddPair("", "a_18822224444", "a_14422224444")
	if !ok || initiator != "a_18822224444" || peer != "a_14422224444" {
		t.Fatalf("empty initiator: initiator=%s peer=%s ok=%v", initiator, peer, ok)
	}

	initiator, peer, ok = resolveFriendAddPair("a_18822224444", "a_18822224444", "a_14422224444")
	if !ok || initiator != "a_18822224444" || peer != "a_14422224444" {
		t.Fatalf("explicit initiator: initiator=%s peer=%s ok=%v", initiator, peer, ok)
	}

	initiator, peer, ok = resolveFriendAddPair("a_14422224444", "a_18822224444", "a_14422224444")
	if !ok || initiator != "a_14422224444" || peer != "a_18822224444" {
		t.Fatalf("accepter as initiator: initiator=%s peer=%s ok=%v", initiator, peer, ok)
	}

	if _, _, ok = resolveFriendAddPair("", "a_same", "a_same"); ok {
		t.Fatal("same account should skip")
	}
	if initiator, peer, ok = resolveFriendAddPair("", "UserCase", "usercase"); !ok || initiator != "UserCase" || peer != "usercase" {
		t.Fatal("Tencent account IDs must remain case sensitive")
	}
}

func TestFriendAddReqUnmarshalTencentPayload(t *testing.T) {
	t.Parallel()

	raw := `{
		"CallbackCommand": "Sns.CallbackFriendAdd",
		"PairList": [
			{
				"From_Account": "a_18822224444",
				"To_Account": "a_14422224444",
				"Initiator_Account": "a_18822224444"
			}
		],
		"ClientCmd": "FriendResponse"
	}`
	var req friendAddReq
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if req.CallbackCommand != cmdFriendAdd {
		t.Fatalf("command=%s", req.CallbackCommand)
	}
	if len(req.PairList) != 1 {
		t.Fatalf("pairs=%d", len(req.PairList))
	}
	initiator, peer, ok := resolveFriendAddPair(req.PairList[0].InitiatorAccount, req.PairList[0].FromAccount, req.PairList[0].ToAccount)
	if !ok || initiator != "a_18822224444" || peer != "a_14422224444" {
		t.Fatalf("initiator=%s peer=%s ok=%v", initiator, peer, ok)
	}
}

func TestPrevFriendResponseUnmarshalAgree(t *testing.T) {
	t.Parallel()

	raw := `{
		"CallbackCommand": "Sns.CallbackPrevFriendResponse",
		"From_Account": "a_14422224444",
		"ResponseFriendItem": [
			{"To_Account": "a_18822224444", "ResponseAction": "Response_Action_AgreeAndAdd"}
		]
	}`
	var req prevFriendResponseReq
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if req.CallbackCommand != cmdPrevFriendResponse {
		t.Fatalf("command=%s", req.CallbackCommand)
	}
	if !isFriendAgreeAction(req.ResponseFriendItem[0].ResponseAction) {
		t.Fatal("expected agree action")
	}
	if isFriendAgreeAction("Response_Action_Reject") {
		t.Fatal("reject should not migrate")
	}
	if !isFriendAgreeAction("AgreeAndAdd") {
		t.Fatal("short agree action should migrate")
	}
}

func TestFriendAddReqFallbackFromFriendItem(t *testing.T) {
	t.Parallel()

	raw := `{
		"CallbackCommand": "Sns.CallbackFriendAdd",
		"From_Account": "u_new",
		"FriendItem": [{"To_Account": "u_assigned"}]
	}`
	var req friendAddReq
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	pairs := req.friendPairs()
	if len(pairs) != 1 || pairs[0].FromAccount != "u_new" || pairs[0].ToAccount != "u_assigned" {
		t.Fatalf("pairs=%+v", pairs)
	}
}

func TestPrevFriendResponseFriendItemAlias(t *testing.T) {
	t.Parallel()

	raw := `{
		"CallbackCommand": "Sns.CallbackPrevFriendResponse",
		"From_Account": "u_assigned",
		"FriendItem": [
			{"To_Account": "u_new", "ResponseAction": "AgreeAndAdd"}
		]
	}`
	var req prevFriendResponseReq
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	items := req.responseItems()
	if len(items) != 1 || items[0].ToAccount != "u_new" {
		t.Fatalf("items=%+v", items)
	}
	if !isFriendAgreeAction(items[0].ResponseAction) {
		t.Fatal("expected agree")
	}
}
