package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OwnershipError struct{ Code string }

func (e OwnershipError) Error() string { return e.Code }

var ErrOwnershipProtected = errors.New("归属字段只能通过受控事务更新")

type OwnershipRepository struct{ DB *gorm.DB }

func NewOwnershipRepository() *OwnershipRepository {
	return &OwnershipRepository{DB: datasource.Datasource.Mysql.GetClient()}
}
func OwnershipID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

type OwnerTx struct {
	DB          *gorm.DB
	Users       map[uint]*model.User
	ByIM        map[string]*model.User
	Owners      map[uint]*model.AdminUser
	groupBefore map[uint]map[string]any
	groupAfter  map[uint]map[string]any
}

func sortedOwnerIDs(ids []uint) []uint {
	seen := map[uint]bool{}
	out := []uint{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Every mutable ownership path locks user rows, then owner rows, then invitation
// rows, then group rows, each in stable primary-key order. No remote call here.
func lockOwnerRows(tx *gorm.DB, ids []uint, extraOwners []uint) (*OwnerTx, error) {
	s := &OwnerTx{DB: tx, Users: map[uint]*model.User{}, ByIM: map[string]*model.User{}, Owners: map[uint]*model.AdminUser{}, groupBefore: map[uint]map[string]any{}, groupAfter: map[uint]map[string]any{}}
	ownerIDs := append([]uint{}, extraOwners...)
	for _, id := range sortedOwnerIDs(ids) {
		var u model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, id).Error; err != nil {
			return nil, err
		}
		s.Users[id] = &u
		s.ByIM[u.IMUserID] = &u
		ownerIDs = append(ownerIDs, u.OwnerID)
	}
	for _, id := range sortedOwnerIDs(ownerIDs) {
		var a model.AdminUser
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		s.Owners[id] = &a
	}
	return s, nil
}

func (s *OwnerTx) Scope(u *model.User) model.OwnerScope {
	if u == nil {
		return model.ClassifyOwner(nil, nil)
	}
	return model.ClassifyOwner(u, s.Owners[u.OwnerID])
}
func (s *OwnerTx) Summary(u *model.User) model.OwnerSummary {
	if u == nil {
		return model.SummarizeOwner(nil, nil)
	}
	return model.SummarizeOwner(u, s.Owners[u.OwnerID])
}

func (s *OwnerTx) audit() string {
	rows := []map[string]any{}
	ids := []uint{}
	for id := range s.Users {
		ids = append(ids, id)
	}
	for _, id := range sortedOwnerIDs(ids) {
		u := s.Users[id]
		rows = append(rows, map[string]any{"user_id": u.ID, "biz_id": u.BizID, "im_user_id": u.IMUserID, "owner_id": u.OwnerID, "invite_code": u.InviteCode, "owner_revision": u.OwnerRevision, "status": u.Status, "owner_state": s.Scope(u).State, "owner_role": s.Scope(u).Role})
	}
	return encodeOwnership(map[string]any{"users": rows, "groups": ownerAuditGroups(s.groupAfter)})
}

func ownerAuditGroups(groups map[uint]map[string]any) []map[string]any {
	ids := []uint{}
	for id := range groups {
		ids = append(ids, id)
	}
	rows := []map[string]any{}
	for _, id := range sortedOwnerIDs(ids) {
		rows = append(rows, groups[id])
	}
	return rows
}
func (s *OwnerTx) auditBefore(raw string) string {
	value := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &value)
	value["groups"] = ownerAuditGroups(s.groupBefore)
	return encodeOwnership(value)
}
func ownerGroupAudit(g model.Group) map[string]any {
	return map[string]any{"group_id": g.ID, "im_group_id": g.IMGroupID, "created_by": g.CreatedBy, "owner_id": g.OwnerID, "owner_revision": g.OwnerRevision, "owner_binding": g.OwnerBinding, "mirror_revision": g.MirrorRevision}
}

