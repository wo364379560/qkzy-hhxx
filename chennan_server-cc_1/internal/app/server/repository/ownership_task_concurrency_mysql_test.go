package repository

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

func TestOwnerApplyRecoversRealDeadlockMySQL(t *testing.T) {
	db := ownershipTestDB(t)
	ownershipFixtures(t, db)
	repo := &OwnershipRepository{DB: db}
	job := claimOwnerPair(t, repo, "real-deadlock", "u_202", "u_201")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	// Make the opposing transaction heavier so InnoDB selects the ownership
	// transaction as the victim. This database contains only fixture data.
	if err := db.Exec("CREATE TABLE owner_deadlock_weight (id INT PRIMARY KEY, weight INT) ENGINE=InnoDB").Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		if err := db.Exec("INSERT INTO owner_deadlock_weight VALUES (?, 0)", i).Error; err != nil {
			t.Fatal(err)
		}
	}
	jobLocked := make(chan struct{})
	var attempts atomic.Int32
	if err := db.Callback().Query().After("gorm:query").Register("test:owner_deadlock", func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*model.FriendOwnerOperation); ok &&
			row.ID == job.ID && tx.Statement.Clauses["FOR"].Expression != nil && tx.Error == nil {
			if attempts.Add(1) == 1 {
				close(jobLocked)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}

	userLocked := make(chan struct{})
	rivalDone := make(chan error, 1)
	go func() {
		rivalDone <- db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("UPDATE owner_deadlock_weight SET weight = weight + 1").Error; err != nil {
				return err
			}
			var id uint
			if err := tx.Raw("SELECT id FROM users WHERE id = ? FOR UPDATE", 201).Scan(&id).Error; err != nil {
				return err
			}
			close(userLocked)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-jobLocked:
			}
			return tx.Model(&model.FriendOwnerOperation{}).Where("id = ?", job.ID).Update("last_error", "fixture rival").Error
		})
	}()
	select {
	case <-userLocked:
	case err := <-rivalDone:
		t.Fatalf("rival failed before taking the lock: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	result, err := repo.ApplyConfirmed(ctx, job, []string{"u_201"})
	if rivalErr := <-rivalDone; rivalErr != nil {
		t.Fatalf("rival transaction: %v", rivalErr)
	}
	if err != nil || result.State != "migrated" || attempts.Load() != 2 {
		t.Fatalf("apply after engine deadlock: result=%+v attempts=%d err=%v", result, attempts.Load(), err)
	}
	var saved model.User
	if err := db.First(&saved, 201).Error; err != nil {
		t.Fatal(err)
	}
	if saved.OwnerID != 110 || saved.OwnerRevision != 1 {
		t.Fatalf("expected one committed assignment, owner=%d revision=%d", saved.OwnerID, saved.OwnerRevision)
	}
	var task model.FriendOwnerOperation
	if err := db.First(&task, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.Attempts != 1 || !task.EffectsPending || task.State != "migrated" {
		t.Fatalf("database retry changed task budget or lost outbox: attempts=%d effects=%v state=%s", task.Attempts, task.EffectsPending, task.State)
	}
	// Emulate a caller that lost the commit response and still has the claimed
	// pending snapshot. Recovery must keep the durable assignment.
	if err := repo.Retry(ctx, job, "assignment_database_unavailable"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&task, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.State != "migrated" || !task.EffectsPending || task.LeaseToken != "" {
		t.Fatal("recovering with a stale snapshot reopened a committed assignment")
	}
}

func TestOwnerReviewSweepAndBindingsMySQL(t *testing.T) {
	db := ownershipTestDB(t)
	ownershipFixtures(t, db)
	repo := &OwnershipRepository{DB: db}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	create := func(task model.FriendOwnerOperation) model.FriendOwnerOperation {
		t.Helper()
		task.OperationID = OwnershipID()
		task.EventKey = OwnershipID()
		task.Source = "client"
		task.Caller = "u_202"
		task.Target = "u_201"
		task.AvailableAt = time.Now()
		if err := db.Create(&task).Error; err != nil {
			t.Fatal(err)
		}
		return task
	}
	held := create(model.FriendOwnerOperation{State: "pending", Attempts: ownerTaskMaxAttempts})
	exhaustedEffects := create(model.FriendOwnerOperation{State: "migrated", Attempts: ownerTaskMaxAttempts, EffectsPending: true})
	activeUntil := time.Now().Add(time.Minute)
	active := create(model.FriendOwnerOperation{State: "pending", Attempts: ownerTaskMaxAttempts, LeaseUntil: &activeUntil, LeaseToken: "active-lease"})
	for i := 0; i < ownerTaskBatchSize+2; i++ {
		create(model.FriendOwnerOperation{State: "pending", Attempts: ownerTaskMaxAttempts})
	}
	for id := uint(204); id <= 205; id++ {
		imID := fmt.Sprintf("u_%d", id)
		if err := db.Create(&model.User{ID: id, BizID: int64(id), Phone: imID, IMUserID: imID, Nickname: imID, Status: "active"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	holder := db.WithContext(ctx).Begin()
	if holder.Error != nil {
		t.Fatal(holder.Error)
	}
	defer holder.Rollback()
	if _, err := lockOwnerTask(holder, held.ID, false); err != nil {
		t.Fatal(err)
	}

	// Queue a durable callback while interactive requests bind the same and
	// distinct targets. The worker and sweep must make progress with the
	// expired record held by an unrelated transaction.
	workerJob, err := repo.Enqueue(ctx, "callback", "concurrent-worker", "u_203", "u_205")
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	done := make(chan error, 5)
	go func() {
		<-start
		done <- repo.MarkExpiredForReview(ctx)
	}()
	go func() {
		<-start
		op, err := repo.Claim(ctx, 0)
		if err == nil {
			_, err = repo.ApplyConfirmed(ctx, op, []string{op.Target})
		}
		if err == nil {
			err = repo.FinishEffects(ctx, op)
		}
		done <- err
	}()
	for i, pair := range [][2]string{{"u_202", "u_201"}, {"u_203", "u_201"}, {"u_202", "u_204"}} {
		go func(i int, caller, target string) {
			<-start
			op, err := repo.Enqueue(ctx, "client", fmt.Sprintf("concurrent-%d", i), caller, target)
			if err == nil {
				op, err = repo.Claim(ctx, op.ID)
			}
			if err == nil {
				_, err = repo.ApplyConfirmed(ctx, op, []string{target})
			}
			if err == nil {
				err = repo.FinishEffects(ctx, op)
			}
			done <- err
		}(i, pair[0], pair[1])
	}
	close(start)
	for i := 0; i < cap(done); i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	var reviewed int64
	if err := db.Model(&model.FriendOwnerOperation{}).Where("attempts >= ? AND needs_review", ownerTaskMaxAttempts).Count(&reviewed).Error; err != nil {
		t.Fatal(err)
	}
	if reviewed != ownerTaskBatchSize-1 {
		t.Fatalf("sweep must skip the locked record and stay bounded, reviewed=%d", reviewed)
	}
	if err := holder.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkExpiredForReview(ctx); err != nil {
		t.Fatal(err)
	}
	load := func(task model.FriendOwnerOperation) model.FriendOwnerOperation {
		t.Helper()
		var row model.FriendOwnerOperation
		if err := db.First(&row, task.ID).Error; err != nil {
			t.Fatal(err)
		}
		return row
	}
	if row := load(held); row.State != "manual" || !row.NeedsReview {
		t.Fatal("maintenance did not recover the unlocked last-attempt task")
	}
	if row := load(exhaustedEffects); row.State != "migrated" || !row.EffectsPending || !row.NeedsReview {
		t.Fatal("maintenance changed committed assignment or cleared unfinished effects")
	}
	if row := load(active); row.NeedsReview || row.LeaseToken != "active-lease" {
		t.Fatal("maintenance changed a current lease")
	}
	if row := load(*workerJob); row.State != "migrated" || row.EffectsPending {
		t.Fatal("worker did not finish the durable callback")
	}
	for id, owner := range map[uint]uint{204: 110, 205: 120} {
		var user model.User
		if err := db.First(&user, id).Error; err != nil || user.OwnerID != owner || user.OwnerRevision != 1 {
			t.Fatalf("independent binding %d failed: owner=%d revision=%d err=%v", id, user.OwnerID, user.OwnerRevision, err)
		}
	}
	var sameTarget []model.FriendOwnerOperation
	if err := db.Where("target = ? AND attempts < ?", "u_201", ownerTaskMaxAttempts).Find(&sameTarget).Error; err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, row := range sameTarget {
		counts[row.State]++
	}
	if counts["migrated"] != 1 || counts["conflict"] != 1 {
		t.Fatalf("competing owners did not preserve the original policy: %v", counts)
	}
	// A repeated request resolves to the original operation, including across
	// a process restart, and cannot claim its finished effects again.
	saved, err := repo.Enqueue(ctx, "callback", "concurrent-worker", "u_203", "u_205")
	if err != nil || saved.OperationID != workerJob.OperationID {
		t.Fatal("event idempotency was lost")
	}
	if _, err := repo.Claim(ctx, saved.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("finished task was claimable again: %v", err)
	}
}
