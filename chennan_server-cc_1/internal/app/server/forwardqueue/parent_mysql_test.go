package forwardqueue

// This suite deliberately requires an explicit, disposable database. It never
// reads .env or the application's configured database and never sends to Tencent.
// Run only with FORWARD_REVIEW_MYSQL_DSN=root@tcp(127.0.0.1:33326)/forward_review?parseTime=true.
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type reviewLegacyJob struct {
	ID          string `gorm:"size:64;primaryKey"`
	Sender      string `gorm:"size:64;index;not null"`
	RequestID   string `gorm:"size:64;not null"`
	Fingerprint string `gorm:"size:64;not null"`
	CreatedAt   time.Time
}

func (reviewLegacyJob) TableName() string { return "forward_jobs" }

type reviewUser struct {
	ID       uint   `gorm:"primaryKey"`
	IMUserID string `gorm:"size:64;uniqueIndex"`
	Nickname string
	Status   string
	OwnerID  uint
}

func (reviewUser) TableName() string { return "users" }

type reviewMaterializeContextKey struct{}

func reviewDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("FORWARD_REVIEW_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set FORWARD_REVIEW_MYSQL_DSN to the isolated localhost review database")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid review database DSN")
	}
	// Hard fail before connecting or dropping anything if a different DB is used.
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:33326" || cfg.DBName != "forward_review" {
		t.Fatal("review tests only allow 127.0.0.1:33326/forward_review")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("isolated review database connection failed: %v", err)
	}
	if err := db.AutoMigrate(&model.ChatKeyword{}); err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = raw.Close() })
	return db
}

func reviewMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func reviewClear(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, table := range []string{"forward_retries", "forward_deliveries", "forward_targets", "forward_messages", "forward_jobs", "forward_gates"} {
		reviewMust(t, db.Exec("DELETE FROM "+table).Error)
	}
}

func reviewMessages(n int) []Message {
	messages := make([]Message, n)
	for i := range messages {
		body, _ := json.Marshal(map[string]string{"Text": fmt.Sprintf("isolated-review-%d", i)})
		messages[i] = Message{Preview: fmt.Sprintf("message-%d", i), Body: []tencentim.MessageElement{{MsgType: "TIMTextElem", MsgContent: body}}}
	}
	return messages
}

func reviewCreate(t *testing.T, q Queue, id string, targets []Target, messages []Message) {
	t.Helper()
	_, err := q.Create(context.Background(), "review_sender", Request{RequestID: id, Targets: targets, Messages: messages})
	reviewMust(t, err)
}

func reviewRows(t *testing.T, db *gorm.DB, id string) []model.ForwardDelivery {
	t.Helper()
	var rows []model.ForwardDelivery
	reviewMust(t, db.Where("job_id = ?", jobID("review_sender", id)).Order("id").Find(&rows).Error)
	return rows
}

func reviewHistoricalParent(t *testing.T, db *gorm.DB, id string, n int, amend func(int, *model.ForwardDelivery)) []model.ForwardDelivery {
	t.Helper()
	old := time.Now().Add(-time.Hour)
	job := model.ForwardJob{ID: jobID("review_sender", id), Sender: "review_sender", RequestID: id, Fingerprint: fingerprint(id), Protocol: 2, State: "queued", TargetCount: int64(n / 2), MessageCount: 2, UploadedTargets: int64(n / 2), UploadedMessages: 2, Materialized: int64(n), FillAt: 1}
	reviewMust(t, db.Create(&job).Error)
	rows := make([]model.ForwardDelivery, n)
	for i := range rows {
		rows[i] = model.ForwardDelivery{JobID: job.ID, Sender: "review_sender", ConversationID: fmt.Sprintf("C2Chistorical_%d", i/2), Ordinal: i % 2, Message: `[{"MsgType":"TIMTextElem","MsgContent":{"Text":"history"}}]`, Status: "success", Random: uint32(i + 1), Sequence: uint32(i%2 + 1), CreatedAt: old, UpdatedAt: old}
		amend(i, &rows[i])
	}
	reviewMust(t, db.CreateInBatches(&rows, 500).Error)
	return rows
}