func ownerInviteCode(tx *gorm.DB, owner uint, fallback string) (string, error) {
	var codes []model.InviteCode
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_id = ?", owner).Order("id").Find(&codes).Error; err != nil {
		return "", err
	}
	now := time.Now()
	for _, code := range codes {
		if code.Status == "active" && (code.ExpiredAt == nil || code.ExpiredAt.After(now)) && (code.MaxUses == 0 || code.UsedCount < code.MaxUses) {
			return strings.TrimSpace(code.Code), nil
		}
	}
	for _, code := range codes {
		if code.Code == strings.TrimSpace(fallback) {
			return code.Code, nil
		}
	}
	return "", nil
}

func (s *OwnerTx) move(u *model.User, owner uint, invite string) ([]uint, error) {
	oldOwner := u.OwnerID
	changed := oldOwner != owner || u.InviteCode != invite
	revision := u.OwnerRevision
	if changed {
		revision++
	}
	var groupIDs []uint
	if err := s.DB.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&model.Group{}).Where("created_by = ? AND status = ?", u.ID, "active").Order("id").Pluck("id", &groupIDs).Error; err != nil {
		return nil, err
	}
	affected := []uint{}
	for _, id := range groupIDs {
		var g model.Group
		if err := s.DB.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, id).Error; err != nil {
			return nil, err
		}
		if g.CreatedBy != u.ID || g.Status != "active" {
			continue
		}
		if g.OwnerBinding == "independent" {
			continue
		}
		if g.OwnerBinding != "follow_user" {
			var count int64
			if err := s.DB.Model(&model.AdminUser{}).Where("id = ?", u.ID).Count(&count).Error; err != nil {
				return nil, err
			}
			if count > 0 {
				return nil, OwnershipError{"group_creator_ambiguous"}
			}
		}
		if g.OwnerBinding != "follow_user" && g.OwnerID != oldOwner {
			return nil, OwnershipError{"group_owner_conflict"}
		}
		if g.OwnerBinding == "follow_user" && g.OwnerRevision > u.OwnerRevision {
			return nil, OwnershipError{"group_revision_conflict"}
		}
		if g.OwnerID != owner || g.OwnerRevision != revision || g.OwnerBinding != "follow_user" {
			s.groupBefore[g.ID] = ownerGroupAudit(g)
			if err := s.DB.Model(&g).Updates(map[string]any{"owner_id": owner, "owner_revision": revision, "owner_binding": "follow_user", "mirror_revision": gorm.Expr("mirror_revision + 1")}).Error; err != nil {
				return nil, err
			}
			g.OwnerID = owner
			g.OwnerRevision = revision
			g.OwnerBinding = "follow_user"
			g.MirrorRevision++
			s.groupAfter[g.ID] = ownerGroupAudit(g)
			affected = append(affected, g.ID)
		}
	}
	if changed {
		result := s.DB.Model(&model.User{}).Where("id = ? AND owner_revision = ?", u.ID, u.OwnerRevision).
			Updates(map[string]any{"owner_id": owner, "invite_code": invite, "owner_revision": revision})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, OwnershipError{"revision_conflict"}
		}
		u.OwnerID = owner
		u.InviteCode = invite
		u.OwnerRevision = revision
	}
	return affected, nil
}

func encodeOwnership(value any) string { b, _ := json.Marshal(value); return string(b) }

const (
	ownerTaskMaxAttempts = 8
	ownerTaskBatchSize   = 32
	ownerTaskLease       = 30 * time.Second
)

