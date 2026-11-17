package repository

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// ChallengeRepository 处理挑战与成员数据。
type ChallengeRepository struct {
	db *gorm.DB
}

// ActiveChallengeRow 表示带成员数的挑战摘要。
type ActiveChallengeRow struct {
	ChallengeID       uint      `gorm:"column:challenge_id"`
	Name              string    `gorm:"column:name"`
	Description       string    `gorm:"column:description"`
	LinkedUserHabitID uint      `gorm:"column:linked_user_habit_id"`
	StartDate         time.Time `gorm:"column:start_date"`
	EndDate           time.Time `gorm:"column:end_date"`
	IMGroupID         string    `gorm:"column:im_group_id"`
	MemberCount       int64     `gorm:"column:member_count"`
}

// NewChallengeRepository 创建 ChallengeRepository。
func NewChallengeRepository() *ChallengeRepository {
	return &ChallengeRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// Create 创建挑战。
func (r *ChallengeRepository) Create(challenge *model.Challenge) error {
	return r.db.Create(challenge).Error
}

// CreateMembers 批量创建挑战成员。
func (r *ChallengeRepository) CreateMembers(members []model.ChallengeMember) error {
	if len(members) == 0 {
		return nil
	}
	return r.db.Create(&members).Error
}

// FindByID 查询挑战。
func (r *ChallengeRepository) FindByID(challengeID uint) (*model.Challenge, error) {
	var challenge model.Challenge
	if err := r.db.First(&challenge, challengeID).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

// ListMembersByChallenge 查询挑战成员。
func (r *ChallengeRepository) ListMembersByChallenge(challengeID uint) ([]model.ChallengeMember, error) {
	var members []model.ChallengeMember
	err := r.db.
		Where("challenge_id = ? AND join_status = 'active'", challengeID).
		Order("id ASC").
		Find(&members).Error
	return members, err
}

// ListActiveByUser 查询用户参与中的挑战摘要。
func (r *ChallengeRepository) ListActiveByUser(userID uint, day time.Time) ([]ActiveChallengeRow, error) {
	var rows []ActiveChallengeRow
	err := r.db.
		Table("challenges AS c").
		Select(`c.id AS challenge_id, c.name, c.description, c.linked_user_habit_id, c.start_date, c.end_date, c.im_group_id,
			(SELECT COUNT(*) FROM challenge_members cm2 WHERE cm2.challenge_id = c.id AND cm2.join_status = 'active') AS member_count`).
		Joins("JOIN challenge_members cm ON cm.challenge_id = c.id").
		Where("cm.user_id = ? AND cm.join_status = 'active'", userID).
		Where("c.status = 'active' AND c.start_date <= ? AND c.end_date >= ?", day, day).
		Order("c.end_date ASC, c.id DESC").
		Scan(&rows).Error
	return rows, err
}

// ListActiveByUserAndHabit 查询用户在指定习惯下参与中的挑战。
func (r *ChallengeRepository) ListActiveByUserAndHabit(userID, habitID uint, day time.Time) ([]model.Challenge, error) {
	var challenges []model.Challenge
	err := r.db.
		Table("challenges AS c").
		Select("c.*").
		Joins("JOIN challenge_members cm ON cm.challenge_id = c.id").
		Where("cm.user_id = ? AND cm.join_status = 'active'", userID).
		Where("c.linked_user_habit_id = ? AND c.status = 'active' AND c.start_date <= ? AND c.end_date >= ?", habitID, day, day).
		Order("c.id DESC").
		Find(&challenges).Error
	return challenges, err
}
