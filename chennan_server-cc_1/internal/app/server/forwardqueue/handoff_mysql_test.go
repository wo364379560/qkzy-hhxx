package forwardqueue

import (
	"context"
	"fmt"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"strings"
	"testing"
)

func TestHandoff3101Targets101MessagesRemainBoundedMySQL(t *testing.T) {
	db := reviewDB(t)
	clearPacingEnv(t)
	reviewMust(t, db.AutoMigrate(&model.ForwardJob{}, &model.ForwardTarget{}, &model.ForwardMessage{}, &model.ForwardDelivery{}, &model.ForwardRetry{}, &model.ForwardGate{}))
	reviewClear(t, db)
	q := Queue{DB: db}
	ctx := context.Background()
	const owner = "handoff-large-plan"
	const request = "handoff-large-plan-20261101"
	reviewMust(t, db.AutoMigrate(&reviewUser{}))
	reviewMust(t, db.Where("im_user_id = ?", owner).FirstOrCreate(&reviewUser{IMUserID: owner, Nickname: "Rui review", Status: "active"}).Error)
	_, err := q.Init(ctx, owner, Manifest{RequestID: request, ContentHash: strings.Repeat("a", 64), TargetCount: 3101, MessageCount: 101})
	reviewMust(t, err)
	targets := make([]Target, 3101)
	for i := range targets {
		targets[i] = Target{ConversationID: fmt.Sprintf("GROUPhandoff_%d", i), Name: fmt.Sprint(i)}
	}
	for offset := 0; offset < len(targets); offset += 200 {
		end := min(offset+200, len(targets))
		_, err = q.UploadTargets(ctx, owner, request, TargetChunk{Offset: int64(offset), Targets: targets[offset:end]})
		reviewMust(t, err)
	}
	messages := reviewMessages(101)
	for offset := 0; offset < len(messages); offset += 100 {
		end := min(offset+100, len(messages))
		_, err = q.UploadMessages(ctx, owner, request, MessageChunk{Offset: int64(offset), Messages: messages[offset:end]})
		reviewMust(t, err)
	}
	snapshot, err := q.Commit(ctx, owner, request)
	reviewMust(t, err)
	if snapshot.Total != 3101*101 {
		t.Fatalf("plan lost deliveries: %d", snapshot.Total)
	}
	reviewMust(t, q.MaterializePending(ctx))
	var count int64
	reviewMust(t, db.Model(&model.ForwardDelivery{}).Where("job_id = ?", jobID(owner, request)).Count(&count).Error)
	if count <= 0 || count > int64(CurrentSettings().MaterializeHighWater) {
		t.Fatalf("Cartesian product was eagerly expanded: %d", count)
	}
	page, err := q.GetPage(ctx, owner, request, 0)
	reviewMust(t, err)
	if len(page.Items) > ProgressPageSize {
		t.Fatalf("unbounded result page: %d", len(page.Items))
	}
	if _, err = q.Status(ctx, "another-account", request); err == nil {
		t.Fatal("another account read this job")
	}
}
