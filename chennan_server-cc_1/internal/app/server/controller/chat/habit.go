package controller_chat

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

// CreateHabitReq 表示创建习惯请求。
type CreateHabitReq struct {
	Name         string
	Icon         string
	GoalType     string
	GoalTarget   string
	ReminderTime string
}

// CheckinHabitReq 表示习惯打卡请求。
type CheckinHabitReq struct {
	Value string
}

// HabitDashboardSummary 表示打卡首页摘要。
type HabitDashboardSummary struct {
	CurrentStreak       int    `json:"current_streak"`
	MonthCompletionRate int    `json:"month_completion_rate"`
	CompletedToday      int    `json:"completed_today"`
	TotalToday          int    `json:"total_today"`
	CheckinDate         string `json:"checkin_date"`

	AccountingStreak   int    `json:"accounting_streak"`
	MonthCoverageRate  int    `json:"month_coverage_rate"`
	RecordedToday      int    `json:"recorded_today"`
	TotalLedgerEntries int    `json:"total_ledger_entries"`
	AccountingDate     string `json:"accounting_date"`
}

// HabitDashboardItem 表示打卡首页中的单个习惯。
type HabitDashboardItem struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Icon          string `json:"icon"`
	GoalType      string `json:"goal_type"`
	GoalTarget    string `json:"goal_target"`
	ReminderTime  string `json:"reminder_time"`
	TodayStatus   string `json:"today_status"`
	TodayValue    string `json:"today_value"`
	CurrentStreak int    `json:"current_streak"`
	BestStreak    int    `json:"best_streak"`
	CheckedInAt   string `json:"checked_in_at,omitempty"`
}

// HabitDashboardResult 表示打卡首页完整数据。
type HabitDashboardResult struct {
	Summary          HabitDashboardSummary `json:"summary"`
	Habits           []HabitDashboardItem  `json:"habits"`
	ActiveChallenges []ChallengeCard       `json:"active_challenges"`

	AccountingSummary HabitDashboardSummary `json:"accounting_summary"`
	Ledgers           []HabitDashboardItem  `json:"ledgers"`
	SplitRecords      []ChallengeCard       `json:"split_records"`
}

// CreateHabitResult 表示创建习惯返回结果。
type CreateHabitResult struct {
	ID uint `json:"id"`

	LedgerID uint `json:"ledger_id"`
}

// CheckinHabitResult 表示习惯打卡结果。
type CheckinHabitResult struct {
	TodayStatus string `json:"today_status"`
	TodayValue  string `json:"today_value"`
	CheckedInAt string `json:"checked_in_at,omitempty"`

	EntryStatus string `json:"entry_status"`
	EntryValue  string `json:"entry_value"`
	RecordedAt  string `json:"recorded_at,omitempty"`
}

