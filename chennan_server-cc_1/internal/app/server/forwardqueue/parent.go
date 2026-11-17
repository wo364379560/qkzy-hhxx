package forwardqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const TargetChunkSize = 200
const MessageChunkSize = 100

var ErrChunk = errors.New("上传分段无效，请从服务端已确认的位置继续")
var ErrNotReady = errors.New("任务内容尚未上传完整")

type Manifest struct {
	RequestID    string `json:"request_id"`
	ContentHash  string `json:"content_hash"`
	TargetCount  int64  `json:"target_count"`
	MessageCount int64  `json:"message_count"`
}
type TargetChunk struct {
	Offset  int64    `json:"offset"`
	Targets []Target `json:"targets"`
}
type MessageChunk struct {
	Offset   int64     `json:"offset"`
	Messages []Message `json:"messages"`
}

func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func totalFor(targets, messages int64) (int64, error) {
	// Representation safety only, not a business quota (JSON consumers use doubles).
	if targets <= 0 || messages <= 0 || targets > 9007199254740991/messages || targets > math.MaxInt64/messages {
		return 0, ErrInvalid
	}
	return targets * messages, nil
}
func activeSender(tx *gorm.DB, sender string) error {
	var user model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id,status").Where("im_user_id = ?", sender).First(&user).Error; err != nil {
		return err
	}
	if user.Status != "active" {
		return ErrInvalid
	}
	return nil
}
func ownedJob(tx *gorm.DB, sender, id string, lock bool) (*model.ForwardJob, error) {
	var job model.ForwardJob
	if lock {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := tx.Where("id = ? AND sender = ?", jobID(sender, id), sender).First(&job).Error
	return &job, err
}

func (q Queue) Init(ctx context.Context, sender string, req Manifest) (*Snapshot, error) {
	if !requestIDPattern.MatchString(req.RequestID) || len(req.ContentHash) != 64 {
		return nil, ErrInvalid
	}
	if _, err := hex.DecodeString(req.ContentHash); err != nil {
		return nil, ErrInvalid
	}
	if _, err := totalFor(req.TargetCount, req.MessageCount); err != nil {
		return nil, err
	}
	digest := fingerprint(req)
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := activeSender(tx, sender); err != nil {
			return err
		}
		existing, err := ownedJob(tx, sender, req.RequestID, false)
		if err == nil {
			if existing.Protocol != 2 || existing.Fingerprint != digest {
				return ErrConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&model.ForwardJob{ID: jobID(sender, req.RequestID), Sender: sender, RequestID: req.RequestID, Fingerprint: digest, Protocol: 2, State: "uploading", TargetCount: req.TargetCount, MessageCount: req.MessageCount}).Error
	})
	if err != nil {
		return nil, err
	}
	return q.Status(ctx, sender, req.RequestID)
}

func (q Queue) UploadTargets(ctx context.Context, sender, id string, c TargetChunk) (*Snapshot, error) {
	if c.Offset < 0 || len(c.Targets) == 0 || len(c.Targets) > TargetChunkSize {
		return nil, ErrChunk
	}
	seen := map[string]bool{}
	for _, t := range c.Targets {
		if !validTarget(t) || seen[t.ConversationID] {
			return nil, ErrInvalid
		}
		seen[t.ConversationID] = true
	}
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := ownedJob(tx, sender, id, true)
		if err != nil {
			return err
		}
		if job.Protocol != 2 || job.State == "cancelled" || c.Offset > job.TargetCount-int64(len(c.Targets)) {
			return ErrChunk
		}
		if c.Offset < job.UploadedTargets {
			if c.Offset+int64(len(c.Targets)) > job.UploadedTargets {
				return ErrConflict
			}
			var rows []model.ForwardTarget
			if err := tx.Where("job_id = ? AND ordinal >= ? AND ordinal < ?", job.ID, c.Offset, c.Offset+int64(len(c.Targets))).Order("ordinal").Find(&rows).Error; err != nil {
				return err
			}
			if len(rows) != len(c.Targets) {
				return ErrConflict
			}
			for i, row := range rows {
				if row.ConversationID != c.Targets[i].ConversationID || row.Name != short(c.Targets[i].Name, 128) {
					return ErrConflict
				}
			}
			return nil
		}
		if job.State != "uploading" || c.Offset != job.UploadedTargets {
			return ErrChunk
		}
		rows := make([]model.ForwardTarget, 0, len(c.Targets))
		for i, t := range c.Targets {
			rows = append(rows, model.ForwardTarget{JobID: job.ID, Ordinal: c.Offset + int64(i), Identity: fingerprint(t.ConversationID), ConversationID: t.ConversationID, Name: short(t.Name, 128)})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		return tx.Model(job).Update("uploaded_targets", job.UploadedTargets+int64(len(rows))).Error
	})
	if err != nil {
		return nil, err
	}
	return q.Status(ctx, sender, id)
}

