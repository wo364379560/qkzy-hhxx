package forwardqueue

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

func clearPacingEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"FORWARD_C2C_SEND_RPS", "FORWARD_GROUP_SEND_RPS", "FORWARD_GROUP_INFO_RPS", "FORWARD_GROUP_MEMBER_RPS", "FORWARD_PER_GROUP_RPS", "FORWARD_APP_RPS", "FORWARD_SENDER_RPS", "FORWARD_WORKERS", "FORWARD_MATERIALIZE_HIGH_WATER", "FORWARD_MATERIALIZE_BATCH", "FORWARD_SDK_SEND_INTERVAL_MS", "FORWARD_SDK_PREPARE_INTERVAL_MS", "FORWARD_SDK_CONCURRENCY"} {
		t.Setenv(key, "")
	}
}
func TestPacingConfigurationIsExplicitAndValidated(t *testing.T) {
	clearPacingEnv(t)
	s, err := LoadPacingSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.AppRPS != 0 || s.SenderRPS != 0 || s.C2CRPS != 100 || s.GroupRPS != 100 {
		t.Fatalf("default unrelated sends must not share an arbitrary 40/s gate: %+v", s)
	}
	t.Setenv("FORWARD_C2C_SEND_RPS", "200")
	t.Setenv("FORWARD_SENDER_RPS", "15")
	t.Setenv("FORWARD_SDK_PREPARE_INTERVAL_MS", "1500")
	s, err = LoadPacingSettings()
	if err != nil || s.C2CRPS != 200 || s.SenderRPS != 15 || s.SDKPrepareIntervalMS != 1500 {
		t.Fatalf("configured quotas ignored: %+v %v", s, err)
	}
	for _, tc := range []struct{ k, v string }{{"FORWARD_PER_GROUP_RPS", "41"}, {"FORWARD_GROUP_INFO_RPS", "garbage"}, {"FORWARD_WORKERS", "0"}, {"FORWARD_APP_RPS", "-1"}, {"FORWARD_MATERIALIZE_BATCH", "201"}} {
		t.Run(tc.k, func(t *testing.T) {
			t.Setenv(tc.k, tc.v)
			s, err := LoadPacingSettings()
			if err == nil || s.ConfigValid {
				t.Fatal("invalid limits must fail closed")
			}
		})
	}
}
func TestBudgetsFollowApplicationCommandSenderAndGroup(t *testing.T) {
	clearPacingEnv(t)
	s := CurrentSettings()
	s.SDKAppID = 123
	a, _ := s.gates("sendmsg", "alice", "C2Cbob")
	b, _ := s.gates("send_group_msg", "alice", "GROUPg")
	query, _ := s.gates("get_group_info", "", "")
	if a[0].id != "im:123:api:sendmsg" || a[0].interval != 10 || a[1].id != "im:123:app" || a[1].interval != 0 {
		t.Fatalf("unexpected C2C gates: %+v", a)
	}
	if b[0].id == a[0].id || query[0].id == a[0].id || query[0].id == b[0].id {
		t.Fatal("unrelated commands must not share a frequency bucket")
	}
	if b[2].id != "im:123:group:"+identityKey("GROUPg") || b[2].interval != 50 {
		t.Fatalf("per-group budget missing: %+v", b)
	}
	s.SenderRPS = 5
	userA, _ := s.gates("sendmsg", "alice", "C2Cbob")
	userB, _ := s.gates("sendmsg", "bob", "C2Cbob")
	if userA[2].id == userB[2].id || userA[2].interval != 200 {
		t.Fatal("sender quota must use actual sender identity, not IP")
	}
	s.SDKAppID = 456
	other, _ := s.gates("sendmsg", "alice", "C2Cbob")
	for i := range other {
		if other[i].id == userA[i].id {
			t.Fatal("separate Tencent apps must never share gates")
		}
	}
}