func TestForwardQueueMySQLReviewIntegration(t *testing.T) {
	db := reviewDB(t)
	clearPacingEnv(t)
	t.Setenv("FORWARD_MATERIALIZE_HIGH_WATER", "300")
	t.Setenv("FORWARD_MATERIALIZE_BATCH", "200")
	ctx := context.Background()
	q := Queue{DB: db}

	t.Run("migrate_legacy_three_tables_without_losing_rows", func(t *testing.T) {
		for _, table := range []string{"forward_retries", "forward_deliveries", "forward_targets", "forward_messages", "forward_jobs", "forward_gates", "users"} {
			reviewMust(t, db.Migrator().DropTable(table))
		}
		reviewMust(t, db.AutoMigrate(&reviewLegacyJob{}, &model.ForwardDelivery{}, &model.ForwardGate{}))
		legacy := reviewLegacyJob{ID: jobID("review_sender", "legacy-review-0001"), Sender: "review_sender", RequestID: "legacy-review-0001", Fingerprint: fingerprint("legacy")}
		reviewMust(t, db.Create(&legacy).Error)
		reviewMust(t, db.Create(&model.ForwardDelivery{JobID: legacy.ID, Sender: "review_sender", ConversationID: "C2Clegacy", Status: "success", Message: "[]", Random: 77, Sequence: 2, Result: "legacy-ack"}).Error)
		reviewMust(t, db.AutoMigrate(&model.ForwardJob{}, &model.ForwardTarget{}, &model.ForwardMessage{}, &model.ForwardDelivery{}, &model.ForwardGate{}, &model.ForwardRetry{}, &reviewUser{}))
		reviewMust(t, db.Create(&reviewUser{ID: 1, IMUserID: "review_sender", Nickname: "Review", Status: "active"}).Error)
		var migrated model.ForwardJob
		reviewMust(t, db.First(&migrated, "id = ?", legacy.ID).Error)
		if migrated.Protocol != 0 || migrated.Fingerprint != legacy.Fingerprint {
			t.Fatalf("migration changed legacy identity: %+v", migrated)
		}
		status, err := q.Get(ctx, "review_sender", legacy.RequestID)
		reviewMust(t, err)
		if status.Total != 1 || status.Success != 1 || status.Items[0].Result != "legacy-ack" {
			t.Fatalf("legacy progress lost: %+v", status)
		}
		t.Log("legacy jobs/deliveries/gates migrated; acknowledged delivery retained")
	})

	t.Run("4301_targets_times_7_chunk_resume_backpressure_concurrency_pagination", func(t *testing.T) {
		reviewClear(t, db)
		const id = "large-review-0000001"
		targets := make([]Target, 4301)
		for i := range targets {
			targets[i] = Target{ConversationID: fmt.Sprintf("C2Creview_%04d", i), Name: fmt.Sprintf("target-%d", i)}
		}
		messages := reviewMessages(7)
		manifest := Manifest{RequestID: id, ContentHash: fingerprint(Request{RequestID: id, Targets: targets, Messages: messages}), TargetCount: 4301, MessageCount: 7}
		_, err := q.Init(ctx, "review_sender", manifest)
		reviewMust(t, err)
		_, err = q.Init(ctx, "review_sender", manifest) // Lost init acknowledgement.
		reviewMust(t, err)
		_, err = q.Commit(ctx, "review_sender", id)
		if !errors.Is(err, ErrNotReady) {
			t.Fatalf("incomplete input committed: %v", err)
		}
		for offset := 0; offset < len(targets); offset += TargetChunkSize {
			chunk := TargetChunk{Offset: int64(offset), Targets: targets[offset:min(offset+TargetChunkSize, len(targets))]}
			_, err = q.UploadTargets(ctx, "review_sender", id, chunk)
			reviewMust(t, err)
			if offset == 0 {
				_, err = q.UploadTargets(ctx, "review_sender", id, chunk) // Lost chunk acknowledgement.
				reviewMust(t, err)
				changed := TargetChunk{Targets: []Target{{ConversationID: chunk.Targets[0].ConversationID, Name: "changed"}}}
				_, err = q.UploadTargets(ctx, "review_sender", id, changed)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("changed acknowledged target accepted: %v", err)
				}
			}
		}
		_, err = q.UploadMessages(ctx, "review_sender", id, MessageChunk{Messages: messages})
		reviewMust(t, err)
		_, err = q.UploadMessages(ctx, "review_sender", id, MessageChunk{Messages: messages})
		reviewMust(t, err)
		_, err = q.Commit(ctx, "review_sender", id)
		reviewMust(t, err)
		_, err = q.Commit(ctx, "review_sender", id)
		reviewMust(t, err)
		status, err := q.Status(ctx, "review_sender", id)
		reviewMust(t, err)
		if status.Total != 30107 || status.Pending != 30107 || !status.Running {
			t.Fatalf("parent volume/restart state invalid: %+v", status)
		}
		var n int64
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Count(&n).Error)
		if n != 0 {
			t.Fatal("commit eagerly expanded target/message Cartesian product")
		}
		var wg sync.WaitGroup
		errs := make(chan error, 12)
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); errs <- (Queue{DB: db}).MaterializePending(ctx) }()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			reviewMust(t, err)
		}
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Count(&n).Error)
		if n != 300 {
			t.Fatalf("shared high-water breached/underfilled: %d", n)
		}
		page, err := q.GetPage(ctx, "review_sender", id, 0)
		reviewMust(t, err)
		if len(page.Items) != 200 || page.NextCursor == "" || page.Total != 30107 {
			t.Fatalf("bounded progress invalid: %+v", page)
		}
		for _, row := range page.Items {
			if row.Message != "" {
				t.Fatal("progress response included stored message bodies")
			}
		}
		after, err := strconv.ParseUint(page.NextCursor, 10, 64)
		reviewMust(t, err)
		page2, err := q.GetPage(ctx, "review_sender", id, after)
		reviewMust(t, err)
		if len(page2.Items) != 100 || page2.NextCursor != "" || page2.Items[0].ID <= page.Items[199].ID {
			t.Fatal("cursor pagination skipped/duplicated items")
		}
		claimed := make(chan *model.ForwardDelivery, 64)
		errs = make(chan error, 64)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 8; j++ {
					row, err := (Queue{DB: db}).claim(ctx)
					errs <- err
					if row != nil {
						claimed <- row
					}
				}
			}()
		}
		wg.Wait()
		close(claimed)
		close(errs)
		for err := range errs {
			reviewMust(t, err)
		}
		seen := map[uint64]bool{}
		for row := range claimed {
			if seen[row.ID] || row.Ordinal != 0 || row.Claim == "" {
				t.Fatalf("concurrent claims duplicated or bypassed predecessor: %+v", row)
			}
			seen[row.ID] = true
		}
		if len(seen) == 0 {
			t.Fatal("no worker could claim materialized work")
		}
		// Acknowledge fixtures directly to test all 30,107 persisted Cartesian
		// rows without sending customer messages. The separate worker subtest
		// below executes processWithClient through its actual persistence path.
		for i := 0; i < 200; i++ {
			reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("job_id = ? AND status IN ?", jobID("review_sender", id), []string{"pending", "preparing"}).Updates(map[string]any{"status": "success", "result": "simulated-ack"}).Error)
			reviewMust(t, (Queue{DB: db}).MaterializePending(ctx)) // New worker instance resumes DB cursor.
			var job model.ForwardJob
			reviewMust(t, db.First(&job, "id = ?", jobID("review_sender", id)).Error)
			if job.Materialized == 30107 {
				break
			}
			if i == 199 {
				t.Fatal("parent expansion did not complete within bounded iterations")
			}
		}
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("job_id = ?", jobID("review_sender", id)).Update("status", "success").Error)
		reviewMust(t, db.Raw("SELECT COUNT(*) FROM (SELECT conversation_id, ordinal FROM forward_deliveries GROUP BY job_id, conversation_id, ordinal HAVING COUNT(*)>1) d").Scan(&n).Error)
		if n != 0 {
			t.Fatal("materialization duplicated delivery identities")
		}
		status, err = q.Status(ctx, "review_sender", id)
		reviewMust(t, err)
		if status.Success != 30107 || status.Pending != 0 || status.Running || status.State != "completed" {
			t.Fatalf("large parent did not finish: %+v", status)
		}
		t.Logf("30,107 rows expanded across bounded batches; 12 materializers respected 300 active maximum; %d unique concurrent claims; 200-item cursor pages", len(seen))
	})

	t.Run("worker_actual_persistence_retries_permissions_unknown_and_lost_ack", func(t *testing.T) {
		reviewClear(t, db)
		const id = "worker-review-00001"
		targets := []Target{{ConversationID: "C2Crate"}, {ConversationID: "C2Cfriend"}, {ConversationID: "GROUPmuted"}, {ConversationID: "C2Cunknown"}, {ConversationID: "C2Cok"}}
		reviewCreate(t, q, id, targets, reviewMessages(2))
		reviewMust(t, q.MaterializePending(ctx))
		before := reviewRows(t, db, id)
		client := &reviewForwardClient{calls: map[string]int{}, identities: map[string][][2]uint32{}}
		for i := 0; i < 6; i++ {
			item, err := q.claim(ctx)
			reviewMust(t, err)
			if item == nil {
				break
			}
			q.processWithClient(ctx, item, client)
		}
		rows := reviewRows(t, db, id)
		want := []string{"retrying", "pending", "failed", "blocked", "failed", "blocked", "unknown", "blocked", "success", "success"}
		for i, row := range rows {
			if row.Status != want[i] {
				t.Fatalf("worker row %d: got %s want %s", i, row.Status, want[i])
			}
		}
		if client.calls["GROUPmuted"] != 0 || client.calls["C2Cunknown"] != 1 {
			t.Fatal("mute bypass or uncertain dispatch replayed")
		}
		// Skip the already-asserted cooldown wall time; preserve real DB gates.
		reviewMust(t, db.Model(&model.ForwardGate{}).Where("id <> ?", "materialize:v2").Update("next_at", 0).Error)
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("status = 'retrying'").Update("next_at", 0).Error)
		for i := 0; i < 2; i++ {
			item, err := q.claim(ctx)
			reviewMust(t, err)
			if item == nil || item.ConversationID != "C2Crate" {
				t.Fatalf("rate-limited target did not retry in order: %+v", item)
			}
			q.processWithClient(ctx, item, client)
		}
		if len(client.identities["C2Crate"]) != 3 || client.identities["C2Crate"][0] != client.identities["C2Crate"][1] {
			t.Fatal("automatic retry changed Tencent deduplication identity")
		}
		_, err := q.RetryFailed(ctx, "review_sender", id, "retry-review-00001")
		reviewMust(t, err)
		for i := 0; i < 4; i++ {
			reviewMust(t, q.MaterializePending(ctx))
		}
		rows = reviewRows(t, db, id)
		for i, row := range rows {
			if row.Random != before[i].Random || row.Sequence != before[i].Sequence {
				t.Fatalf("manual retry replaced identity at row %d", i)
			}
		}
		if rows[0].Status != "success" || rows[6].Status != "unknown" || rows[7].Status != "blocked" || rows[2].Status != "pending" || rows[4].Status != "pending" {
			t.Fatalf("retry crossed successful/unknown fence: %+v", rows)
		}
		// Simulate a second rejection after the first retry request was accepted.
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("id = ?", rows[2].ID).Update("status", "failed").Error)
		_, err = q.RetryFailed(ctx, "review_sender", id, "retry-review-00001") // Lost retry ACK.
		reviewMust(t, err)
		reviewMust(t, q.MaterializePending(ctx))
		if got := reviewRows(t, db, id)[2].Status; got != "failed" {
			t.Fatalf("duplicate retry request restarted new attempt: %s", got)
		}
		t.Log("actual worker DB transition path: explicit limit retries, mute/friend refusal, success, unknown result, safe manual retry and repeated retry acknowledgement")
	})

	t.Run("manual_retry_uses_shared_backpressure_and_survives_restart", func(t *testing.T) {
		reviewClear(t, db)
		const id = "retry-bulk-review-01"
		targets := make([]Target, 501)
		for i := range targets {
			targets[i].ConversationID = fmt.Sprintf("C2Cretry%d", i)
		}
		reviewCreate(t, q, id, targets, reviewMessages(1))
		for i := 0; i < 3; i++ {
			reviewMust(t, q.MaterializePending(ctx))
			reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("job_id = ?", jobID("review_sender", id)).Update("status", "failed").Error)
		}
		before := reviewRows(t, db, id)
		if len(before) != 501 {
			t.Fatalf("retry fixture was not fully materialized: %d", len(before))
		}
		_, err := q.RetryFailed(ctx, "review_sender", id, "retry-bulk-action-01")
		reviewMust(t, err)
		var active int64
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("status = 'pending'").Count(&active).Error)
		if active != 0 {
			t.Fatal("manual retry eagerly reset the complete parent")
		}
		for i := 0; i < 5; i++ {
			reviewMust(t, (Queue{DB: db}).MaterializePending(ctx))
		}
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("status = 'pending'").Count(&active).Error)
		if active != 300 {
			t.Fatalf("manual retry bypassed shared active budget: %d", active)
		}
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("status = 'pending'").Update("status", "success").Error)
		for i := 0; i < 3; i++ {
			reviewMust(t, (Queue{DB: db}).MaterializePending(ctx))
		}
		after := reviewRows(t, db, id)
		var succeeded, pending int
		for i, row := range after {
			if row.ID != before[i].ID || row.Random != before[i].Random || row.Sequence != before[i].Sequence {
				t.Fatal("chunked retry replaced a delivery's idempotency identity")
			}
			if row.Status == "success" {
				succeeded++
			}
			if row.Status == "pending" {
				pending++
			}
		}
		if succeeded != 300 || pending != 201 {
			t.Fatalf("retry cursor skipped/repeated rows after restart: success=%d pending=%d", succeeded, pending)
		}
		t.Log("501 failed deliveries released as 200+100 then 200+1 under the same 300-active budget; Random/Sequence preserved")
	})

	t.Run("sparse_retry_skips_success_history_and_rotates_with_new_parents", func(t *testing.T) {
		reviewClear(t, db)
		const historyID = "sparse-history-0001"
		before := reviewHistoricalParent(t, db, historyID, 6004, func(i int, row *model.ForwardDelivery) {
			switch i {
			case 5998:
				row.Status = "failed"
			case 5999, 6001:
				row.Status = "blocked"
			case 6000:
				row.Status = "unknown"
			case 6002:
				row.Status = "failed"
				row.UpdatedAt = time.Now().Add(time.Hour) // Outside this retry's creation window.
			}
		})
		_, err := q.RetryFailed(ctx, "review_sender", historyID, "sparse-retry-000001")
		reviewMust(t, err)
		const freshID = "sparse-fresh-000001"
		reviewCreate(t, q, freshID, []Target{{ConversationID: "C2Cfresh"}}, reviewMessages(1))
		reviewMust(t, q.MaterializePending(ctx))
		rows := reviewRows(t, db, historyID)
		for i, row := range rows {
			want := before[i].Status
			if i == 5998 || i == 5999 {
				want = "pending"
			}
			if row.Status != want || row.Random != before[i].Random || row.Sequence != before[i].Sequence {
				t.Fatalf("sparse retry touched excluded history or missed eligible row %d: %s want %s", i, row.Status, want)
			}
		}
		var retry model.ForwardRetry
		reviewMust(t, db.First(&retry, "job_id = ? AND retry_id = ?", jobID("review_sender", historyID), "sparse-retry-000001").Error)
		if !retry.Done || retry.Cursor != retry.UpperID {
			t.Fatal("two sparse failures needed repeated ticks to scan 6004 historical rows")
		}
		reviewMust(t, (Queue{DB: db}).MaterializePending(ctx))
		if len(reviewRows(t, db, freshID)) != 1 {
			t.Fatal("fresh parent did not run immediately after sparse retry")
		}

		// A retry with many actual failures must also rotate; optimizing only
		// sparse historical scans is insufficient to protect new submissions.
		reviewClear(t, db)
		t.Setenv("FORWARD_MATERIALIZE_HIGH_WATER", "1000")
		t.Setenv("FORWARD_MATERIALIZE_BATCH", "100")
		const bulkID = "rotation-history-01"
		reviewHistoricalParent(t, db, bulkID, 600, func(_ int, row *model.ForwardDelivery) { row.Status = "failed" })
		_, err = q.RetryFailed(ctx, "review_sender", bulkID, "rotation-retry-0001")
		reviewMust(t, err)
		const nextID = "rotation-fresh-0001"
		targets := make([]Target, 400)
		for i := range targets {
			targets[i].ConversationID = fmt.Sprintf("C2Cnew_%d", i)
		}
		reviewCreate(t, q, nextID, targets, reviewMessages(1))
		for turn := 0; turn < 4; turn++ {
			reviewMust(t, (Queue{DB: db}).MaterializePending(ctx))
			var released int64
			reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("job_id = ? AND status = 'pending'", jobID("review_sender", bulkID)).Count(&released).Error)
			newRows := len(reviewRows(t, db, nextID))
			wantReleased := int64((turn/2 + 1) * 100)
			wantFresh := ((turn + 1) / 2) * 100
			if released != wantReleased || newRows != wantFresh {
				t.Fatalf("parent starvation at turn %d: retried=%d/%d fresh=%d/%d", turn, released, wantReleased, newRows, wantFresh)
			}
		}
		t.Log("6004-row history with two eligible failures finishes retry selection in one tick; 600 actual failures alternate 100-row turns with fresh parent across new worker instances")
	})

	t.Run("retry_committed_after_materializer_snapshot_is_current_read", func(t *testing.T) {
		reviewClear(t, db)
		const id = "legacy-late-retry-01"
		parentID := jobID("review_sender", id)
		reviewMust(t, db.Create(&model.ForwardJob{ID: parentID, Sender: "review_sender", RequestID: id, Fingerprint: fingerprint(id)}).Error)
		row := model.ForwardDelivery{JobID: parentID, Sender: "review_sender", ConversationID: "C2Clate_retry", Message: "[]", Status: "failed", Random: 55, Sequence: 1}
		reviewMust(t, db.Create(&row).Error)
		fired := false
		const callbackName = "review:retry_after_materializer_snapshot"
		reviewMust(t, db.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
			if fired || tx.Statement.Context.Value(reviewMaterializeContextKey{}) != true || tx.Statement.Table != "forward_deliveries" {
				return
			}
			fired = true
			// The active-count SELECT has established the materializer's old
			// snapshot. The parent locking SELECT can see this later commit, so
			// its retry lookup must also be a current read, including protocol 0.
			_, err := (Queue{DB: db}).RetryFailed(ctx, "review_sender", id, "legacy-late-action-01")
			if err != nil {
				tx.AddError(err)
			}
		}))
		t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })
		readCtx := context.WithValue(ctx, reviewMaterializeContextKey{}, true)
		reviewMust(t, q.MaterializePending(readCtx))
		if !fired {
			t.Fatal("retry insertion did not interleave after the materializer snapshot")
		}
		// MySQL may use the old snapshot for EXISTS inside the parent query,
		// deferring selection to the next tick. That is safe, but the next
		// transaction must resume this legacy job without attempting expansion.
		reviewMust(t, q.MaterializePending(ctx))
		if reviewRows(t, db, id)[0].Status != "pending" {
			t.Fatal("newly committed legacy retry did not resume by the next tick")
		}

		// Independently force selection of a locked parent after a new retry
		// commit, exercising the helper's current-read contract regardless of
		// the optimizer's visibility choice for the outer EXISTS subquery.
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("id = ?", row.ID).Update("status", "failed").Error)
		reviewMust(t, db.Transaction(func(tx *gorm.DB) error {
			var count int64
			if err := tx.Model(&model.ForwardRetry{}).Count(&count).Error; err != nil {
				return err
			}
			if _, err := q.RetryFailed(ctx, "review_sender", id, "legacy-late-action-02"); err != nil {
				return err
			}
			var parent model.ForwardJob
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&parent, "id = ?", parentID).Error; err != nil {
				return err
			}
			handled, err := releaseRetry(tx, &parent, 100)
			if err != nil {
				return err
			}
			if !handled {
				return errors.New("locked parent's newly committed retry was hidden by older snapshot")
			}
			return nil
		}))
		if reviewRows(t, db, id)[0].Status != "pending" {
			t.Fatal("current-read retry helper did not release the eligible legacy delivery")
		}
	})

	t.Run("mysql_gates_are_shared_but_permission_queries_are_separate", func(t *testing.T) {
		reviewClear(t, db)
		t.Setenv("FORWARD_C2C_SEND_RPS", "1")
		type reservation struct {
			wait time.Duration
			err  error
		}
		results := make(chan reservation, 12)
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				wait, err := (Queue{DB: db}).slot(ctx, &model.ForwardDelivery{Sender: fmt.Sprintf("sender%d", i), ConversationID: "C2Cone"})
				results <- reservation{wait, err}
			}(i)
		}
		wg.Wait()
		close(results)
		won := 0
		for result := range results {
			reviewMust(t, result.err)
			if result.wait == 0 {
				won++
			}
		}
		if won != 1 {
			t.Fatalf("multiple server instances did not share a command gate: %d immediate reservations", won)
		}
		wait, err := q.querySlot(ctx, "get_group_info")
		reviewMust(t, err)
		if wait != 0 {
			t.Fatal("C2C send budget incorrectly blocked the permission-read command")
		}
		wait, err = q.querySlot(ctx, "get_specified_group_member_info")
		reviewMust(t, err)
		if wait != 0 {
			t.Fatal("different permission-read commands incorrectly shared one rate bucket")
		}
	})

	t.Run("cancellation_stops_future_materialization_and_dispatch", func(t *testing.T) {
		reviewClear(t, db)
		const id = "cancel-review-00001"
		targets := make([]Target, 500)
		for i := range targets {
			targets[i].ConversationID = fmt.Sprintf("C2Ccancel%d", i)
		}
		reviewCreate(t, q, id, targets, reviewMessages(7))
		reviewMust(t, q.MaterializePending(ctx))
		item, err := q.claim(ctx)
		reviewMust(t, err)
		if item == nil {
			t.Fatal("no cancellable preparation")
		}
		reviewMust(t, q.Cancel(ctx, "review_sender", id))
		client := &reviewForwardClient{calls: map[string]int{}, identities: map[string][][2]uint32{}}
		q.processWithClient(ctx, item, client)
		for i := 0; i < 3; i++ {
			reviewMust(t, q.MaterializePending(ctx))
		}
		status, err := q.Status(ctx, "review_sender", id)
		reviewMust(t, err)
		if status.Cancelled != 3500 || status.Running || len(reviewRows(t, db, id)) != 200 || len(client.calls) != 0 {
			t.Fatalf("cancelled work resumed: %+v", status)
		}
		_, err = q.RetryFailed(ctx, "review_sender", id, "cancel-retry-00001")
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("cancelled task restarted through retry: %v", err)
		}
	})

	t.Run("expired_send_unknown_and_late_ack_fencing", func(t *testing.T) {
		reviewClear(t, db)
		const id = "recover-review-0001"
		reviewCreate(t, q, id, []Target{{ConversationID: "C2Crecover"}, {ConversationID: "C2Cprepare"}}, reviewMessages(2))
		reviewMust(t, q.MaterializePending(ctx))
		rows := reviewRows(t, db, id)
		expired := time.Now().Add(-time.Minute).UnixMilli()
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("id = ?", rows[0].ID).Updates(map[string]any{"status": "sending", "claim": "old-claim", "lease_until": expired}).Error)
		reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("id = ?", rows[2].ID).Updates(map[string]any{"status": "preparing", "claim": "old-preparation", "lease_until": expired}).Error)
		reviewMust(t, q.recoverExpired(ctx))
		rows = reviewRows(t, db, id)
		if rows[0].Status != "unknown" || rows[1].Status != "blocked" || rows[2].Status != "pending" || rows[2].Claim != "" {
			t.Fatalf("restart misclassified dispatch uncertainty: %+v", rows)
		}
		stale := rows[0]
		stale.Claim = "different-claim"
		q.finish(&stale, "wrong-ack", nil, true)
		if reviewRows(t, db, id)[0].Status != "unknown" {
			t.Fatal("stale claim overwrote unknown result")
		}
		q.finish(&rows[0], "late-confirmed-ack", nil, true)
		updated := reviewRows(t, db, id)[0]
		if updated.Status != "success" || updated.Result != "late-confirmed-ack" {
			t.Fatal("matching late acknowledgement could not confirm unknown result")
		}
	})
}