func (r *OwnershipRepository) Enqueue(ctx context.Context, source, event, caller, target string) (*model.FriendOwnerOperation, error) {
	if strings.TrimSpace(caller) == "" || len(caller) > 64 || len(target) > 64 || caller == target {
		return nil, OwnershipError{"invalid_pair"}
	}
	if event == "" {
		event = OwnershipID()
	}
	sum := sha256.Sum256([]byte(source + "\x00" + event + "\x00" + caller + "\x00" + target))
	operationID := OwnershipID()
	eventKey := hex.EncodeToString(sum[:])
	var saved model.FriendOwnerOperation
	err := r.taskTransaction(ctx, "enqueue", operationID, false, func(tx *gorm.DB, attempt *ownerTaskAttempt) error {
		row := model.FriendOwnerOperation{OperationID: operationID, EventKey: eventKey, Source: source, Caller: caller, Target: target, State: "pending", AvailableAt: time.Now()}
		inserted := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		attempt.rows = inserted.RowsAffected
		if inserted.Error != nil {
			return inserted.Error
		}
		saved = model.FriendOwnerOperation{}
		if err := tx.Where("event_key = ?", eventKey).First(&saved).Error; err != nil {
			return err
		}
		attempt.operationID = saved.OperationID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// lockOwnerTask locks just the primary-key record. Queue discovery uses
// non-locking reads, so a secondary-index scan never locks a range of tasks.
func lockOwnerTask(tx *gorm.DB, id uint64, skipLocked bool) (*model.FriendOwnerOperation, error) {
	lock := clause.Locking{Strength: "UPDATE"}
	if skipLocked {
		lock.Options = "SKIP LOCKED"
	}
	var row model.FriendOwnerOperation
	if err := tx.Clauses(lock).Take(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func ownerTaskLeaseExpired(row *model.FriendOwnerOperation, now time.Time) bool {
	return row.LeaseUntil == nil || row.LeaseUntil.Before(now)
}

func (r *OwnershipRepository) Claim(ctx context.Context, id uint64) (*model.FriendOwnerOperation, error) {
	var claimed *model.FriendOwnerOperation
	err := r.taskTransaction(ctx, "claim", "", false, func(tx *gorm.DB, attempt *ownerTaskAttempt) error {
		now := time.Now()
		ids := []uint64{id}
		if id == 0 {
			ids = nil
			if err := tx.Model(&model.FriendOwnerOperation{}).
				Where("available_at <= ? AND (lease_until IS NULL OR lease_until < ?)", now, now).
				Where("(state = ? OR effects_pending = ?) AND attempts < ?", "pending", true, ownerTaskMaxAttempts).
				Order("id").Limit(ownerTaskBatchSize).Pluck("id", &ids).Error; err != nil {
				return err
			}
		}
		for _, candidate := range ids {
			row, err := lockOwnerTask(tx, candidate, true)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if row.AvailableAt.After(now) || !ownerTaskLeaseExpired(row, now) ||
				row.Attempts >= ownerTaskMaxAttempts || (row.State != "pending" && !row.EffectsPending) {
				continue
			}
			attempt.operationID = row.OperationID
			until := time.Now().Add(ownerTaskLease)
			row.LeaseUntil = &until
			row.LeaseToken = OwnershipID()
			row.Attempts++
			update := tx.Model(row).Updates(map[string]any{"lease_token": row.LeaseToken, "lease_until": until, "attempts": row.Attempts})
			attempt.rows = update.RowsAffected
			claimed = row
			return update.Error
		}
		return gorm.ErrRecordNotFound
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// MarkExpiredForReview recovers workers that crashed on their last attempt.
// Only the maintenance loop calls it. The scan is unlocked, and each candidate
// is rechecked under a primary-key lock before a bounded, ordered update.
func (r *OwnershipRepository) MarkExpiredForReview(ctx context.Context) error {
	return r.taskTransaction(ctx, "review_expired", "", false, func(tx *gorm.DB, attempt *ownerTaskAttempt) error {
		now := time.Now()
		var ids []uint64
		if err := tx.Model(&model.FriendOwnerOperation{}).
			Where("attempts >= ? AND (lease_until IS NULL OR lease_until < ?)", ownerTaskMaxAttempts, now).
			Where("(state = ? OR (effects_pending = ? AND needs_review = ?))", "pending", true, false).
			Order("id").Limit(ownerTaskBatchSize).Pluck("id", &ids).Error; err != nil {
			return err
		}
		for _, id := range ids {
			row, err := lockOwnerTask(tx, id, true)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if row.Attempts < ownerTaskMaxAttempts || !ownerTaskLeaseExpired(row, now) ||
				(row.State != "pending" && (!row.EffectsPending || row.NeedsReview)) {
				continue
			}
			attempt.operationID = row.OperationID
			state := row.State
			if state == "pending" {
				state = "manual"
			}
			update := tx.Model(row).Updates(map[string]any{"needs_review": true, "state": state})
			attempt.rows += update.RowsAffected
			if update.Error != nil {
				return update.Error
			}
		}
		return nil
	})
}

func (r *OwnershipRepository) updateLeasedTask(ctx context.Context, row *model.FriendOwnerOperation, stage string, values func(*model.FriendOwnerOperation, time.Time) map[string]any) error {
	if row.LeaseToken == "" {
		return OwnershipError{"lease_lost"}
	}
	return r.taskTransaction(ctx, stage, row.OperationID, false, func(tx *gorm.DB, attempt *ownerTaskAttempt) error {
		current, err := lockOwnerTask(tx.Select("id", "lease_token", "lease_until", "state", "attempts", "effects_pending"), row.ID, false)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return OwnershipError{"lease_lost"}
		}
		if err != nil {
			return err
		}
		now := time.Now()
		if current.LeaseToken != row.LeaseToken || ownerTaskLeaseExpired(current, now) {
			return OwnershipError{"lease_lost"}
		}
		result := tx.Model(&model.FriendOwnerOperation{}).
			Where("id = ? AND lease_token = ? AND lease_until >= ?", row.ID, row.LeaseToken, now).
			Updates(values(current, now))
		attempt.rows = result.RowsAffected
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return OwnershipError{"lease_lost"}
		}
		return nil
	})
}

// RetryAsAdmin rechecks permission and appends the retry audit in the same
// transaction that resets the job. A rolled-back retry never increments it.
func (r *OwnershipRepository) RetryAsAdmin(ctx context.Context, id uint64, adminID uint) error {
	return r.taskTransaction(ctx, "admin_retry", "", false, func(tx *gorm.DB, attempt *ownerTaskAttempt) error {
		var actor model.AdminUser
		if err := tx.First(&actor, adminID).Error; err != nil {
			return err
		}
		if actor.Role != "super_admin" || actor.Status != "active" {
			return gorm.ErrRecordNotFound
		}
		row, err := lockOwnerTask(tx, id, false)
		if err != nil {
			return err
		}
		attempt.operationID = row.OperationID
		now := time.Now()
		if row.LeaseUntil != nil && row.LeaseUntil.After(now) {
			return gorm.ErrInvalidTransaction
		}
		if (row.State == "migrated" || row.State == "unchanged") && !row.EffectsPending {
			return nil
		}
		state := row.State
		if state == "blocked" || state == "conflict" || state == "manual" {
			state = "pending"
		}
		history := []map[string]any{}
		if row.RetryHistory != "" && json.Unmarshal([]byte(row.RetryHistory), &history) != nil {
			return gorm.ErrInvalidData
		}
		history = append(history, map[string]any{"actor_id": actor.ID, "time": now, "state": row.State, "reason": row.ReasonCode, "attempts": row.Attempts, "before": row.BeforeJSON, "after": row.AfterJSON, "result": row.ResultJSON})
		reason := row.ReasonCode
		if state == "pending" {
			reason = "manual_retry_queued"
		}
		update := tx.Model(row).Updates(map[string]any{
			"state": state, "reason_code": reason, "attempts": 0, "manual_retries": gorm.Expr("manual_retries + 1"),
			"last_retried_by": actor.ID, "last_retried_at": now, "retry_history": encodeOwnership(history),
			"needs_review": false, "available_at": now, "lease_until": nil, "lease_token": "",
		})
		attempt.rows = update.RowsAffected
		return update.Error
	})
}

func (r *OwnershipRepository) Retry(ctx context.Context, row *model.FriendOwnerOperation, reason string) error {
	return r.updateLeasedTask(ctx, row, "retry_task", func(current *model.FriendOwnerOperation, now time.Time) map[string]any {
		state := current.State
		if state == "processing" {
			state = "pending"
		}
		// A lost commit response may leave the database ahead of the caller's
		// snapshot. Keep the recorded outcome and resume only pending effects.
		if state != "pending" && !current.EffectsPending {
			return map[string]any{"lease_until": nil, "lease_token": ""}
		}
		exhausted := current.Attempts >= ownerTaskMaxAttempts
		if exhausted && state == "pending" {
			state = "manual"
		}
		delay := time.Second * time.Duration(1<<min(current.Attempts, ownerTaskMaxAttempts))
		return map[string]any{"state": state, "reason_code": reason, "last_error": reason, "needs_review": exhausted, "available_at": now.Add(delay), "lease_until": nil, "lease_token": ""}
	})
}

func (r *OwnershipRepository) FinishEffects(ctx context.Context, row *model.FriendOwnerOperation) error {
	return r.updateLeasedTask(ctx, row, "finish_effects", func(*model.FriendOwnerOperation, time.Time) map[string]any {
		return map[string]any{"effects_pending": false, "last_error": "", "lease_until": nil, "lease_token": ""}
	})
}

// ApplyConfirmed receives ONLY externally verified bilateral relationships.
// Reconciliation checks every verified peer together, so list ordering never
// chooses among historical operating spaces.
func (r *OwnershipRepository) ApplyConfirmed(ctx context.Context, job *model.FriendOwnerOperation, peers []string) (*model.OwnerAssignment, error) {
	var answer model.OwnerAssignment
	err := r.taskTransaction(ctx, "apply_confirmed", job.OperationID, true, func(tx *gorm.DB, attempt *ownerTaskAttempt) error {
		answer = model.OwnerAssignment{}
		current, err := lockOwnerTask(tx, job.ID, false)
		if err != nil {
			return err
		}
		if job.LeaseToken == "" || current.LeaseToken != job.LeaseToken || ownerTaskLeaseExpired(current, time.Now()) {
			return OwnershipError{"lease_lost"}
		}
		if current.State != "pending" {
			if current.ResultJSON != "" {
				return json.Unmarshal([]byte(current.ResultJSON), &answer)
			}
			return nil
		}
		accounts := append([]string{job.Caller}, peers...)
		var ids []uint
		if err := tx.Model(&model.User{}).Where("im_user_id IN ?", accounts).Pluck("id", &ids).Error; err != nil {
			return err
		}
		s, err := lockOwnerRows(tx, ids, nil)
		if err != nil {
			return err
		}
		caller := s.ByIM[job.Caller]
		target := s.ByIM[job.Target]
		answer = model.OwnerAssignment{State: "unchanged", RequestID: job.OperationID, Caller: s.Summary(caller), Target: s.Summary(target), AffectedGroups: []uint{}}
		before := s.audit()
		reason := ""
		selected := target
		if caller == nil {
			reason = "user_unavailable"
		} else if s.Scope(caller).State == "unavailable" {
			reason = s.Scope(caller).Reason
		}
		if job.Source == "reconcile" && reason == "" {
			spaces := map[uint]*model.User{}
			for _, peer := range peers {
				u := s.ByIM[peer]
				if u == nil {
					continue
				}
				scope := s.Scope(u)
				if scope.State == "unavailable" {
					reason = scope.Reason
					break
				}
				if scope.State == "assigned" {
					spaces[u.OwnerID] = u
				}
			}
			if len(spaces) > 1 {
				reason = "historical_owner_conflict"
			}
			if len(spaces) == 1 {
				for _, u := range spaces {
					selected = u
				}
			}
			if len(spaces) == 0 {
				selected = caller
			}
		}
		if reason == "" && selected == nil {
			reason = "target_unavailable"
		}
		if reason == "" {
			move, owner, blocked := model.SelectOwnerMove(caller, selected, s.Scope(caller), s.Scope(selected))
			reason = blocked
			if reason == "" {
				if move != nil {
					stay := selected
					if move == selected {
						stay = caller
					}
					invite, err := ownerInviteCode(tx, owner, stay.InviteCode)
					if err != nil {
						return err
					}
					groups, err := s.move(move, owner, invite)
					if err != nil {
						return err
					}
					answer.AffectedGroups = groups
					answer.State = "migrated"
				} else {
					for _, u := range []*model.User{caller, selected} {
						if u == nil {
							continue
						}
						groups, err := s.move(u, u.OwnerID, u.InviteCode)
						if err != nil {
							return err
						}
						answer.AffectedGroups = append(answer.AffectedGroups, groups...)
					}
				}
			}
		}
		if reason != "" {
			answer.State = "blocked"
			if strings.Contains(reason, "conflict") || reason == "cross_owner" || reason == "group_creator_ambiguous" {
				answer.State = "conflict"
			}
			answer.ReasonCode = reason
		}
		answer.Caller = s.Summary(caller)
		answer.Target = s.Summary(selected)
		actorID := uint(0)
		if caller != nil {
			actorID = caller.ID
		}
		update := tx.Model(current).Updates(map[string]any{"state": answer.State, "reason_code": reason, "before_json": s.auditBefore(before), "after_json": s.audit(), "result_json": encodeOwnership(answer), "effects_pending": reason == "", "needs_review": answer.State == "conflict", "actor_id": actorID})
		attempt.rows = update.RowsAffected
		return update.Error
	})
	return &answer, err
}

func (r *OwnershipRepository) Block(ctx context.Context, row *model.FriendOwnerOperation, reason string) error {
	state := "blocked"
	if strings.Contains(reason, "conflict") || reason == "cross_owner" || reason == "group_creator_ambiguous" {
		state = "conflict"
	}
	return r.updateLeasedTask(ctx, row, "block_task", func(*model.FriendOwnerOperation, time.Time) map[string]any {
		return map[string]any{"state": state, "reason_code": reason, "last_error": reason, "needs_review": state == "conflict", "effects_pending": false, "lease_until": nil, "lease_token": ""}
	})
}

func (r *OwnershipRepository) RepairTestOwner(ctx context.Context, userID, ownerID uint) error {
	return ownershipTransaction(ctx, r.DB, func(tx *gorm.DB) error {
		s, err := lockOwnerRows(tx, []uint{userID}, []uint{ownerID})
		if err != nil {
			return err
		}
		u := s.Users[userID]
		if u.OwnerID != 0 {
			return nil
		}
		a := s.Owners[ownerID]
		if a == nil || a.Role != "super_admin" || a.Status != "active" {
			return OwnershipError{"owner_inactive"}
		}
		_, err = s.move(u, ownerID, u.InviteCode)
		return err
	})
}

func (r *OwnershipRepository) Assignment(ctx context.Context, id uint64) (*model.OwnerAssignment, error) {
	var row model.FriendOwnerOperation
	if err := r.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	result := &model.OwnerAssignment{State: row.State, ReasonCode: row.ReasonCode, RequestID: row.OperationID, AffectedGroups: []uint{}}
	if row.ResultJSON != "" && row.State != "pending" && row.State != "manual" {
		if err := json.Unmarshal([]byte(row.ResultJSON), result); err != nil {
			return nil, err
		}
	}
	if result.State == "manual" {
		result.State = "pending"
		result.ReasonCode = "manual_review_required"
	}
	return result, nil
}

func (r *OwnershipRepository) Transfer(ctx context.Context, actorID, userID, ownerID uint) (*model.OwnerAssignment, error) {
	var answer model.OwnerAssignment
	err := ownershipTransaction(ctx, r.DB, func(tx *gorm.DB) error {
		s, err := lockOwnerRows(tx, []uint{userID}, []uint{actorID, ownerID})
		if err != nil {
			return err
		}
		actor := s.Owners[actorID]
		targetOwner := s.Owners[ownerID]
		u := s.Users[userID]
		if actor == nil || actor.Status != "active" || (actor.Role != "super_admin" && actor.Role != "admin") {
			return OwnershipError{"actor_forbidden"}
		}
		if targetOwner == nil || targetOwner.Status != "active" {
			return OwnershipError{"owner_inactive"}
		}
		if targetOwner.Role != "super_admin" && targetOwner.Role != "admin" && targetOwner.Role != "assistant" {
			return OwnershipError{"owner_role_unknown"}
		}
		if u.Status == "deleted" {
			return OwnershipError{"user_unavailable"}
		}
		if actor.Role != "super_admin" {
			managed := func(a *model.AdminUser) bool {
				return a != nil && (a.ID == actor.ID || a.Role == "assistant" && a.ParentAdminID != nil && *a.ParentAdminID == actor.ID)
			}
			if !managed(s.Owners[u.OwnerID]) || !managed(targetOwner) {
				return OwnershipError{"actor_forbidden"}
			}
		}
		rawBefore := s.audit()
		before := s.Summary(u)
		invite, err := ownerInviteCode(tx, ownerID, "")
		if err != nil {
			return err
		}
		groups, err := s.move(u, ownerID, invite)
		if err != nil {
			return err
		}
		answer = model.OwnerAssignment{State: "unchanged", RequestID: OwnershipID(), Caller: before, Target: s.Summary(u), AffectedGroups: groups}
		if before.OwnerRevision != u.OwnerRevision {
			answer.State = "migrated"
		}
		audit := model.FriendOwnerOperation{OperationID: answer.RequestID, EventKey: OwnershipID(), Source: "admin_transfer", Caller: u.IMUserID, Target: "", State: answer.State, AvailableAt: time.Now(), BeforeJSON: s.auditBefore(rawBefore), AfterJSON: s.audit(), ActorID: actorID, ResultJSON: encodeOwnership(answer), EffectsPending: true}
		return tx.Create(&audit).Error
	})
	return &answer, err
}

func (r *OwnershipRepository) BindInvite(ctx context.Context, imID, codeText string) (*model.User, error) {
	var answer *model.User
	var initial model.InviteCode
	if err := r.DB.WithContext(ctx).Where("code = ?", codeText).First(&initial).Error; err != nil {
		return nil, OwnershipError{"invite_invalid"}
	}
	var initialUser model.User
	if err := r.DB.WithContext(ctx).Where("im_user_id = ?", imID).First(&initialUser).Error; err != nil {
		return nil, err
	}
	err := ownershipTransaction(ctx, r.DB, func(tx *gorm.DB) error {
		s, err := lockOwnerRows(tx, []uint{initialUser.ID}, []uint{initial.OwnerID})
		if err != nil {
			return err
		}
		u := s.Users[initialUser.ID]
		if u.Status != "active" {
			return OwnershipError{"user_unavailable"}
		}
		if scope := s.Scope(u); scope.State == "unavailable" {
			return OwnershipError{scope.Reason}
		}
		if strings.TrimSpace(u.InviteCode) != "" {
			return OwnershipError{"invite_already_bound"}
		}
		var code model.InviteCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&code, initial.ID).Error; err != nil {
			return err
		}
		if code.OwnerID != initial.OwnerID || code.Code != codeText || code.Status != "active" || code.ExpiredAt != nil && !code.ExpiredAt.After(time.Now()) || code.MaxUses > 0 && code.UsedCount >= code.MaxUses {
			return OwnershipError{"invite_invalid"}
		}
		scope := model.ClassifyOwner(&model.User{OwnerID: code.OwnerID, Status: "active"}, s.Owners[code.OwnerID])
		if scope.State == "unavailable" {
			return OwnershipError{scope.Reason}
		}
		rawBefore := s.audit()
		before := s.Summary(u)
		groups, err := s.move(u, code.OwnerID, codeText)
		if err != nil {
			return err
		}
		values := map[string]any{"used_count": code.UsedCount + 1}
		if code.MaxUses > 0 && code.UsedCount+1 >= code.MaxUses {
			values["status"] = "used"
		}
		if err := tx.Model(&code).Updates(values).Error; err != nil {
			return err
		}
		result := model.OwnerAssignment{State: "migrated", RequestID: OwnershipID(), Caller: before, Target: s.Summary(u), AffectedGroups: groups}
		audit := model.FriendOwnerOperation{OperationID: result.RequestID, EventKey: OwnershipID(), Source: "invite_bind", Caller: imID, State: "migrated", AvailableAt: time.Now(), BeforeJSON: s.auditBefore(rawBefore), AfterJSON: s.audit(), ActorID: u.ID, ResultJSON: encodeOwnership(result), EffectsPending: true}
		answer = u
		return tx.Create(&audit).Error
	})
	return answer, err
}
