package controller_chat

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/messagelimit"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/app/server/usercache"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

type FriendshipProof interface {
	CheckBilateral(context.Context, string, []string) (map[string]bool, error)
	ListFriendIDsContext(context.Context, string) ([]string, error)
}

type OwnerService struct {
	Repo    *repository.OwnershipRepository
	Proof   FriendshipProof
	Effects func(context.Context, *model.OwnerAssignment) error
}

func currentOwnerService() OwnerService {
	return OwnerService{Repo: repository.NewOwnershipRepository(), Proof: tencentim.NewClient(), Effects: applyOwnerEffects}
}

func ownerForPolicy(u *model.User) *model.AdminUser {
	if u == nil || u.OwnerID == 0 {
		return nil
	}
	a, err := repository.NewAdminUserRepository().FindByID(u.OwnerID)
	if err != nil {
		return nil
	}
	return a
}
func CurrentOwnerSummary(u *model.User) model.OwnerSummary {
	return model.SummarizeOwner(u, ownerForPolicy(u))
}

func OwnerReasonMessage(code string) string {
	switch code {
	case "owner_missing":
		return "归属账号不存在，暂时无法操作，请联系管理员"
	case "owner_inactive":
		return "归属账号已停用，暂时无法操作"
	case "owner_role_unknown":
		return "归属角色不可确认，请联系管理员"
	case "user_unavailable", "target_unavailable":
		return "账号当前不可用"
	case "not_globally_searchable":
		return "对方未开放全局搜索，无法从默认池发起申请"
	case "historical_owner_conflict":
		return "历史好友涉及多个服务空间，归属待管理员确认"
	case "group_owner_conflict", "group_revision_conflict", "group_creator_ambiguous":
		return "群归属存在独立转移或历史冲突，请联系管理员核对"
	case "invite_already_bound":
		return "已绑定邀请码，无法再次修改"
	case "invite_invalid":
		return "邀请码无效或已失效"
	case "actor_forbidden":
		return "当前账号无权转移此归属"
	case "friendship_not_confirmed":
		return "好友关系尚未确认，归属待同步"
	case "cross_owner":
		return ErrInfoCrossOwnerFriendAdd
	default:
		return "归属暂未同步，请稍后重试"
	}
}

func (s OwnerService) Process(ctx context.Context, op *model.FriendOwnerOperation) error {
	if op.LeaseToken == "" || op.LeaseUntil == nil || !op.LeaseUntil.After(time.Now()) {
		return repository.OwnershipError{Code: "lease_lost"}
	}
	ctx, release := context.WithDeadline(ctx, *op.LeaseUntil)
	defer release()

	if op.State != "pending" && op.EffectsPending {
		assignment, err := s.Repo.Assignment(ctx, op.ID)
		if err != nil {
			return err
		}
		if s.Effects != nil {
			if err := s.Effects(ctx, assignment); err != nil {
				return s.Repo.Retry(ctx, op, "post_commit_effect_failed")
			}
		}
		return s.Repo.FinishEffects(ctx, op)
	}
	proofCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	peers := []string{op.Target}
	if op.Source == "reconcile" {
		ids, err := s.Proof.ListFriendIDsContext(proofCtx, op.Caller)
		if err != nil {
			return s.Repo.Retry(ctx, op, "friendship_lookup_unavailable")
		}
		if len(ids) > 1000 {
			return s.Repo.Block(ctx, op, "historical_owner_conflict")
		}
		peers = ids
		sort.Strings(peers)
	}
	confirmations, err := s.Proof.CheckBilateral(proofCtx, op.Caller, peers)
	if err != nil {
		return s.Repo.Retry(ctx, op, "friendship_lookup_unavailable")
	}
	verified := []string{}
	for _, peer := range peers {
		if confirmations[peer] {
			verified = append(verified, peer)
		}
	}
	if op.Source != "reconcile" && len(verified) != 1 {
		return s.Repo.Retry(ctx, op, "friendship_not_confirmed")
	}
	result, err := s.Repo.ApplyConfirmed(ctx, op, verified)
	if err != nil {
		var policy repository.OwnershipError
		if errors.As(err, &policy) {
			if repository.IsOwnershipLeaseLost(err) {
				return err
			}
			return s.Repo.Block(ctx, op, policy.Code)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		rescheduleErr := s.Repo.Retry(ctx, op, "assignment_database_unavailable")
		if repository.IsOwnershipDeadlock(err) {
			// The task can still be recovered later. The request that exhausted
			// its database retry budget must report the original failure.
			return err
		}
		return rescheduleErr
	}
	if result.State == "migrated" || result.State == "unchanged" {
		// The assignment transaction is already committed. Failure here leaves an
		// outbox task, rather than turning an established friendship into a failure.
		if s.Effects != nil {
			if err := s.Effects(ctx, result); err != nil {
				return s.Repo.Retry(ctx, op, "post_commit_effect_failed")
			}
		}
	}
	return s.Repo.FinishEffects(ctx, op)
}

func applyOwnerEffects(ctx context.Context, a *model.OwnerAssignment) error {
	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)
	repo := repository.NewUserRepositoryWithDB(db)
	realname := repository.NewUserRealNameRepositoryWithDB(db)
	for _, summary := range []model.OwnerSummary{a.Caller, a.Target} {
		if summary.IMUserID == "" {
			continue
		}
		usercache.InvalidateByIMUserID(ctx, summary.IMUserID)
		user, err := repo.FindByIMUserID(summary.IMUserID)
		if err != nil {
			return err
		}
		if CurrentOwnerSummary(user).IsAssigned {
			if err := realname.UpsertAssignedVerified(user.ID); err != nil {
				return err
			}
			messagelimit.InvalidateRealNameVerified(ctx, user.ID)
		}
	}
	keywordfilter.InvalidateFilterSenders()
	return nil
}

