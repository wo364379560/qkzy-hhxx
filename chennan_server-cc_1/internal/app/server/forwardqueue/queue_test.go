package forwardqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
	"time"
)

func validRequest() Request {
	return Request{RequestID: "request-00000001", Targets: []Target{{ConversationID: "GROUPg"}}, Messages: []Message{{Body: []tencentim.MessageElement{{MsgType: "TIMTextElem", MsgContent: json.RawMessage(`{"Text":"hello"}`)}}}}}
}
func mockQueue(t *testing.T) (Queue, sqlmock.Sqlmock) {
	t.Helper()
	raw, m, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: raw, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return Queue{DB: db}, m
}

func TestValidationAndSenderIsolation(t *testing.T) {
	r := validRequest()
	if err := Validate(r); err != nil {
		t.Fatal(err)
	}
	if jobID("alice", r.RequestID) == jobID("bob", r.RequestID) {
		t.Fatal("job ownership must be isolated")
	}
	r.Targets = append(r.Targets, r.Targets[0])
	if Validate(r) == nil {
		t.Fatal("duplicate target accepted")
	}
	r = validRequest()
	r.Messages[0].Body[0].MsgType = "TIMGroupSystemNoticeElem"
	if Validate(r) == nil {
		t.Fatal("system notice accepted")
	}
	r = validRequest()
	r.Messages[0].Body[0].MsgContent = json.RawMessage(`{"Text":""}`)
	if Validate(r) == nil {
		t.Fatal("empty text accepted")
	}
	r = validRequest()
	r.Messages = make([]Message, 3001)
	if Validate(r) == nil {
		t.Fatal("invalid empty message accepted")
	}
}

func TestRecoveryDoesNotReplayUncertainSends(t *testing.T) {
	q, m := mockQueue(t)
	m.ExpectBegin()
	m.ExpectExec("UPDATE `forward_deliveries` SET .* WHERE status = 'sending' AND lease_until <").WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectCommit()
	m.ExpectBegin()
	m.ExpectExec("UPDATE `forward_deliveries` SET .* WHERE status = 'preparing' AND lease_until <").WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectCommit()
	m.ExpectExec("UPDATE forward_deliveries d JOIN forward_deliveries p.*p.status IN").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := q.recoverExpired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRetryPolicy(t *testing.T) {
	for _, tc := range []struct {
		err     error
		sent    bool
		attempt int
		status  string
	}{
		{nil, true, 1, "success"}, {&tencentim.DeliveryError{Code: 60018}, true, 1, "retrying"},
		{&tencentim.DeliveryError{Code: 10023}, true, 7, "failed"}, {&tencentim.DeliveryError{Code: 10017}, true, 1, "failed"},
		{&tencentim.DeliveryError{Code: 20009}, true, 1, "failed"}, {errors.New("timeout"), true, 1, "unknown"},
		{&tencentim.DeliveryError{Unknown: true}, true, 1, "unknown"}, {errors.New("lookup timeout"), false, 1, "retrying"},
	} {
		status, delay, _ := outcome(tc.err, tc.sent, tc.attempt)
		if status != tc.status {
			t.Fatalf("%+v => %s", tc, status)
		}
		if status == "retrying" && (delay < 2*time.Second || delay > 20*time.Second) {
			t.Fatal("invalid delay")
		}
	}
}

func TestClaimOnlyOneWorkerCanWin(t *testing.T) {
	for _, won := range []bool{false, true} {
		q, m := mockQueue(t)
		m.ExpectQuery("SELECT.*FROM.*forward_deliveries.*NOT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(12, "pending"))
		m.ExpectBegin()
		var affected int64
		if won {
			affected = 1
		}
		m.ExpectExec("UPDATE `forward_deliveries` SET .* WHERE id = .* AND status IN").WillReturnResult(sqlmock.NewResult(0, affected))
		m.ExpectCommit()
		item, err := q.claim(context.Background())
		if err != nil || (item != nil) != won {
			t.Fatalf("winner=%t item=%v err=%v", won, item, err)
		}
		if won && item.Claim == "" {
			t.Fatal("lease must have a fencing token")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNoBusinessCapacityLimit(t *testing.T) {
	for _, pair := range [][2]int{{3001, 1}, {500, 7}, {4301, 7}, {1, 5000}} {
		r := validRequest()
		message := r.Messages[0]
		r.Targets = make([]Target, pair[0])
		r.Messages = make([]Message, pair[1])
		for i := range r.Targets {
			r.Targets[i].ConversationID = fmt.Sprintf("C2Cuser%d", i)
		}
		for i := range r.Messages {
			r.Messages[i] = message
		}
		if err := Validate(r); err != nil {
			t.Fatalf("%v: %v", pair, err)
		}
	}
}