func (q Queue) UploadMessages(ctx context.Context, sender, id string, c MessageChunk) (*Snapshot, error) {
	if c.Offset < 0 || len(c.Messages) == 0 || len(c.Messages) > MessageChunkSize {
		return nil, ErrChunk
	}
	for _, m := range c.Messages {
		if err := validMessage(m); err != nil {
			return nil, err
		}
	}
	rows := make([]model.ForwardMessage, 0, len(c.Messages))
	for i, m := range c.Messages {
		b, _ := json.Marshal(m.Body)
		rows = append(rows, model.ForwardMessage{JobID: jobID(sender, id), Ordinal: c.Offset + int64(i), Fingerprint: fingerprint(m), Body: string(b), Preview: short(m.Preview, 200)})
	}
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := ownedJob(tx, sender, id, true)
		if err != nil {
			return err
		}
		if job.Protocol != 2 || job.State == "cancelled" || c.Offset > job.MessageCount-int64(len(rows)) {
			return ErrChunk
		}
		if c.Offset < job.UploadedMessages {
			if c.Offset+int64(len(rows)) > job.UploadedMessages {
				return ErrConflict
			}
			var old []model.ForwardMessage
			if err := tx.Where("job_id = ? AND ordinal >= ? AND ordinal < ?", job.ID, c.Offset, c.Offset+int64(len(rows))).Order("ordinal").Find(&old).Error; err != nil {
				return err
			}
			if len(old) != len(rows) {
				return ErrConflict
			}
			for i, m := range old {
				if m.Fingerprint != rows[i].Fingerprint {
					return ErrConflict
				}
			}
			return nil
		}
		if job.State != "uploading" || c.Offset != job.UploadedMessages {
			return ErrChunk
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		return tx.Model(job).Update("uploaded_messages", job.UploadedMessages+int64(len(rows))).Error
	})
	if err != nil {
		return nil, err
	}
	return q.Status(ctx, sender, id)
}