func QueueFriendOwnership(ctx context.Context, source, event, caller, target string) (*model.FriendOwnerOperation, error) {
	return repository.NewOwnershipRepository().Enqueue(ctx, source, event, strings.TrimSpace(caller), strings.TrimSpace(target))
}

func SyncFriendOwnership(ctx context.Context, caller, target, request string) (*model.OwnerAssignment, error) {
	s := currentOwnerService()
	op, err := s.Repo.Enqueue(ctx, "client", request, caller, target)
	if err != nil {
		return nil, err
	}
	claimed, err := s.Repo.Claim(ctx, op.ID)
	if err == nil {
		if err := s.Process(ctx, claimed); err != nil && !repository.IsOwnershipLeaseLost(err) {
			return nil, err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	result, err := s.Repo.Assignment(ctx, op.ID)
	if err != nil {
		return nil, err
	}
	users, err := repository.NewUserRepository().FindByIMUserIDs([]string{caller, target})
	if err != nil {
		return nil, err
	}
	for i := range users {
		u := &users[i]
		if u.IMUserID == caller {
			result.Caller = CurrentOwnerSummary(u)
		}
		if u.IMUserID == target {
			result.Target = CurrentOwnerSummary(u)
		}
	}
	return result, nil
}

func StartFriendOwnerWorker(ctx context.Context, db *gorm.DB) {
	repo := &repository.OwnershipRepository{DB: db}
	go maintainFriendOwnerTasks(ctx, repo)
	for i := 0; i < 2; i++ {
		go func() {
			s := currentOwnerService()
			s.Repo = repo
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					taskCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
					op, err := s.Repo.Claim(taskCtx, 0)
					if err == nil {
						if err := s.Process(taskCtx, op); err != nil && !repository.IsOwnershipLeaseLost(err) {
							logger.Warnf("friend ownership durable worker failed operation=%s", op.OperationID)
						}
					}
					cancel()
				}
			}
		}()
	}
}

func maintainFriendOwnerTasks(ctx context.Context, repo *repository.OwnershipRepository) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			taskCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			// The repository logs failed attempts. The next tick resumes the
			// bounded sweep after an error or a busy row.
			_ = repo.MarkExpiredForReview(taskCtx)
			cancel()
		}
	}
}

// Exported for operational diagnostics without exposing the datasource client.
func OwnerDatabase() *gorm.DB { return datasource.Datasource.Mysql.GetClient() }
