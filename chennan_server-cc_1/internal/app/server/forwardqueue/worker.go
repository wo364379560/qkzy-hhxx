package forwardqueue

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func pause(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func Start(ctx context.Context, db *gorm.DB) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("FORWARD_QUEUE_ENABLED")), "false") || !tencentim.NewClient().IsConfigured() {
		return
	}
	settings, err := LoadPacingSettings()
	if err != nil {
		logger.Warnf("forward queue configuration invalid: %s", err)
		return
	}
	q := Queue{DB: db}
	go func() {
		for ctx.Err() == nil {
			if err := q.MaterializePending(ctx); err != nil && ctx.Err() == nil {
				logger.Warnf("forward parent materialization failed")
			}
			if !pause(ctx, 500*time.Millisecond) {
				return
			}
		}
	}()
	go func() {
		for {
			if err := q.recoverExpired(ctx); err != nil && ctx.Err() == nil {
				logger.Warnf("forward queue recovery failed")
			}
			if !pause(ctx, 5*time.Second) {
				return
			}
		}
	}()
	for i := 0; i < settings.Workers; i++ {
		go func() {
			for ctx.Err() == nil {
				item, err := q.claim(ctx)
				if err != nil || item == nil {
					if !pause(ctx, 500*time.Millisecond) {
						return
					}
					continue
				}
				q.process(ctx, item)
			}
		}()
	}
}

func (q Queue) process(parent context.Context, item *model.ForwardDelivery) {
	q.processWithClient(parent, item, tencentim.NewClient())
}

type forwardClient interface {
	CheckForwardGroup(context.Context, string, string, ...tencentim.ForwardQueryGate) (string, error)
	SendForward(context.Context, string, string, []tencentim.MessageElement, uint32, uint32, *tencentim.OfflinePushInfo) (string, error)
}

// Keep the complete worker path testable against an isolated transport. The
// production worker always supplies Tencent's configured client above.
func (q Queue) processWithClient(parent context.Context, item *model.ForwardDelivery, client forwardClient) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	item.Attempts++
	var body []tencentim.MessageElement
	if json.Unmarshal([]byte(item.Message), &body) != nil {
		q.finish(item, "", &tencentim.DeliveryError{Code: 400, Command: "validate_forward_message", Message: "消息参数校验：已保存的消息不是有效JSON"}, false)
		return
	}
	// Inspect persisted legacy tasks too, before permission queries or dispatch.
	// Invalid metadata is a permanent local failure, never a transient retry.
	prepared, err := tencentim.PrepareForwardBody(body)
	if err != nil {
		q.finish(item, "", err, false)
		return
	}
	body = prepared
	originalBody := body
	var user model.User
	groupName := ""
	checkedAt := time.Time{}
	for {
		if time.Since(checkedAt) >= time.Second {
			if err := q.DB.WithContext(ctx).Select("id, im_user_id, nickname, status, owner_id").Where("im_user_id = ?", item.Sender).First(&user).Error; err != nil {
				q.finish(item, "", err, false)
				return
			}
			if user.Status != "active" {
				q.finish(item, "", &tencentim.DeliveryError{Code: 403, Message: "发送账号已停用"}, false)
				return
			}
			if user.OwnerID > 0 {
				var owner model.AdminUser
				if err := q.DB.WithContext(ctx).Select("id,status").First(&owner, user.OwnerID).Error; err != nil {
					q.finish(item, "", err, false)
					return
				}
				if owner.Status != "active" {
					q.finish(item, "", &tencentim.DeliveryError{Code: 403, Message: "账号归属已停用"}, false)
					return
				}
			}
			if strings.HasPrefix(item.ConversationID, "GROUP") {
				var err error
				groupName, err = client.CheckForwardGroup(ctx, strings.TrimPrefix(item.ConversationID, "GROUP"), item.Sender, q.waitQuery)
				if err != nil {
					q.finish(item, "", err, false)
					return
				}
			}
			// All permission/content I/O precedes the pacing reservation.
			var err error
			body, _, err = keywordfilter.MaskMessageElementsWithDB(ctx, q.DB, originalBody)
			if err != nil {
				q.finish(item, "", err, false)
				return
			}
			checkedAt = time.Now()
		}
		wait, err := q.slot(ctx, item)
		if err != nil {
			q.finish(item, "", err, false)
			return
		}
		if wait == 0 {
			break
		}
		if !pause(ctx, min(wait, time.Second)) {
			q.finish(item, "", ctx.Err(), false)
			return
		}
	}
	preview := messagePreview(body)
	chatType := 1
	pushSender := item.Sender
	if strings.HasPrefix(item.ConversationID, "GROUP") {
		chatType = 2
		pushSender = strings.TrimPrefix(item.ConversationID, "GROUP")
	}
	ext, _ := json.Marshal(map[string]any{"entity": map[string]any{"sender": pushSender, "nickName": user.Nickname, "chatType": chatType, "version": 1, "action": 1}})
	push := tencentim.NewForwardOfflinePushInfo(user.Nickname, preview, string(ext), user.Nickname)
	if strings.HasPrefix(item.ConversationID, "GROUP") {
		push = tencentim.NewForwardOfflinePushInfo(groupName, preview, string(ext), user.Nickname)
	}
	// Persist the dispatch boundary. Cancellation/reclaimed leases cannot dispatch.
	r := q.DB.WithContext(ctx).Model(&model.ForwardDelivery{}).Where("id = ? AND claim = ? AND status = 'preparing' AND lease_until > ?", item.ID, item.Claim, time.Now().UnixMilli()).Updates(map[string]any{"status": "sending", "attempts": item.Attempts, "lease_until": time.Now().UnixMilli() + leaseMS})
	if r.Error != nil || r.RowsAffected != 1 {
		return
	}
	result, err := client.SendForward(ctx, item.Sender, item.ConversationID, body, item.Random, item.Sequence, push)
	q.finish(item, result, err, true)
}