func expectPacing(m sqlmock.Sqlmock, ids []string, next map[string]int64, updates map[string]int64) {
	m.ExpectBegin()
	for _, id := range ids {
		m.ExpectExec("INSERT INTO `forward_gates`").WillReturnResult(sqlmock.NewResult(0, 1))
		m.ExpectQuery("SELECT .* FROM `forward_gates`.*FOR UPDATE").WithArgs(id, 1).WillReturnRows(sqlmock.NewRows([]string{"id", "next_at"}).AddRow(id, next[id]))
	}
	m.ExpectQuery("SELECT CAST").WillReturnRows(sqlmock.NewRows([]string{"now"}).AddRow(1000))
	for _, id := range ids {
		if n, ok := updates[id]; ok {
			m.ExpectExec("UPDATE `forward_gates`").WithArgs(n, id).WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}
	m.ExpectCommit()
}
func TestDatabasePacingWaitsWithoutConsumingOtherBudgets(t *testing.T) {
	clearPacingEnv(t)
	prefix := fmt.Sprintf("im:%d:", CurrentSettings().SDKAppID)
	for _, tc := range []struct {
		name    string
		next    map[string]int64
		wait    time.Duration
		updates map[string]int64
	}{
		{"first C2C", nil, 0, map[string]int64{prefix + "api:sendmsg": 1010}},
		{"same command reserved by other server", map[string]int64{prefix + "api:sendmsg": 1010}, 10 * time.Millisecond, nil},
		{"explicit app-wide 60011 pause", map[string]int64{prefix + "app": 3000}, 2 * time.Second, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, m := mockQueue(t)
			expectPacing(m, []string{prefix + "api:sendmsg", prefix + "app"}, tc.next, tc.updates)
			wait, err := q.slot(context.Background(), &model.ForwardDelivery{Sender: "alice", ConversationID: "C2Cbob"})
			if err != nil || wait != tc.wait {
				t.Fatalf("wait=%v err=%v", wait, err)
			}
			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestEachPermissionQueryReservesOwnBudgetAndHonorsCancellation(t *testing.T) {
	clearPacingEnv(t)
	prefix := fmt.Sprintf("im:%d:", CurrentSettings().SDKAppID)
	for _, command := range []string{"get_group_info", "get_specified_group_member_info"} {
		q, m := mockQueue(t)
		id := prefix + "api:" + command
		expectPacing(m, []string{id, prefix + "app"}, nil, map[string]int64{id: 1010})
		wait, err := q.querySlot(context.Background(), command)
		if err != nil || wait != 0 {
			t.Fatalf("query=%s wait=%v err=%v", command, wait, err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := q.waitQuery(ctx, command); !errors.Is(err, context.Canceled) {
			t.Fatal("cancelled query must not contact database/Tencent")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestCooldownOnlyUsesKnownSharedDimension(t *testing.T) {
	s := defaultPacing()
	s.SDKAppID = 123
	for _, tc := range []struct {
		err                 error
		sent                bool
		conversation, scope string
	}{
		{errors.New("network"), false, "GROUPg", ""},
		{&tencentim.DeliveryError{Unknown: true}, false, "GROUPg", ""},
		{&tencentim.DeliveryError{Code: 10017}, true, "GROUPg", ""},
		{&tencentim.DeliveryError{Code: 60011, Command: "get_group_info"}, false, "GROUPg", "app"},
		{&tencentim.DeliveryError{Code: 60007, Command: "get_group_info"}, false, "GROUPg", "api:get_group_info"},
		{&tencentim.DeliveryError{Code: 60018, Command: "get_specified_group_member_info"}, false, "GROUPg", "api:get_specified_group_member_info"},
		{&tencentim.DeliveryError{Code: 60019}, false, "GROUPg", ""},
		{&tencentim.DeliveryError{Code: 60018}, true, "C2Cbob", "api:sendmsg"},
		{&tencentim.DeliveryError{Code: 10023}, true, "GROUPg", "group:" + identityKey("GROUPg")},
	} {
		want := ""
		if tc.scope != "" {
			want = s.gateID(tc.scope)
		}
		if got := s.cooldownGate(tc.err, tc.sent, &model.ForwardDelivery{ConversationID: tc.conversation, Sender: "alice"}); got != want {
			t.Fatalf("%+v: got %q want %q", tc.err, got, want)
		}
	}
}
