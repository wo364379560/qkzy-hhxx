package forwardqueue

import (
	"context"
	"fmt"
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

type snapshotReadContextKey struct{}

// Use the same explicitly opted-in disposable MySQL database as the parent
// suite. No application configuration or Tencent transport is accessed.
func TestForwardStatusRetrySnapshotMySQL(t *testing.T) {
	db := reviewDB(t)
	reviewMust(t, db.AutoMigrate(&model.ForwardJob{}, &model.ForwardDelivery{}, &model.ForwardRetry{}))
	for _, page := range []bool{false, true} {
		t.Run(fmt.Sprintf("page_%t_default_read_committed", page), func(t *testing.T) {
			const sender = "snapshot_review_sender"
			id := fmt.Sprintf("snapshot-review-%t", page)
			parentID := jobID(sender, id)
			clean := func() {
				for _, table := range []string{"forward_retries", "forward_deliveries", "forward_jobs"} {
					key := "job_id"
					if table == "forward_jobs" {
						key = "id"
					}
					reviewMust(t, db.Table(table).Where(key+" = ?", parentID).Delete(map[string]any{}).Error)
				}
			}
			clean()
			t.Cleanup(clean)
			job := model.ForwardJob{ID: parentID, Sender: sender, RequestID: id, Fingerprint: fingerprint(id), Protocol: 2, State: "queued", TargetCount: 1, MessageCount: 1, UploadedTargets: 1, UploadedMessages: 1, Materialized: 1}
			reviewMust(t, db.Create(&job).Error)
			row := model.ForwardDelivery{JobID: parentID, Sender: sender, ConversationID: "C2Csnapshot_review", Status: "failed", Message: "[]", Random: 29, Sequence: 1}
			reviewMust(t, db.Create(&row).Error)
			retry := model.ForwardRetry{JobID: parentID, RetryID: "snapshot-retry-001", UpperID: row.ID}
			reviewMust(t, db.Create(&retry).Error)

			// Trigger the last retry-page commit exactly after statusFor's delivery
			// aggregate, before its retry count. A separate connection commits the
			// real state transition while the read transaction remains open.
			fired := false
			callbackName := fmt.Sprintf("review:retry_snapshot:%t", page)
			interleave := func(tx *gorm.DB) {
				if fired || tx.Statement.Context.Value(snapshotReadContextKey{}) != true || tx.Statement.Table != "forward_retries" {
					return
				}
				fired = true
				err := db.Transaction(func(writer *gorm.DB) error {
					if err := writer.Model(&model.ForwardDelivery{}).Where("id = ?", row.ID).Update("status", "pending").Error; err != nil {
						return err
					}
					return writer.Model(&model.ForwardRetry{}).Where("job_id = ? AND retry_id = ?", parentID, retry.RetryID).Updates(map[string]any{"cursor": row.ID, "done": true}).Error
				})
				if err != nil {
					tx.AddError(err)
				}
			}
			reviewMust(t, db.Callback().Query().Before("gorm:query").Register(callbackName, interleave))
			t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })
			reviewMust(t, db.Callback().Row().Before("gorm:row").Register(callbackName, interleave))
			t.Cleanup(func() { _ = db.Callback().Row().Remove(callbackName) })

			var status *Snapshot
			reviewMust(t, db.Connection(func(conn *gorm.DB) error {
				// Prove the production helper chooses RR explicitly, rather than
				// accidentally relying on this database's default isolation level.
				if err := conn.Exec("SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED").Error; err != nil {
					return err
				}
				defer conn.Exec("SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ")
				ctx := context.WithValue(context.Background(), snapshotReadContextKey{}, true)
				q := Queue{DB: conn}
				var err error
				if page {
					status, err = q.GetPage(ctx, sender, id, 0)
				} else {
					status, err = q.Status(ctx, sender, id)
				}
				return err
			}))
			if !fired {
				t.Fatal("retry commit was not interleaved between the two progress reads")
			}
			if !status.Running || status.State != "queued" || status.Failed != 1 {
				t.Fatalf("mixed snapshots would prematurely stop Web polling: %+v", status)
			}
			if page && (len(status.Items) != 1 || status.Items[0].Status != "failed") {
				t.Fatalf("progress page does not belong to the summary snapshot: %+v", status.Items)
			}
			latest, err := (Queue{DB: db}).Status(context.Background(), sender, id)
			reviewMust(t, err)
			if !latest.Running || latest.Pending != 1 || latest.Failed != 0 {
				t.Fatalf("the independent retry commit was not visible to the next read: %+v", latest)
			}
			t.Log("last retry page committed between aggregate/retry reads; explicit RR preserved running state and current-page consistency")
		})
	}
}
