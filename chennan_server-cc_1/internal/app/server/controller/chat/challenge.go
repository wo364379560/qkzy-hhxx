package controller_chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/quietlog"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

// CreateChallengeReq 表示创建挑战请求。
type CreateChallengeReq struct {
	Name            string
	Description     string
	LinkedHabitID   uint
	DurationDays    int
	MemberIMUserIDs []string
}

// ChallengeCard 表示打卡首页中的挑战卡片。
type ChallengeCard struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	GroupID     string `json:"group_id"`

	RecordID          uint   `json:"record_id"`
	RecordTitle       string `json:"record_title"`
	RecordDescription string `json:"record_description"`
	DiscussionGroupID string `json:"discussion_group_id"`
}

// CreateChallengeResult 表示创建挑战结果。
type CreateChallengeResult struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	GroupID     string `json:"group_id"`

	RecordID          uint   `json:"record_id"`
	RecordTitle       string `json:"record_title"`
	RecordDescription string `json:"record_description"`
	DiscussionGroupID string `json:"discussion_group_id"`
}

// ChallengeMemberResult 表示挑战详情中的成员。
type ChallengeMemberResult struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// ChallengeMemberProgressResult 表示挑战详情中的成员进度板条目。
type ChallengeMemberProgressResult struct {
	UserID                  string `json:"user_id"`
	Nickname                string `json:"nickname"`
	Role                    string `json:"role"`
	TodayStatus             string `json:"today_status"`
	CurrentStreak           int    `json:"current_streak"`
	ChallengeCompletionRate int    `json:"challenge_completion_rate"`
	LastCheckinAt           string `json:"last_checkin_at,omitempty"`
	CompletedDays           int    `json:"completed_days"`
	TargetDays              int    `json:"target_days"`
}

// ChallengeDetailResult 表示挑战详情。
type ChallengeDetailResult struct {
	ID               uint                            `json:"id"`
	Title            string                          `json:"title"`
	Description      string                          `json:"description"`
	GroupID          string                          `json:"group_id"`
	LinkedHabitID    uint                            `json:"linked_habit_id"`
	LinkedHabitName  string                          `json:"linked_habit_name"`
	StartDate        string                          `json:"start_date"`
	EndDate          string                          `json:"end_date"`
	MemberCount      int                             `json:"member_count"`
	CompletedToday   int                             `json:"completed_today"`
	DurationDays     int                             `json:"duration_days"`
	ElapsedDays      int                             `json:"elapsed_days"`
	CompletionRate   int                             `json:"completion_rate"`
	Members          []ChallengeMemberResult         `json:"members"`
	MemberProgresses []ChallengeMemberProgressResult `json:"member_progresses"`

	RecordID             uint                            `json:"record_id"`
	RecordTitle          string                          `json:"record_title"`
	RecordDescription    string                          `json:"record_description"`
	DiscussionGroupID    string                          `json:"discussion_group_id"`
	LinkedLedgerID       uint                            `json:"linked_ledger_id"`
	LinkedLedgerName     string                          `json:"linked_ledger_name"`
	ParticipantCount     int                             `json:"participant_count"`
	RecordCompletionRate int                             `json:"record_completion_rate"`
	Participants         []ChallengeMemberResult         `json:"participants"`
	ParticipantProgress  []ChallengeMemberProgressResult `json:"participant_progress"`
}

