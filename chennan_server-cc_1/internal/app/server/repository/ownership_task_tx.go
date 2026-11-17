package repository

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

const ownerTransactionAttempts = 3

type ownershipRequestIDKey struct{}

// OwnershipRequestTrace uses a server-generated ID so raw client input is
// never copied into transaction logs. Workers can be joined by operation ID.
func OwnershipRequestTrace(ctx context.Context) (context.Context, string) {
	id := OwnershipID()
	return context.WithValue(ctx, ownershipRequestIDKey{}, id), id
}

func ownershipMySQLCode(err error) uint16 {
	var dbErr *mysql.MySQLError
	if errors.As(err, &dbErr) {
		return dbErr.Number
	}
	return 0
}

func IsOwnershipDeadlock(err error) bool {
	return ownershipMySQLCode(err) == 1213
}

func IsOwnershipLeaseLost(err error) bool {
	var policy OwnershipError
	return errors.As(err, &policy) && policy.Code == "lease_lost"
}

type ownerTaskAttempt struct {
	requestID   string
	operationID string
	stage       string
	number      int
	rows        int64
}

func (a *ownerTaskAttempt) fields(elapsed time.Duration, rows int64, err error) []zap.Field {
	return []zap.Field{
		zap.String("request_id", a.requestID),
		zap.String("operation_id", a.operationID),
		zap.String("stage", a.stage),
		zap.Int("attempt", a.number),
		zap.Uint16("mysql_code", ownershipMySQLCode(err)),
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
	}
}

// The default GORM logger interpolates values, including lease tokens and audit
// payloads. For this chain keep SQL templates and codes only.
type ownerTaskSQLLogger struct {
	gormlog.Interface
	attempt *ownerTaskAttempt
}

func (ownerTaskSQLLogger) ParamsFilter(_ context.Context, query string, _ ...interface{}) (string, []interface{}) {
	return query, nil
}

func (l ownerTaskSQLLogger) Trace(_ context.Context, start time.Time, queryAndRows func() (string, int64), err error) {
	elapsed := time.Since(start)
	if (err == nil && elapsed < time.Second) || errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	query, rows := queryAndRows()
	logger.Warn("friend ownership SQL", append(l.attempt.fields(elapsed, rows, err), zap.String("sql", query))...)
}

// taskTransaction retries the database unit only. Each attempt opens a new
// transaction and re-reads the state. IM calls and post-commit effects are
// outside this function.
func (r *OwnershipRepository) taskTransaction(ctx context.Context, stage, operationID string, guarded bool, work func(*gorm.DB, *ownerTaskAttempt) error) error {
	requestID, _ := ctx.Value(ownershipRequestIDKey{}).(string)
	for number := 1; ; number++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		attempt := ownerTaskAttempt{requestID: requestID, operationID: operationID, stage: stage, number: number}
		db := r.DB.Session(&gorm.Session{
			Context: ctx,
			Logger:  ownerTaskSQLLogger{Interface: gormlog.Discard, attempt: &attempt},
		})
		run := func(tx *gorm.DB) error { return work(tx, &attempt) }
		start := time.Now()
		var err error
		if guarded {
			err = ownershipTransaction(ctx, db, run)
		} else {
			err = db.Transaction(run)
		}
		fields := attempt.fields(time.Since(start), attempt.rows, err)
		if err == nil {
			if number > 1 {
				logger.Info("friend ownership transaction recovered", fields...)
			} else if attempt.rows > 0 {
				logger.Debug("friend ownership transaction committed", fields...)
			}
			return nil
		}
		if IsOwnershipLeaseLost(err) {
			logger.Debug("friend ownership lease no longer held", fields...)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("friend ownership transaction failed", fields...)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if !IsOwnershipDeadlock(err) || number >= ownerTransactionAttempts {
			return err
		}

		// Short exponential backoff with jitter. Wait outside any transaction.
		delay := time.Duration(15*(1<<(number-1))+rand.IntN(16)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
