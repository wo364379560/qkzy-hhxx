package forwardqueue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

const legacyReviewImage = `[{"MsgType":"TIMImageElem","MsgContent":{"UUID":"review-image","ImageFormat":1,"ImageInfoArray":[{"Type":0,"Size":100,"Width":20,"Height":10,"URL":"https://media.invalid/review.jpg?token=not-public"},{"Type":1,"Size":10,"Width":2,"Height":1,"URL":"https://media.invalid/thumb.jpg"}]}}]`

type mediaReviewClient struct {
	queries, sends int
	errorNext      error
	identities     [][2]uint32
}

func (c *mediaReviewClient) CheckForwardGroup(context.Context, string, string, ...tencentim.ForwardQueryGate) (string, error) {
	c.queries++
	return "isolated-media-review", nil
}
func (c *mediaReviewClient) SendForward(_ context.Context, _, _ string, body []tencentim.MessageElement, random, sequence uint32, _ *tencentim.OfflinePushInfo) (string, error) {
	c.sends++
	c.identities = append(c.identities, [2]uint32{random, sequence})
	var p struct {
		UUID           string
		ImageInfoArray []struct {
			Type, Size, Width, Height int
			URL                       string
		}
	}
	if len(body) != 1 || json.Unmarshal(body[0].MsgContent, &p) != nil || p.UUID != "review-image" || len(p.ImageInfoArray) != 2 || p.ImageInfoArray[0].Type != 1 || p.ImageInfoArray[1].Type != 3 || p.ImageInfoArray[0].Width != 20 {
		return "", errors.New("test client received unnormalized/corrupted image")
	}
	return "review-ack", c.errorNext
}

func TestForwardMediaAdmissionWithoutDatabase(t *testing.T) {
	var body []tencentim.MessageElement
	reviewMust(t, json.Unmarshal([]byte(legacyReviewImage), &body))
	r := validRequest()
	r.Messages = []Message{{Body: body}}
	before, _ := json.Marshal(r)
	if err := Validate(r); err != nil {
		t.Fatal("complete old-Web message must remain compatible:", err)
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("admission changed source/fingerprint")
	}
	for _, raw := range []string{
		strings.Replace(legacyReviewImage, `"Width":20,`, "", 1),
		strings.Replace(legacyReviewImage, `"ImageFormat":1`, `"ImageFormat":"PNG"`, 1),
		strings.Replace(legacyReviewImage, `"Size":100`, `"Size":"100"`, 1),
	} {
		reviewMust(t, json.Unmarshal([]byte(raw), &body))
		r.Messages[0].Body = body
		if err := Validate(r); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "消息参数校验") || strings.Contains(err.Error(), "not-public") {
			t.Fatalf("unsafe or missing admission diagnostic: %v", err)
		}
	}
}