type reviewForwardClient struct {
	calls      map[string]int
	identities map[string][][2]uint32
}

func (c *reviewForwardClient) CheckForwardGroup(ctx context.Context, group, sender string, gates ...tencentim.ForwardQueryGate) (string, error) {
	if len(gates) > 0 {
		for _, command := range []string{"get_group_info", "get_specified_group_member_info"} {
			if err := gates[0](ctx, command); err != nil {
				return "", err
			}
		}
	}
	if group == "muted" {
		return "", &tencentim.DeliveryError{Code: 10017, Message: "muted test member", Command: "get_specified_group_member_info"}
	}
	return "review-group", nil
}

func (c *reviewForwardClient) SendForward(ctx context.Context, sender, conversation string, body []tencentim.MessageElement, random, sequence uint32, push *tencentim.OfflinePushInfo) (string, error) {
	c.calls[conversation]++
	c.identities[conversation] = append(c.identities[conversation], [2]uint32{random, sequence})
	switch conversation {
	case "C2Crate":
		if c.calls[conversation] == 1 {
			return "", &tencentim.DeliveryError{Code: 60018, Message: "rate test refusal", Command: "sendmsg"}
		}
	case "C2Cfriend":
		return "", &tencentim.DeliveryError{Code: 20009, Message: "not a friend", Command: "sendmsg"}
	case "C2Cunknown":
		return "", &tencentim.DeliveryError{Unknown: true, Message: "response lost"}
	}
	return fmt.Sprintf("test-ack-%s-%d", conversation, sequence), nil
}
