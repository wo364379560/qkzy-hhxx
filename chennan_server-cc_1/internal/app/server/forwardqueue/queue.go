package forwardqueue

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"gorm.io/gorm"
)

const ProgressPageSize = 200
const leaseMS int64 = 90000

var requestIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,64}$`)
var ErrInvalid = errors.New("转发任务参数无效")
var ErrConflict = errors.New("任务标识已使用，请勿改变原任务内容")

type Target struct {
	ConversationID string `json:"conversation_id"`
	Name           string `json:"name"`
}
type Message struct {
	Body    []tencentim.MessageElement `json:"body"`
	Preview string                     `json:"preview"`
}
type Request struct {
	RequestID string    `json:"request_id"`
	Targets   []Target  `json:"targets"`
	Messages  []Message `json:"messages"`
}
type Snapshot struct {
	RequestID        string                  `json:"request_id"`
	Items            []model.ForwardDelivery `json:"items"`
	Protocol         int                     `json:"protocol"`
	State            string                  `json:"state"`
	Running          bool                    `json:"running"`
	Total            int64                   `json:"total"`
	Success          int64                   `json:"success"`
	Failed           int64                   `json:"failed"`
	Unknown          int64                   `json:"unknown"`
	Blocked          int64                   `json:"blocked"`
	Cancelled        int64                   `json:"cancelled"`
	Pending          int64                   `json:"pending"`
	RetryableCount   int64                   `json:"retryable_count"`
	TargetCount      int64                   `json:"target_count"`
	MessageCount     int64                   `json:"message_count"`
	UploadedTargets  int64                   `json:"uploaded_targets"`
	UploadedMessages int64                   `json:"uploaded_messages"`
	PageSize         int                     `json:"page_size,omitempty"`
	NextCursor       string                  `json:"next_cursor,omitempty"`
}
type Queue struct{ DB *gorm.DB }

func jobID(sender, id string) string {
	h := sha256.Sum256([]byte(sender + "\x00" + id))
	return hex.EncodeToString(h[:])
}
func randomID() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func randomNumber() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	n := binary.BigEndian.Uint32(b[:])
	if n == 0 {
		n = 1
	}
	return n
}
func short(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
func running(s string) bool {
	return s == "pending" || s == "preparing" || s == "sending" || s == "retrying"
}

func validTarget(t Target) bool {
	id := t.ConversationID
	return len(id) <= 128 && (strings.HasPrefix(id, "GROUP") && len(id) > 5 || strings.HasPrefix(id, "C2C") && len(id) > 3) && !strings.ContainsAny(id, "\r\n\x00") && !strings.Contains(id, "@TIM#SYSTEM")
}
func validMessage(m Message) error {
	if len(m.Body) != 1 {
		return ErrInvalid
	}
	// Validate normalized legacy data without changing the original fingerprint.
	if _, err := tencentim.PrepareForwardBody(m.Body); err != nil {
		return fmt.Errorf("%w：%v", ErrInvalid, err)
	}
	return nil
}
func Validate(r Request) error {
	if !requestIDPattern.MatchString(r.RequestID) {
		return ErrInvalid
	}
	if _, err := totalFor(int64(len(r.Targets)), int64(len(r.Messages))); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, t := range r.Targets {
		if !validTarget(t) || seen[t.ConversationID] {
			return ErrInvalid
		}
		seen[t.ConversationID] = true
	}
	for _, m := range r.Messages {
		if err := validMessage(m); err != nil {
			return err
		}
	}
	return nil
}

// Compatibility submission: bounded HTTP inputs use the same durable parent
// pipeline, never eagerly expand the targets x messages product.
func (q Queue) Create(ctx context.Context, sender string, req Request, paged ...bool) (*Snapshot, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}
	// Resume old already-created jobs with their original fingerprint.
	if job, err := ownedJob(q.DB.WithContext(ctx), sender, req.RequestID, false); err == nil && job.Protocol == 0 {
		if job.Fingerprint != fingerprint(req) {
			return nil, ErrConflict
		}
		return q.Get(ctx, sender, req.RequestID)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	manifest := Manifest{RequestID: req.RequestID, ContentHash: fingerprint(req), TargetCount: int64(len(req.Targets)), MessageCount: int64(len(req.Messages))}
	if _, err := q.Init(ctx, sender, manifest); err != nil {
		return nil, err
	}
	for offset := 0; offset < len(req.Targets); offset += TargetChunkSize {
		if _, err := q.UploadTargets(ctx, sender, req.RequestID, TargetChunk{Offset: int64(offset), Targets: req.Targets[offset:min(offset+TargetChunkSize, len(req.Targets))]}); err != nil {
			return nil, err
		}
	}
	for offset := 0; offset < len(req.Messages); offset += MessageChunkSize {
		if _, err := q.UploadMessages(ctx, sender, req.RequestID, MessageChunk{Offset: int64(offset), Messages: req.Messages[offset:min(offset+MessageChunkSize, len(req.Messages))]}); err != nil {
			return nil, err
		}
	}
	if _, err := q.Commit(ctx, sender, req.RequestID); err != nil {
		return nil, err
	}
	return q.GetPage(ctx, sender, req.RequestID, 0)
}

func (q Queue) statusFor(ctx context.Context, job *model.ForwardJob) (*Snapshot, error) {
	r := &Snapshot{RequestID: job.RequestID, Protocol: job.Protocol, Items: []model.ForwardDelivery{}, State: job.State, TargetCount: job.TargetCount, MessageCount: job.MessageCount, UploadedTargets: job.UploadedTargets, UploadedMessages: job.UploadedMessages}
	var counts []struct {
		Status string
		Count  int64
	}
	if err := q.DB.WithContext(ctx).Model(&model.ForwardDelivery{}).Select("status, COUNT(*) AS count").Where("job_id = ?", job.ID).Group("status").Scan(&counts).Error; err != nil {
		return nil, err
	}
	var stored, active int64
	for _, c := range counts {
		stored += c.Count
		if running(c.Status) {
			active += c.Count
		}
		switch c.Status {
		case "success":
			r.Success += c.Count
		case "failed":
			r.Failed += c.Count
		case "unknown":
			r.Unknown += c.Count
		case "blocked":
			r.Blocked += c.Count
		case "cancelled":
			r.Cancelled += c.Count
		}
	}
	r.Total = stored
	if job.Protocol == 2 {
		r.Total = job.TargetCount * job.MessageCount
	}
	unmaterialized := max(int64(0), r.Total-stored)
	if r.State == "cancelled" {
		r.Cancelled += unmaterialized
	} else {
		r.Pending += unmaterialized
	}
	r.Pending += active
	r.Running = active > 0 || job.Protocol == 2 && job.State == "queued" && unmaterialized > 0
	if job.State != "cancelled" && job.State != "uploading" {
		var retrying int64
		if err := q.DB.WithContext(ctx).Model(&model.ForwardRetry{}).Where("job_id = ? AND done = ?", job.ID, false).Count(&retrying).Error; err != nil {
			return nil, err
		}
		r.Running = r.Running || retrying > 0
	}
	if job.Protocol == 0 {
		if r.Running {
			r.State = "queued"
		} else if job.State != "cancelled" {
			r.State = "completed"
		}
	}
	if r.State == "queued" && !r.Running {
		r.State = "completed"
	}
	if r.State != "cancelled" && r.State != "uploading" {
		if err := q.DB.WithContext(ctx).Table("forward_deliveries AS d").Where("d.job_id = ? AND d.status IN ?", job.ID, []string{"failed", "blocked"}).Where("NOT EXISTS (SELECT 1 FROM forward_deliveries u WHERE u.job_id=d.job_id AND u.conversation_id=d.conversation_id AND u.ordinal<d.ordinal AND u.status='unknown')").Count(&r.RetryableCount).Error; err != nil {
			return nil, err
		}
	}
	return r, nil
}
func (q Queue) Status(ctx context.Context, sender, id string) (*Snapshot, error) {
	return q.readSnapshot(ctx, sender, id, nil)
}
func (q Queue) Get(ctx context.Context, sender, id string) (*Snapshot, error) {
	return q.GetPage(ctx, sender, id, 0)
}
func (q Queue) GetPage(ctx context.Context, sender, id string, after uint64) (*Snapshot, error) {
	return q.readSnapshot(ctx, sender, id, &after)
}

// A retry can atomically release its last page and mark its cursor done between
// the delivery count and retry count. Read both (and the parent/page) from one
// snapshot, or the Web could see completed while the retry is actively sending.
// Explicit isolation also covers installations whose default is READ COMMITTED.
func (q Queue) readSnapshot(ctx context.Context, sender, id string, after *uint64) (*Snapshot, error) {
	var result *Snapshot
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := ownedJob(tx, sender, id, false)
		if err != nil {
			return err
		}
		result, err = (Queue{DB: tx}).statusFor(ctx, job)
		if err != nil || after == nil {
			return err
		}
		result.PageSize = ProgressPageSize
		if err := tx.Omit("message").Where("job_id = ? AND id > ?", job.ID, *after).Order("id").Limit(ProgressPageSize + 1).Find(&result.Items).Error; err != nil {
			return err
		}
		if len(result.Items) > ProgressPageSize {
			result.Items = result.Items[:ProgressPageSize]
			result.NextCursor = strconv.FormatUint(result.Items[len(result.Items)-1].ID, 10)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func (q Queue) Cancel(ctx context.Context, sender, id string) error {
	return q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := ownedJob(tx, sender, id, true)
		if err != nil {
			return err
		}
		if err := tx.Model(job).Update("state", "cancelled").Error; err != nil {
			return err
		}
		return tx.Model(&model.ForwardDelivery{}).Where("job_id = ? AND sender = ? AND status IN ?", job.ID, sender, []string{"pending", "retrying", "preparing", "blocked"}).Updates(map[string]any{"status": "cancelled", "error": "已取消未发送消息"}).Error
	})
}

// Claim only the first unfinished message per target. CAS protects concurrent
// workers without requiring MySQL 8 SKIP LOCKED support.
func (q Queue) claim(ctx context.Context) (*model.ForwardDelivery, error) {
	now := time.Now().UnixMilli()
	var items []model.ForwardDelivery
	err := q.DB.WithContext(ctx).Table("forward_deliveries AS d").Select("d.*").
		Where("d.status IN ? AND d.next_at <= ?", []string{"pending", "retrying"}, now).
		Where("NOT EXISTS (SELECT 1 FROM forward_deliveries p WHERE p.job_id=d.job_id AND p.conversation_id=d.conversation_id AND p.ordinal<d.ordinal AND p.status NOT IN ('success','cancelled'))").Order("d.next_at, d.id").Limit(8).Find(&items).Error
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		claim := randomID()
		r := q.DB.WithContext(ctx).Model(&model.ForwardDelivery{}).Where("id = ? AND status IN ?", item.ID, []string{"pending", "retrying"}).Updates(map[string]any{"status": "preparing", "claim": claim, "lease_until": now + leaseMS})
		if r.Error != nil {
			return nil, r.Error
		}
		if r.RowsAffected == 1 {
			item.Claim = claim
			item.Status = "preparing"
			return &item, nil
		}
	}
	return nil, nil
}

func (q Queue) recoverExpired(ctx context.Context) error {
	now := time.Now().UnixMilli()
	// A crashed process may already have delivered its sending item. Never replay it.
	if err := q.DB.WithContext(ctx).Model(&model.ForwardDelivery{}).Where("status = 'sending' AND lease_until < ?", now).Updates(map[string]any{"status": "unknown", "error": "服务中断时正在发送，请核对目标会话"}).Error; err != nil {
		return err
	}
	if err := q.DB.WithContext(ctx).Model(&model.ForwardDelivery{}).Where("status = 'preparing' AND lease_until < ?", now).Updates(map[string]any{"status": "pending", "claim": ""}).Error; err != nil {
		return err
	}
	return q.DB.WithContext(ctx).Exec("UPDATE forward_deliveries d JOIN forward_deliveries p ON p.job_id=d.job_id AND p.conversation_id=d.conversation_id AND p.ordinal<d.ordinal SET d.status='blocked', d.error=? WHERE p.status IN ('failed','blocked','unknown') AND d.status IN ('pending','retrying')", "前序消息未完成，后续转发等待处理").Error
}