// CreateChallenge 创建监督挑战并自动建群。
func CreateChallenge(ctx context.Context, imUserID string, req CreateChallengeReq) (*CreateChallengeResult, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		return nil, errors.New("挑战名称不能为空")
	}
	if req.DurationDays <= 0 {
		req.DurationDays = 7
	}

	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	if !caller.CanCreateGroup {
		return nil, AuthErr("当前账号未开启建群权限，请联系管理员")
	}

	if err := ensureCreateGroupAllowedNow(ctx, caller.OwnerID, time.Now()); err != nil {
		if authErr, ok := err.(AuthErr); ok {
			_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
				Action:        "create_group",
				Source:        "challenge_create",
				ActorIMUserID: caller.IMUserID,
				ActorPhone:    caller.Phone,
				Reason:        authErr.Error(),
			})
		}
		return nil, err
	}

	habitRepo := repository.NewHabitRepository()
	habit, err := habitRepo.FindHabitByIDAndUser(req.LinkedHabitID, caller.ID)
	if err != nil {
		return nil, errors.New("习惯不存在")
	}

	memberIDs := []uint{caller.ID}
	seenUserID := map[uint]bool{caller.ID: true}
	for _, memberIMUserID := range req.MemberIMUserIDs {
		memberIMUserID = strings.TrimSpace(memberIMUserID)
		if memberIMUserID == "" || memberIMUserID == caller.IMUserID {
			continue
		}
		member, err := userRepo.FindByIMUserID(memberIMUserID)
		if err != nil {
			return nil, errors.New("参与成员不存在")
		}
		if member.OwnerID != caller.OwnerID {
			return nil, errors.New("参与成员必须和你属于同一归属范围")
		}
		if seenUserID[member.ID] {
			continue
		}
		seenUserID[member.ID] = true
		memberIDs = append(memberIDs, member.ID)
	}

	groupName := fmt.Sprintf("%s（打卡小组）", req.Name)
	groupResult, err := controller_admin.CreateGroup(controller_admin.CreateGroupReq{
		Name:          groupName,
		GroupType:     "Work",
		OwnerID:       caller.OwnerID,
		CreatedBy:     caller.ID,
		MemberIDs:     memberIDs,
		OwnerMemberID: caller.ID,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	startDate := normalizeDay(now)
	endDate := startDate.AddDate(0, 0, req.DurationDays-1)

	challenge := &model.Challenge{
		OwnerID:           caller.OwnerID,
		CreatorUserID:     caller.ID,
		Name:              req.Name,
		Description:       req.Description,
		LinkedUserHabitID: habit.ID,
		StartDate:         startDate,
		EndDate:           endDate,
		Status:            "active",
		IMGroupID:         groupResult.IMGroupID,
	}

	challengeRepo := repository.NewChallengeRepository()
	if err := challengeRepo.Create(challenge); err != nil {
		return nil, err
	}

	members := make([]model.ChallengeMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		role := "member"
		if memberID == caller.ID {
			role = "owner"
		}
		members = append(members, model.ChallengeMember{
			ChallengeID: challenge.ID,
			UserID:      memberID,
			Role:        role,
			JoinStatus:  "active",
			JoinedAt:    now,
		})
	}
	if err := challengeRepo.CreateMembers(members); err != nil {
		return nil, err
	}

	description := buildChallengeDescription(challenge.StartDate, challenge.EndDate, int64(len(memberIDs)))
	return &CreateChallengeResult{
		ID:                challenge.ID,
		Title:             challenge.Name,
		Description:       description,
		GroupID:           challenge.IMGroupID,
		RecordID:          challenge.ID,
		RecordTitle:       challenge.Name,
		RecordDescription: description,
		DiscussionGroupID: challenge.IMGroupID,
	}, nil
}

func listActiveChallengesForDashboard(userID uint, day time.Time) ([]ChallengeCard, error) {
	challengeRepo := repository.NewChallengeRepository()
	rows, err := challengeRepo.ListActiveByUser(userID, day)
	if err != nil {
		return nil, err
	}
	cards := make([]ChallengeCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, ChallengeCard{
			ID:                row.ChallengeID,
			Title:             row.Name,
			Description:       buildChallengeDescription(row.StartDate, row.EndDate, row.MemberCount),
			GroupID:           row.IMGroupID,
			RecordID:          row.ChallengeID,
			RecordTitle:       row.Name,
			RecordDescription: buildChallengeDescription(row.StartDate, row.EndDate, row.MemberCount),
			DiscussionGroupID: row.IMGroupID,
		})
	}
	return cards, nil
}

func buildChallengeDescription(startDate, endDate time.Time, memberCount int64) string {
	return fmt.Sprintf(
		"%s - %s · %d 人参与",
		startDate.Format("01/02"),
		endDate.Format("01/02"),
		memberCount,
	)
}

