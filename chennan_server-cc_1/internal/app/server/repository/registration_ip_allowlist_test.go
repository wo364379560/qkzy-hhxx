package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
)

func TestNormalizeRegistrationIP(t *testing.T) {
	for _, bad := range []string{"", "example.com", "8.8.8.8/32", "8.8.8.8:80", "10.0.0.1", "127.0.0.1", "100.64.0.1", "192.168.1.1", "192.0.2.1", "198.51.100.1", "203.0.113.1", "198.18.0.1", "224.0.0.1", "240.0.0.1", "::", "::1", "fc00::1", "fe80::1%en0", "2001:db8::1", "3fff::1", "::ffff:127.0.0.1"} {
		if _, err := NormalizeRegistrationIP(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	for input, want := range map[string]string{"8.8.8.8": "8.8.8.8", " ::ffff:8.8.4.4 ": "8.8.4.4", "2001:4860:4860::8888": "2001:4860:4860::8888"} {
		got, err := NormalizeRegistrationIP(input)
		if err != nil || got != want {
			t.Errorf("%q -> %q, %v", input, got, err)
		}
	}
}

func TestRegistrationIPReadsDatabaseOnEveryDecision(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		value string
		want  bool
	}{{"[]", false}, {`[{"ip":"8.8.8.8"}]`, true}, {"[]", false}} {
		mock.ExpectQuery("SELECT .*system_settings").WillReturnRows(sqlmock.NewRows([]string{"setting_key", "value"}).AddRow(registrationIPAllowlistKey, tt.value))
		got, err := registrationIPAllowed(context.Background(), db, "8.8.8.8")
		if err != nil || got != tt.want {
			t.Fatalf("allowed=%v want=%v err=%v", got, tt.want, err)
		}
	}
	mock.ExpectQuery("SELECT .*system_settings").WillReturnError(errors.New("storage unavailable"))
	if got, err := registrationIPAllowed(context.Background(), db, "8.8.8.8"); got || err == nil {
		t.Fatal("storage failure granted exemption")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationIPEditsLockAndPreserveOtherEntries(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	for _, remove := range []bool{false, true} {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO .*system_settings").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery("SELECT .*system_settings.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"setting_key", "value"}).AddRow(registrationIPAllowlistKey, `[{"ip":"8.8.4.4"}]`))
		mock.ExpectExec("UPDATE .*system_settings").WithArgs(uint(9), allowlistJSON{removed: remove}, sqlmock.AnyArg(), registrationIPAllowlistKey).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO .*admin_access_audits").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		if err := ChangeRegistrationIP(context.Background(), db, 9, "8.8.8.8", "test", remove, model.AdminAccessAudit{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type allowlistJSON struct{ removed bool }

func (m allowlistJSON) Match(value driver.Value) bool {
	raw, ok := value.(string)
	if !ok {
		return false
	}
	var entries []RegistrationIPEntry
	if json.Unmarshal([]byte(raw), &entries) != nil {
		return false
	}
	if len(entries) == 0 || entries[0].IP != "8.8.4.4" {
		return false
	}
	if m.removed {
		return len(entries) == 1
	}
	return len(entries) == 2 && entries[1].IP == "8.8.8.8" && entries[1].UpdatedBy == 9
}

func TestRegistrationIPAuditFailureRollsBackChange(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO .*system_settings").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT .*system_settings.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"setting_key", "value"}).AddRow(registrationIPAllowlistKey, "[]"))
	mock.ExpectExec("UPDATE .*system_settings").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO .*admin_access_audits").WillReturnError(errors.New("audit unavailable"))
	mock.ExpectRollback()
	if err := ChangeRegistrationIP(context.Background(), db, 9, "8.8.8.8", "test", false, model.AdminAccessAudit{}); err == nil {
		t.Fatal("unaudited change succeeded")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