// GetHabitDashboard 获取当前用户的打卡首页数据。
func GetHabitDashboard(_ context.Context, imUserID string) (*HabitDashboardResult, error) {
	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	habitRepo := repository.NewHabitRepository()
	habits, err := habitRepo.ListActiveHabitsByUser(user.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dayStart := normalizeDay(now)
	todayCheckins, err := habitRepo.ListCheckinsByUserAndDate(user.ID, dayStart)
	if err != nil {
		return nil, err
	}
	todayCheckinMap := make(map[uint]model.HabitCheckin, len(todayCheckins))
	completedToday := 0
	for _, checkin := range todayCheckins {
		todayCheckinMap[checkin.UserHabitID] = checkin
		if checkin.Status == "completed" {
			completedToday++
		}
	}

	items := make([]HabitDashboardItem, 0, len(habits))
	maxCurrentStreak := 0
	for _, habit := range habits {
		checkins, err := habitRepo.ListCheckinsByHabit(habit.ID)
		if err != nil {
			return nil, err
		}
		currentStreak, bestStreak := calculateStreaks(checkins, dayStart)
		if currentStreak > maxCurrentStreak {
			maxCurrentStreak = currentStreak
		}

		item := HabitDashboardItem{
			ID:            habit.ID,
			Name:          habit.Name,
			Icon:          habit.Icon,
			GoalType:      habit.GoalType,
			GoalTarget:    habit.GoalTarget,
			ReminderTime:  habit.ReminderTime,
			TodayStatus:   "pending",
			TodayValue:    "",
			CurrentStreak: currentStreak,
			BestStreak:    bestStreak,
		}
		if todayCheckin, ok := todayCheckinMap[habit.ID]; ok {
			item.TodayStatus = todayCheckin.Status
			item.TodayValue = todayCheckin.Value
			item.CheckedInAt = todayCheckin.CheckedInAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthDaysElapsed := now.Day()
	completedCount, err := habitRepo.CountCompletedCheckinsInRange(user.ID, monthStart, dayStart)
	if err != nil {
		return nil, err
	}
	monthCompletionRate := 0
	if len(habits) > 0 && monthDaysElapsed > 0 {
		totalTargets := len(habits) * monthDaysElapsed
		monthCompletionRate = int((float64(completedCount) / float64(totalTargets)) * 100)
		if monthCompletionRate > 100 {
			monthCompletionRate = 100
		}
	}

	summary := HabitDashboardSummary{
		CurrentStreak:       maxCurrentStreak,
		MonthCompletionRate: monthCompletionRate,
		CompletedToday:      completedToday,
		TotalToday:          len(habits),
		CheckinDate:         dayStart.Format("2006-01-02"),
		AccountingStreak:    maxCurrentStreak,
		MonthCoverageRate:   monthCompletionRate,
		RecordedToday:       completedToday,
		TotalLedgerEntries:  len(habits),
		AccountingDate:      dayStart.Format("2006-01-02"),
	}

	activeChallenges, err := listActiveChallengesForDashboard(user.ID, dayStart)
	if err != nil {
		return nil, err
	}

	return &HabitDashboardResult{
		Summary:           summary,
		Habits:            items,
		ActiveChallenges:  activeChallenges,
		AccountingSummary: summary,
		Ledgers:           items,
		SplitRecords:      activeChallenges,
	}, nil
}

// CreateHabit 创建当前用户的习惯。
func CreateHabit(_ context.Context, imUserID string, req CreateHabitReq) (*CreateHabitResult, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Icon = strings.TrimSpace(req.Icon)
	req.GoalType = strings.TrimSpace(req.GoalType)
	req.GoalTarget = strings.TrimSpace(req.GoalTarget)
	req.ReminderTime = strings.TrimSpace(req.ReminderTime)

	if req.Name == "" {
		return nil, errors.New("习惯名称不能为空")
	}
	if len([]rune(req.Name)) > 32 {
		return nil, errors.New("习惯名称不能超过 32 个字符")
	}
	if req.Icon == "" {
		req.Icon = "✅"
	}
	if req.GoalType != "binary" && req.GoalType != "numeric" && req.GoalType != "timer" {
		return nil, errors.New("不支持的打卡类型")
	}
	if req.GoalType == "numeric" || req.GoalType == "timer" {
		if _, err := strconv.Atoi(req.GoalTarget); err != nil || req.GoalTarget == "" {
			return nil, errors.New("数值型或计时型习惯的目标值必须是数字")
		}
	}
	if req.ReminderTime != "" && !isValidReminderTime(req.ReminderTime) {
		return nil, errors.New("提醒时间格式必须为 HH:mm")
	}

	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	habit := &model.UserHabit{
		UserID:       user.ID,
		Name:         req.Name,
		Icon:         req.Icon,
		GoalType:     req.GoalType,
		GoalTarget:   req.GoalTarget,
		ReminderTime: req.ReminderTime,
		Status:       "active",
	}

	habitRepo := repository.NewHabitRepository()
	if err := habitRepo.CreateHabit(habit); err != nil {
		return nil, err
	}

	return &CreateHabitResult{ID: habit.ID, LedgerID: habit.ID}, nil
}

// CheckinHabit 完成或更新今日打卡。
func CheckinHabit(_ context.Context, imUserID string, habitID uint, req CheckinHabitReq) (*CheckinHabitResult, error) {
	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	habitRepo := repository.NewHabitRepository()
	habit, err := habitRepo.FindHabitByIDAndUser(habitID, user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("习惯不存在")
		}
		return nil, err
	}

	dayStart := normalizeDay(time.Now())
	checkin, err := habitRepo.FindTodayCheckin(habit.ID, dayStart)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		checkin = &model.HabitCheckin{
			UserHabitID: habit.ID,
			UserID:      user.ID,
			CheckinDate: dayStart,
			Source:      "manual",
		}
	}

	value, status, err := resolveCheckinValueAndStatus(habit, strings.TrimSpace(req.Value), checkin.Value)
	if err != nil {
		return nil, err
	}

	checkin.Value = value
	checkin.Status = status
	checkin.CheckedInAt = time.Now()
	if err := habitRepo.SaveCheckin(checkin); err != nil {
		return nil, err
	}

	return &CheckinHabitResult{
		TodayStatus: checkin.Status,
		TodayValue:  checkin.Value,
		CheckedInAt: checkin.CheckedInAt.Format(time.RFC3339),
		EntryStatus: checkin.Status,
		EntryValue:  checkin.Value,
		RecordedAt:  checkin.CheckedInAt.Format(time.RFC3339),
	}, nil
}

func normalizeDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func calculateStreaks(checkins []model.HabitCheckin, today time.Time) (int, int) {
	if len(checkins) == 0 {
		return 0, 0
	}

	seen := make(map[string]struct{}, len(checkins))
	days := make([]time.Time, 0, len(checkins))
	for _, checkin := range checkins {
		day := normalizeDay(checkin.CheckinDate)
		key := day.Format("2006-01-02")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Before(days[j])
	})

	bestStreak := 1
	currentRun := 1
	for i := 1; i < len(days); i++ {
		if days[i].Sub(days[i-1]).Hours() == 24 {
			currentRun++
		} else {
			if currentRun > bestStreak {
				bestStreak = currentRun
			}
			currentRun = 1
		}
	}
	if currentRun > bestStreak {
		bestStreak = currentRun
	}

	currentStreak := 0
	lastDay := days[len(days)-1]
	if sameDay(lastDay, today) || sameDay(lastDay, today.AddDate(0, 0, -1)) {
		currentStreak = 1
		for i := len(days) - 1; i > 0; i-- {
			if days[i].Sub(days[i-1]).Hours() == 24 {
				currentStreak++
			} else {
				break
			}
		}
		if !sameDay(lastDay, today) {
			currentStreak = 0
		}
	}

	return currentStreak, bestStreak
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func isValidReminderTime(value string) bool {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return false
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func resolveCheckinValueAndStatus(habit *model.UserHabit, requestedValue, currentValue string) (string, string, error) {
	switch habit.GoalType {
	case "binary":
		return "1", "completed", nil
	case "numeric", "timer":
		target, _ := strconv.Atoi(habit.GoalTarget)
		valueText := requestedValue
		if valueText == "" {
			current := 0
			if currentValue != "" {
				if parsedCurrent, err := strconv.Atoi(currentValue); err == nil {
					current = parsedCurrent
				}
			}
			valueText = strconv.Itoa(current + 1)
		}
		value, err := strconv.Atoi(valueText)
		if err != nil {
			return "", "", fmt.Errorf("打卡值必须是数字")
		}
		status := "completed"
		if target > 0 && value < target {
			status = "in_progress"
		}
		return strconv.Itoa(value), status, nil
	default:
		return "", "", fmt.Errorf("不支持的打卡类型")
	}
}