// Real disposable MySQL, isolated from application .env and Tencent. Exercises
// schema upgrade, durable diagnostics, old jobs, retries and worker dispatch.
func TestForwardMediaMigrationAndRetryMySQL(t *testing.T) {
	db := reviewDB(t)
	clearPacingEnv(t)
	ctx := context.Background()
	q := Queue{DB: db}
	reviewMust(t, db.AutoMigrate(&model.ForwardJob{}, &model.ForwardDelivery{}, &model.ForwardTarget{}, &model.ForwardMessage{}, &model.ForwardRetry{}, &model.ForwardGate{}, &reviewUser{}))
	reviewClear(t, db)
	reviewMust(t, db.Where("im_user_id = ?", "review_sender").FirstOrCreate(&reviewUser{IMUserID: "review_sender", Nickname: "Review", Status: "active"}).Error)
	columns := []string{"LastErrorCommand", "LastErrorCode", "LastErrorInfo", "LastTencentRequestID"}
	for _, column := range columns {
		reviewMust(t, db.Migrator().DropColumn(&model.ForwardDelivery{}, column))
	}
	const id = "media-legacy-review-001"
	job := model.ForwardJob{ID: jobID("review_sender", id), Sender: "review_sender", RequestID: id, Fingerprint: "original-fingerprint", Protocol: 2, State: "queued", TargetCount: 3, MessageCount: 1, UploadedTargets: 3, UploadedMessages: 1, Materialized: 3}
	reviewMust(t, db.Create(&job).Error)
	for i, status := range []string{"failed", "success", "unknown"} {
		row := model.ForwardDelivery{JobID: job.ID, Sender: "review_sender", ConversationID: []string{"GROUPretry", "GROUPsuccess", "GROUPunknown"}[i], Message: legacyReviewImage, Status: status, Random: uint32(71 + i), Sequence: 1, Error: "old diagnostic", Result: "prior-result"}
		reviewMust(t, db.Omit(columns...).Create(&row).Error)
	}
	reviewMust(t, db.AutoMigrate(&model.ForwardDelivery{}))
	before := reviewRows(t, db, id)
	if len(before) != 3 {
		t.Fatal("migration lost rows")
	}
	for _, row := range before {
		if row.Message != legacyReviewImage || row.LastErrorCommand != "" || row.LastErrorCode != 0 || row.LastTencentRequestID != "" {
			t.Fatal("additive migration changed legacy data")
		}
	}
	_, err := q.RetryFailed(ctx, "review_sender", id, "media-retry-original-001")
	reviewMust(t, err)
	_, err = q.RetryFailed(ctx, "review_sender", id, "media-retry-original-001")
	reviewMust(t, err)
	reviewMust(t, q.MaterializePending(ctx))
	item, err := q.claim(ctx)
	reviewMust(t, err)
	if item == nil || item.ID != before[0].ID {
		t.Fatal("retry selected successful or uncertain delivery")
	}
	client := &mediaReviewClient{errorNext: &tencentim.DeliveryError{Code: 10004, Command: "send_group_msg", Message: "群消息发送：腾讯拒绝请求（10004）：invalid Width", ErrorInfo: "invalid Width https://media.invalid/a?token=not-public", RequestID: "review-request-10004"}}
	q.processWithClient(ctx, item, client)
	failed := reviewRows(t, db, id)[0]
	if failed.Status != "failed" || failed.LastErrorCode != 10004 || failed.LastErrorCommand != "send_group_msg" || failed.LastTencentRequestID != "review-request-10004" || !strings.Contains(failed.LastErrorInfo, "invalid Width") || strings.Contains(failed.LastErrorInfo, "not-public") || strings.Contains(failed.LastErrorInfo, "https:") {
		t.Fatalf("diagnostic not persisted safely: %+v", failed)
	}
	public, _ := json.Marshal(failed)
	if strings.Contains(string(public), "review-request-10004") || strings.Contains(string(public), "not-public") {
		t.Fatal("private correlation or source URL exposed in progress")
	}
	_, err = q.RetryFailed(ctx, "review_sender", id, "media-retry-repaired-002")
	reviewMust(t, err)
	reviewMust(t, q.MaterializePending(ctx))
	item, err = q.claim(ctx)
	reviewMust(t, err)
	if item == nil {
		t.Fatal("failed item not released")
	}
	client.errorNext = nil
	q.processWithClient(ctx, item, client)
	after := reviewRows(t, db, id)
	if after[0].Status != "success" || after[0].Error != "" || after[0].LastTencentRequestID != "review-request-10004" || after[0].LastErrorCode != 10004 {
		t.Fatal("successful retry erased diagnostic or remained failed")
	}
	for i, row := range after {
		if row.Message != before[i].Message || row.Random != before[i].Random || row.Sequence != before[i].Sequence {
			t.Fatal("retry rewrote durable payload or message identity")
		}
	}
	if after[1].Status != "success" || after[2].Status != "unknown" || client.sends != 2 || client.identities[0] != client.identities[1] {
		t.Fatal("retry replayed success/unknown or replaced identity")
	}
	// A late callback from the old claim cannot overwrite success/diagnostics.
	q.finish(item, "", &tencentim.DeliveryError{Code: 10004, Command: "send_group_msg", RequestID: "late-request"}, true)
	if reviewRows(t, db, id)[0].LastTencentRequestID != "review-request-10004" {
		t.Fatal("late result overwrote latest accepted diagnostic")
	}
	_, err = q.RetryFailed(ctx, "review_sender", id, "media-retry-completed-003")
	reviewMust(t, err)
	reviewMust(t, q.MaterializePending(ctx))
	item, err = q.claim(ctx)
	reviewMust(t, err)
	if item != nil {
		t.Fatal("completed/unknown work was rescheduled")
	}

	// Old stored records bypassed admission; dispatch must still reject them.
	bad := strings.Replace(legacyReviewImage, `"Width":20,`, "", 1)
	const badID = "media-invalid-review-001"
	badJob := model.ForwardJob{ID: jobID("review_sender", badID), Sender: "review_sender", RequestID: badID, Fingerprint: "old-bad-image", Protocol: 2, State: "queued", TargetCount: 1, MessageCount: 1, Materialized: 1}
	reviewMust(t, db.Create(&badJob).Error)
	reviewMust(t, db.Create(&model.ForwardDelivery{JobID: badJob.ID, Sender: "review_sender", ConversationID: "GROUPbad", Message: bad, Status: "pending", Random: 90, Sequence: 1}).Error)
	item, err = q.claim(ctx)
	reviewMust(t, err)
	if item == nil {
		t.Fatal("bad legacy row missing")
	}
	queries, sends := client.queries, client.sends
	q.processWithClient(ctx, item, client)
	badRow := reviewRows(t, db, badID)[0]
	if badRow.Status != "failed" || badRow.LastErrorCode != 400 || badRow.LastErrorCommand != "validate_forward_message" || !strings.Contains(badRow.Error, "Width") || client.queries != queries || client.sends != sends || badRow.Message != bad {
		t.Fatal("invalid stored image reached a Tencent query/send or was silently rewritten")
	}

	// Admission rejects a malformed chunk atomically, before storing any part.
	const newID = "media-admission-review-001"
	_, err = q.Init(ctx, "review_sender", Manifest{RequestID: newID, ContentHash: strings.Repeat("a", 64), TargetCount: 1, MessageCount: 2})
	reviewMust(t, err)
	var badBody []tencentim.MessageElement
	reviewMust(t, json.Unmarshal([]byte(bad), &badBody))
	_, err = q.UploadMessages(ctx, "review_sender", newID, MessageChunk{Messages: append(reviewMessages(1), Message{Body: badBody})})
	if !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid chunk admitted")
	}
	var count int64
	reviewMust(t, db.Model(&model.ForwardMessage{}).Where("job_id = ?", jobID("review_sender", newID)).Count(&count).Error)
	if count != 0 {
		t.Fatal("invalid chunk partially stored")
	}
	t.Log("real MySQL: additive migration; both validation boundaries; preserved source and dedup identity; durable redacted Tencent diagnostics retained after successful retry; success/unknown/late-result fences")
}