func messagePreview(body []tencentim.MessageElement) string {
	if len(body) > 0 && body[0].MsgType == "TIMTextElem" {
		var t struct{ Text string }
		if json.Unmarshal(body[0].MsgContent, &t) == nil {
			return short(t.Text, 120)
		}
	}
	if len(body) > 0 {
		switch body[0].MsgType {
		case "TIMImageElem":
			return "[图片]"
		case "TIMVideoFileElem":
			return "[视频]"
		case "TIMFileElem":
			return "[文件]"
		case "TIMSoundElem":
			return "[语音]"
		case "TIMFaceElem":
			return "[表情]"
		case "TIMCustomElem":
			return "[分享消息]"
		}
	}
	return "[消息]"
}

func outcome(err error, sent bool, attempts int) (status string, delay time.Duration, message string) {
	if err == nil {
		return "success", 0, ""
	}
	var e *tencentim.DeliveryError
	explicit := errors.As(err, &e)
	if sent && (!explicit || e.Unknown) {
		return "unknown", 0, "发送结果待确认，请核对目标会话"
	}
	rate := isRateError(err)
	if attempts < 7 && (rate || !sent && (!explicit || e.Unknown)) {
		return "retrying", time.Duration(min(20, 1<<min(attempts, 5))) * time.Second, "等待自动重试"
	}
	if explicit {
		return "failed", 0, e.Message
	}
	return "failed", 0, "发送前检查未通过，请稍后重试"
}

func isRateError(err error) bool {
	var e *tencentim.DeliveryError
	return errors.As(err, &e) && !e.Unknown && (e.Code == 60007 || e.Code == 60011 || e.Code == 60018 || e.Code == 60019 || e.Code == 10023 || e.Code == 429)
}

func (q Queue) finish(item *model.ForwardDelivery, result string, err error, sent bool) {
	status, delay, message := outcome(err, sent, item.Attempts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dbErr := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		from := []string{"preparing"}
		if sent {
			from = []string{"sending", "unknown"}
		}
		updates := map[string]any{"status": status, "attempts": item.Attempts, "next_at": time.Now().Add(delay).UnixMilli(), "error": short(message, 200), "result": short(result, 128), "lease_until": 0}
		if diagnostic, ok := tencentim.ForwardFailureDetails(err); ok {
			updates["last_error_command"] = diagnostic.Command
			updates["last_error_code"] = diagnostic.Code
			updates["last_error_info"] = diagnostic.Info
			updates["last_tencent_request_id"] = diagnostic.RequestID
		}
		r := tx.Model(&model.ForwardDelivery{}).Where("id = ? AND claim = ? AND status IN ?", item.ID, item.Claim, from).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return nil
		}
		if gateID := CurrentSettings().cooldownGate(err, sent, item); status == "retrying" && gateID != "" {
			// Query failures cool down that command; transient local/network
			// preflight errors do not pause every user's sends.
			gate := model.ForwardGate{ID: gateID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&gate).Error; err != nil {
				return err
			}
			return tx.Model(&gate).Update("next_at", gorm.Expr("GREATEST(next_at, CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS SIGNED) + ?)", delay.Milliseconds())).Error
		}
		if status == "failed" || status == "unknown" {
			return tx.Model(&model.ForwardDelivery{}).Where("job_id = ? AND conversation_id = ? AND ordinal > ? AND status IN ?", item.JobID, item.ConversationID, item.Ordinal, []string{"pending", "retrying"}).Updates(map[string]any{"status": "blocked", "error": "前序消息未完成，本会话后续转发等待补发"}).Error
		}
		return nil
	})
	if dbErr != nil {
		logger.Warnf("forward delivery result persistence failed: delivery=%d", item.ID)
	}
}