// GetChallengeDetail 获取挑战详情。
func GetChallengeDetail(_ context.Context, imUserID string, challengeID uint) (*ChallengeDetailResult, error) {
	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	challengeRepo := repository.NewChallengeRepository()
	challenge, err := challengeRepo.FindByID(challengeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("挑战不存在")
		}
		return nil, err
	}
	if challenge.OwnerID != caller.OwnerID {
		return nil, errors.New("无权查看该挑战")
	}

	members, err := challengeRepo.ListMembersByChallenge(challenge.ID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(members))
	seenUserID := make(map[uint]bool, len(members))
	for _, member := range members {
		if !seenUserID[member.UserID] {
			seenUserID[member.UserID] = true
			userIDs = append(userIDs, member.UserID)
		}
	}

	userNameMap := make(map[uint]*model.User, len(userIDs))
	for _, userID := range userIDs {
		u, err := userRepo.FindByID(userID)
		if err == nil {
			userNameMap[userID] = u
		}
	}

	habitRepo := repository.NewHabitRepository()
	habit, err := habitRepo.FindHabitByIDAndUser(challenge.LinkedUserHabitID, challenge.CreatorUserID)
	if err != nil {
		return nil, errors.New("关联账本不存在")
	}

	memberResults := make([]ChallengeMemberResult, 0, len(members))
	callerRole := ""
	callerIsMember := false
	for _, member := range members {
		u := userNameMap[member.UserID]
		if u == nil {
			continue
		}
		if member.UserID == caller.ID {
			callerRole = member.Role
			callerIsMember = true
		}
		memberResults = append(memberResults, ChallengeMemberResult{
			UserID:   u.IMUserID,
			Nickname: u.Nickname,
			Role:     member.Role,
		})
	}

	durationDays := int(challenge.EndDate.Sub(challenge.StartDate).Hours()/24) + 1
	if durationDays < 1 {
		durationDays = 1
	}
	dayStart := normalizeDay(time.Now())
	elapsedDays := int(dayStart.Sub(challenge.StartDate).Hours()/24) + 1
	if elapsedDays < 0 {
		elapsedDays = 0
	}
	if elapsedDays > durationDays {
		elapsedDays = durationDays
	}

	myProgresses := []ChallengeMemberProgressResult{}
	if callerIsMember {
		allCompletedCheckins, err := habitRepo.ListCompletedCheckinsByUsersUpToDate([]uint{caller.ID}, dayStart)
		if err != nil {
			return nil, err
		}

		currentStreak, _ := calculateStreaks(allCompletedCheckins, dayStart)
		todayStatus := "pending"
		lastCheckinAt := ""
		completedDays := 0
		completedDaysSet := make(map[string]struct{}, len(allCompletedCheckins))
		challengeProgressEnd := dayStart
		if challenge.EndDate.Before(challengeProgressEnd) {
			challengeProgressEnd = challenge.EndDate
		}

		for _, checkin := range allCompletedCheckins {
			normalizedDay := normalizeDay(checkin.CheckinDate)
			if lastCheckinAt == "" {
				lastCheckinAt = checkin.CheckedInAt.Format(time.RFC3339)
			}
			if normalizedDay.Equal(dayStart) {
				todayStatus = "completed"
			}
			if normalizedDay.Before(challenge.StartDate) || normalizedDay.After(challengeProgressEnd) {
				continue
			}

			key := normalizedDay.Format("2006-01-02")
			if _, ok := completedDaysSet[key]; ok {
				continue
			}
			completedDaysSet[key] = struct{}{}
			completedDays++
		}

		challengeCompletionRate := 0
		if elapsedDays > 0 {
			challengeCompletionRate = int((float64(completedDays) / float64(elapsedDays)) * 100)
			if challengeCompletionRate > 100 {
				challengeCompletionRate = 100
			}
		}

		myProgresses = append(myProgresses, ChallengeMemberProgressResult{
			UserID:                  caller.IMUserID,
			Nickname:                caller.Nickname,
			Role:                    callerRole,
			TodayStatus:             todayStatus,
			CurrentStreak:           currentStreak,
			ChallengeCompletionRate: challengeCompletionRate,
			LastCheckinAt:           lastCheckinAt,
			CompletedDays:           completedDays,
			TargetDays:              elapsedDays,
		})
	}

	return &ChallengeDetailResult{
		ID:                   challenge.ID,
		Title:                challenge.Name,
		Description:          challenge.Description,
		GroupID:              challenge.IMGroupID,
		LinkedHabitID:        challenge.LinkedUserHabitID,
		LinkedHabitName:      habit.Name,
		StartDate:            challenge.StartDate.Format("2006-01-02"),
		EndDate:              challenge.EndDate.Format("2006-01-02"),
		MemberCount:          len(memberResults),
		CompletedToday:       0,
		DurationDays:         durationDays,
		ElapsedDays:          elapsedDays,
		CompletionRate:       0,
		Members:              memberResults,
		MemberProgresses:     myProgresses,
		RecordID:             challenge.ID,
		RecordTitle:          challenge.Name,
		RecordDescription:    challenge.Description,
		DiscussionGroupID:    challenge.IMGroupID,
		LinkedLedgerID:       challenge.LinkedUserHabitID,
		LinkedLedgerName:     habit.Name,
		ParticipantCount:     len(memberResults),
		RecordCompletionRate: 0,
		Participants:         memberResults,
		ParticipantProgress:  myProgresses,
	}, nil
}