func (q Queue) Commit(ctx context.Context, sender, id string) (*Snapshot, error) {
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := activeSender(tx, sender); err != nil {
			return err
		}
		job, err := ownedJob(tx, sender, id, true)
		if err != nil {
			return err
		}
		if job.Protocol != 2 || job.State == "cancelled" {
			return ErrConflict
		}
		if job.UploadedTargets != job.TargetCount || job.UploadedMessages != job.MessageCount {
			return ErrNotReady
		}
		if job.State == "uploading" {
			return tx.Model(job).Updates(map[string]any{"state": "queued", "fill_at": gorm.Expr("CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS SIGNED)")}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return q.Status(ctx, sender, id)
}

// MaterializePending applies queue-wide backpressure without rejecting a user's
// task. A shared DB lock and per-parent cursor make restart/multiple processes
// safe. FillAt rotates parents instead of letting the largest task monopolize IO.
func (q Queue) MaterializePending(ctx context.Context) error {
	cfg, err := LoadPacingSettings()
	if err != nil {
		return err
	}
	return q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		gate := model.ForwardGate{ID: "materialize:v2"}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&gate).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&gate).Error; err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&model.ForwardDelivery{}).Where("status IN ?", []string{"pending", "preparing", "sending", "retrying"}).Count(&active).Error; err != nil {
			return err
		}
		room := int64(cfg.MaterializeHighWater) - active
		if room <= 0 {
			return nil
		}
		var job model.ForwardJob
		// First dispatch and manual retries participate in the same persistent
		// parent rotation. An older retry cannot monopolize every materializer
		// tick while newly submitted parents wait, including after restart.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(`(protocol = 2 AND state = 'queued' AND materialized < target_count * message_count) OR EXISTS (SELECT 1 FROM forward_retries r WHERE r.job_id=forward_jobs.id AND r.done=?)`, false).Order("fill_at,id").First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		// DB time and a monotonic tie-breaker make fairness independent of
		// application-server clock differences or multiple ticks in one ms.
		var now int64
		if err := tx.Raw("SELECT CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS SIGNED)").Scan(&now).Error; err != nil {
			return err
		}
		fillAt := max(now, gate.NextAt+1)
		if err := tx.Model(&gate).Update("next_at", fillAt).Error; err != nil {
			return err
		}
		if err := tx.Model(&job).Update("fill_at", fillAt).Error; err != nil {
			return err
		}
		released, err := releaseRetry(tx, &job, min(room, int64(cfg.MaterializeBatch)))
		if err != nil || released {
			return err
		}
		// A legacy/completed parent may have been selected only for retry work.
		// Never fall through to Cartesian expansion without valid new input.
		if job.Protocol != 2 || job.State != "queued" || job.MessageCount <= 0 || job.Materialized >= job.TargetCount*job.MessageCount {
			return nil
		}
		total := job.TargetCount * job.MessageCount
		n := min(room, int64(cfg.MaterializeBatch), total-job.Materialized)
		firstTarget, lastTarget := job.Materialized/job.MessageCount, (job.Materialized+n-1)/job.MessageCount
		var targets []model.ForwardTarget
		if err := tx.Where("job_id = ? AND ordinal BETWEEN ? AND ?", job.ID, firstTarget, lastTarget).Order("ordinal").Find(&targets).Error; err != nil {
			return err
		}
		targetMap := map[int64]model.ForwardTarget{}
		for _, t := range targets {
			targetMap[t.Ordinal] = t
		}
		firstMessage, lastMessage := job.Materialized%job.MessageCount, (job.Materialized+n-1)%job.MessageCount
		var messages []model.ForwardMessage
		query := tx.Where("job_id = ?", job.ID)
		if firstTarget == lastTarget {
			query = query.Where("ordinal BETWEEN ? AND ?", firstMessage, lastMessage)
		} else if lastTarget == firstTarget+1 {
			query = query.Where("ordinal >= ? OR ordinal <= ?", firstMessage, lastMessage)
		}
		if err := query.Order("ordinal").Find(&messages).Error; err != nil {
			return err
		}
		messageMap := map[int64]model.ForwardMessage{}
		for _, m := range messages {
			messageMap[m.Ordinal] = m
		}
		// A newly materialized suffix must inherit earlier failures/unknown state.
		blocked := map[string]bool{}
		var failed []struct{ ConversationID string }
		names := make([]string, 0, len(targets))
		for _, t := range targets {
			names = append(names, t.ConversationID)
		}
		if err := tx.Model(&model.ForwardDelivery{}).Select("DISTINCT conversation_id").Where("job_id = ? AND conversation_id IN ? AND status IN ?", job.ID, names, []string{"failed", "blocked", "unknown"}).Scan(&failed).Error; err != nil {
			return err
		}
		for _, f := range failed {
			blocked[f.ConversationID] = true
		}
		rows := make([]model.ForwardDelivery, 0, n)
		for cursor := job.Materialized; cursor < job.Materialized+n; cursor++ {
			t, ok := targetMap[cursor/job.MessageCount]
			if !ok {
				return ErrNotReady
			}
			ordinal := cursor % job.MessageCount
			m, ok := messageMap[ordinal]
			if !ok {
				return ErrNotReady
			}
			status, description := "pending", ""
			if blocked[t.ConversationID] {
				status, description = "blocked", "前序消息未完成，等待处理"
			}
			rows = append(rows, model.ForwardDelivery{JobID: job.ID, Sender: job.Sender, ConversationID: t.ConversationID, Name: t.Name, Ordinal: int(ordinal), Preview: m.Preview, Message: m.Body, Status: status, Error: description, Random: randomNumber(), Sequence: uint32(ordinal%math.MaxUint32 + 1)})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		return tx.Model(&job).Update("materialized", job.Materialized+n).Error
	})
}

