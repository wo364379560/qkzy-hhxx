package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestOwnerTaskRetriesEntireTransactionOnlyOnDeadlock(t *testing.T) {
	deadlock := fmt.Errorf("apply: %w", &mysql.MySQLError{Number: 1213})
	for _, test := range []struct {
		name   string
		errors []error
	}{
		{name: "one deadlock then success", errors: []error{deadlock, nil}},
		{name: "deadlock budget exhausted", errors: []error{deadlock, deadlock, deadlock}},
		{name: "lock timeout", errors: []error{&mysql.MySQLError{Number: 1205}}},
		{name: "duplicate key", errors: []error{&mysql.MySQLError{Number: 1062}}},
		{name: "policy error", errors: []error{OwnershipError{"actor_forbidden"}}},
		{name: "untyped text", errors: []error{errors.New("Error 1213")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			db, err := gorm.Open(mysqlgorm.New(mysqlgorm.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, attemptErr := range test.errors {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
				if attemptErr != nil {
					mock.ExpectRollback()
				} else {
					mock.ExpectCommit()
				}
			}
			calls := 0
			repo := &OwnershipRepository{DB: db}
			err = repo.taskTransaction(context.Background(), "test", "op", false, func(tx *gorm.DB, _ *ownerTaskAttempt) error {
				var n int
				if err := tx.Raw("SELECT 1").Scan(&n).Error; err != nil {
					return err
				}
				if calls >= len(test.errors) {
					t.Fatal("retried after budget or non-deadlock")
				}
				attemptErr := test.errors[calls]
				calls++
				return attemptErr
			})
			if calls != len(test.errors) || !errors.Is(err, test.errors[len(test.errors)-1]) {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOwnerTaskCancellationStopsDeadlockRetry(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		sqlDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		db, err := gorm.Open(mysqlgorm.New(mysqlgorm.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		want := context.Canceled
		calls := 0
		if deadline {
			cancel()
			ctx, cancel = context.WithTimeout(context.Background(), 0)
			want = context.DeadlineExceeded
		} else {
			mock.ExpectBegin()
			mock.ExpectRollback()
		}
		repo := &OwnershipRepository{DB: db}
		err = repo.taskTransaction(ctx, "test", "op", false, func(*gorm.DB, *ownerTaskAttempt) error {
			calls++
			cancel()
			return &mysql.MySQLError{Number: 1213}
		})
		cancel()
		_ = sqlDB.Close()
		if !errors.Is(err, want) || (deadline && calls != 0) || (!deadline && calls != 1) {
			t.Fatalf("deadline=%v calls=%d err=%v", deadline, calls, err)
		}
	}
}