// Retry only definitely rejected work and its blocked suffix. The request ID
// fences a lost HTTP acknowledgement; Random/Sequence and successful/unknown
// deliveries are never replaced. Cancellation remains terminal.
func (q Queue) RetryFailed(ctx context.Context, sender, id, retryID string) (*Snapshot, error) {
	if !requestIDPattern.MatchString(retryID) {
		return nil, ErrInvalid
	}
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := activeSender(tx, sender); err != nil {
			return err
		}
		job, err := ownedJob(tx, sender, id, true)
		if err != nil {
			return err
		}
		if job.State == "cancelled" || job.State == "uploading" {
			return ErrConflict
		}
		var retry model.ForwardRetry
		err = tx.Where("job_id = ? AND retry_id = ?", job.ID, retryID).First(&retry).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var upper uint64
		if err := tx.Model(&model.ForwardDelivery{}).Select("COALESCE(MAX(id),0)").Where("job_id = ?", job.ID).Scan(&upper).Error; err != nil {
			return err
		}
		return tx.Create(&model.ForwardRetry{JobID: job.ID, RetryID: retryID, UpperID: upper}).Error
	})
	if err != nil {
		return nil, err
	}
	return q.Status(ctx, sender, id)
}

// Releasing retries participates in exactly the same active-work budget as
// first dispatch. The upper ID/cursor prevents a lost acknowledgement or a
// second failure from making this one manual retry loop forever.
func releaseRetry(tx *gorm.DB, job *model.ForwardJob, limit int64) (bool, error) {
	var retry model.ForwardRetry
	// The parent selection is a current locking read. Match that visibility
	// even if an earlier aggregate established an older RR snapshot.
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("job_id = ? AND done = ?", job.ID, false).Order("created_at,retry_id").First(&retry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if job.State == "cancelled" {
		return true, tx.Model(&retry).Update("done", true).Error
	}
	var rows []model.ForwardDelivery
	// Select only eligible work, not pages of an arbitrarily long successful
	// history. The status index, fixed input window and bounded result preserve
	// both idempotency and progress when only a few deliveries need retrying.
	if err := tx.Table("forward_deliveries AS d").Clauses(clause.Locking{Strength: "UPDATE"}).Select("d.id").
		Where("d.job_id = ? AND d.id > ? AND d.id <= ? AND d.status IN ? AND d.updated_at <= ?", job.ID, retry.Cursor, retry.UpperID, []string{"failed", "blocked"}, retry.CreatedAt).
		Where("NOT EXISTS (SELECT 1 FROM forward_deliveries u WHERE u.job_id=d.job_id AND u.conversation_id=d.conversation_id AND u.ordinal<d.ordinal AND u.status='unknown')").
		Order("d.id").Limit(int(limit)).Find(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return true, tx.Model(&retry).Updates(map[string]any{"cursor": retry.UpperID, "done": true}).Error
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	if err := tx.Exec(`UPDATE forward_deliveries d LEFT JOIN forward_deliveries u ON u.job_id=d.job_id AND u.conversation_id=d.conversation_id AND u.ordinal<d.ordinal AND u.status='unknown' SET d.status='pending',d.attempts=0,d.next_at=0,d.claim='',d.lease_until=0,d.error='' WHERE d.job_id=? AND d.id IN ? AND d.status IN ('failed','blocked') AND d.updated_at<=? AND u.id IS NULL`, job.ID, ids, retry.CreatedAt).Error; err != nil {
		return false, err
	}
	cursor := rows[len(rows)-1].ID
	done := len(rows) < int(limit) || cursor >= retry.UpperID
	if done {
		cursor = retry.UpperID
	}
	return true, tx.Model(&retry).Updates(map[string]any{"cursor": cursor, "done": done}).Error
}
