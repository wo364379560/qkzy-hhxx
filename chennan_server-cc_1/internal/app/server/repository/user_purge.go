package repository

import (
	"errors"
	"fmt"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

// UserPurgeSummary 汇总用户物理删除时产生的附带清理信息。
type UserPurgeSummary struct {
	ChallengeGroupIDs []string
}

// UserPurgePreview 表示用户物理删除前的本地影响预览。
type UserPurgePreview struct {
	CommunityPosts       int `json:"community_posts"`
	CommunityComments    int `json:"community_comments"`
	CommunityLikes       int `json:"community_likes"`
	Habits               int `json:"habits"`
	HabitCheckins        int `json:"habit_checkins"`
	Challenges           int `json:"challenges"`
	ChallengeMembers     int `json:"challenge_members"`
	ChallengeGroups      int `json:"challenge_groups"`
	OfflineRegistrations int `json:"offline_registrations"`
	Reports              int `json:"reports"`
}

type userPurgeTargets struct {
	habitIDs          []uint
	challengeIDs      []uint
	challengeGroupIDs []string
	postIDs           []uint
	commentIDs        []uint
	activityIDs       []uint
}

// PreviewPurgeAccount 预览用户物理删除将清理的本地数据规模。
func (r *UserRepository) PreviewPurgeAccount(user *model.User) (*UserPurgePreview, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	targets, err := collectUserPurgeTargets(r.db, user)
	if err != nil {
		return nil, err
	}

	preview := &UserPurgePreview{
		Habits:               len(targets.habitIDs),
		Challenges:           len(targets.challengeIDs),
		ChallengeGroups:      len(targets.challengeGroupIDs),
		CommunityPosts:       len(targets.postIDs),
		CommunityComments:    len(targets.commentIDs),
		OfflineRegistrations: len(targets.activityIDs),
	}

	var communityLikes int64
	if err := r.db.Model(&model.CommunityPostLike{}).
		Where("user_id = ?", user.ID).
		Or("post_id IN ?", targets.postIDs).
		Distinct("id").
		Count(&communityLikes).Error; err != nil {
		return nil, err
	}
	preview.CommunityLikes = int(communityLikes)

	var challengeMembers int64
	challengeMemberQuery := r.db.Model(&model.ChallengeMember{}).Where("user_id = ?", user.ID)
	if len(targets.challengeIDs) > 0 {
		challengeMemberQuery = challengeMemberQuery.Or("challenge_id IN ?", targets.challengeIDs)
	}
	if err := challengeMemberQuery.Distinct("id").Count(&challengeMembers).Error; err != nil {
		return nil, err
	}
	preview.ChallengeMembers = int(challengeMembers)

	var habitCheckins int64
	habitCheckinQuery := r.db.Model(&model.HabitCheckin{}).Where("user_id = ?", user.ID)
	if len(targets.habitIDs) > 0 {
		habitCheckinQuery = habitCheckinQuery.Or("user_habit_id IN ?", targets.habitIDs)
	}
	if err := habitCheckinQuery.Distinct("id").Count(&habitCheckins).Error; err != nil {
		return nil, err
	}
	preview.HabitCheckins = int(habitCheckins)

	reportIDs := make([]uint, 0)
	targetIMIDColumn := reportTargetIMIDColumn(r.db)
	if err := r.db.Model(&model.Report{}).
		Where("reporter_id = ?", user.ID).
		Or("target_type = ? AND target_id = ?", "user", user.ID).
		Or(
			fmt.Sprintf("target_type = ? AND %s = ?", targetIMIDColumn),
			"user",
			user.IMUserID,
		).
		Pluck("id", &reportIDs).Error; err != nil {
		return nil, err
	}
	if len(targets.postIDs) > 0 {
		postReportIDs := make([]uint, 0)
		if err := r.db.Model(&model.Report{}).
			Where("target_type = ? AND target_id IN ?", "community_post", targets.postIDs).
			Pluck("id", &postReportIDs).Error; err != nil {
			return nil, err
		}
		reportIDs = append(reportIDs, postReportIDs...)
	}
	if len(targets.commentIDs) > 0 {
		commentReportIDs := make([]uint, 0)
		if err := r.db.Model(&model.Report{}).
			Where("target_type = ? AND target_id IN ?", "community_comment", targets.commentIDs).
			Pluck("id", &commentReportIDs).Error; err != nil {
			return nil, err
		}
		reportIDs = append(reportIDs, commentReportIDs...)
	}
	if len(targets.challengeGroupIDs) > 0 {
		groupReportIDs := make([]uint, 0)
		if err := r.db.Model(&model.Report{}).
			Where(
				fmt.Sprintf("target_type = ? AND %s IN ?", targetIMIDColumn),
				"group",
				targets.challengeGroupIDs,
			).
			Pluck("id", &groupReportIDs).Error; err != nil {
			return nil, err
		}
		reportIDs = append(reportIDs, groupReportIDs...)
	}
	preview.Reports = len(uniqueUints(reportIDs))

	return preview, nil
}

// PurgeAccount 物理删除用户及其依赖的本地业务数据。
func (r *UserRepository) PurgeAccount(user *model.User) (*UserPurgeSummary, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	summary := &UserPurgeSummary{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var habitIDs []uint
		if err := tx.Model(&model.UserHabit{}).
			Where("user_id = ?", user.ID).
			Pluck("id", &habitIDs).Error; err != nil {
			return err
		}

		var challenges []model.Challenge
		if err := tx.Where("creator_user_id = ?", user.ID).Find(&challenges).Error; err != nil {
			return err
		}
		challengeIDs := make([]uint, 0, len(challenges))
		challengeGroupIDs := make([]string, 0, len(challenges))
		for _, challenge := range challenges {
			challengeIDs = append(challengeIDs, challenge.ID)
			if challenge.IMGroupID != "" {
				challengeGroupIDs = append(challengeGroupIDs, challenge.IMGroupID)
			}
		}
		summary.ChallengeGroupIDs = uniqueStrings(challengeGroupIDs)

		var postIDs []uint
		if err := tx.Model(&model.CommunityPost{}).
			Where("user_id = ?", user.ID).
			Pluck("id", &postIDs).Error; err != nil {
			return err
		}

		commentIDs := make([]uint, 0)
		var authoredCommentIDs []uint
		if err := tx.Model(&model.CommunityPostComment{}).
			Where("user_id = ?", user.ID).
			Pluck("id", &authoredCommentIDs).Error; err != nil {
			return err
		}
		commentIDs = append(commentIDs, authoredCommentIDs...)

		if len(postIDs) > 0 {
			var postCommentIDs []uint
			if err := tx.Model(&model.CommunityPostComment{}).
				Where("post_id IN ?", postIDs).
				Pluck("id", &postCommentIDs).Error; err != nil {
				return err
			}
			commentIDs = append(commentIDs, postCommentIDs...)
		}
		commentIDs = uniqueUints(commentIDs)

		var activityIDs []uint
		if err := tx.Model(&model.OfflineActivityRegistration{}).
			Where("user_id = ?", user.ID).
			Pluck("activity_id", &activityIDs).Error; err != nil {
			return err
		}
		activityIDs = uniqueUints(activityIDs)

		if len(habitIDs) > 0 {
			if err := tx.Model(&model.CommunityPost{}).
				Where("linked_user_habit_id IN ?", habitIDs).
				Update("linked_user_habit_id", nil).Error; err != nil {
				return err
			}
		}

		if len(challengeIDs) > 0 {
			if err := tx.Model(&model.CommunityPost{}).
				Where("linked_challenge_id IN ?", challengeIDs).
				Update("linked_challenge_id", nil).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("reporter_id = ?", user.ID).Delete(&model.Report{}).Error; err != nil {
			return err
		}

		targetIMIDColumn := reportTargetIMIDColumn(tx)
		reportUserQuery := tx.Where("target_type = ? AND target_id = ?", "user", user.ID)
		if user.IMUserID != "" {
			reportUserQuery = reportUserQuery.Or(
				fmt.Sprintf("target_type = ? AND %s = ?", targetIMIDColumn),
				"user",
				user.IMUserID,
			)
		}
		if err := reportUserQuery.Delete(&model.Report{}).Error; err != nil {
			return err
		}

		if len(postIDs) > 0 {
			if err := tx.Where("target_type = ? AND target_id IN ?", "community_post", postIDs).
				Delete(&model.Report{}).Error; err != nil {
				return err
			}
		}

		if len(commentIDs) > 0 {
			if err := tx.Where("target_type = ? AND target_id IN ?", "community_comment", commentIDs).
				Delete(&model.Report{}).Error; err != nil {
				return err
			}
		}

		if len(summary.ChallengeGroupIDs) > 0 {
			if err := tx.Where(
				fmt.Sprintf("target_type = ? AND %s IN ?", targetIMIDColumn),
				"group",
				summary.ChallengeGroupIDs,
			).
				Delete(&model.Report{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", user.ID).Delete(&model.CommunityPostLike{}).Error; err != nil {
			return err
		}
		if len(postIDs) > 0 {
			if err := tx.Where("post_id IN ?", postIDs).Delete(&model.CommunityPostLike{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", user.ID).Delete(&model.CommunityPostComment{}).Error; err != nil {
			return err
		}
		if len(postIDs) > 0 {
			if err := tx.Where("post_id IN ?", postIDs).Delete(&model.CommunityPostComment{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", user.ID).Delete(&model.CommunityPost{}).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			UPDATE community_posts cp
			LEFT JOIN (
				SELECT post_id, COUNT(*) AS cnt
				FROM community_post_likes
				GROUP BY post_id
			) likes_stat ON likes_stat.post_id = cp.id
			LEFT JOIN (
				SELECT post_id, COUNT(*) AS cnt
				FROM community_post_comments
				WHERE status = 'active'
				GROUP BY post_id
			) comments_stat ON comments_stat.post_id = cp.id
			SET
				cp.like_count = COALESCE(likes_stat.cnt, 0),
				cp.comment_count = COALESCE(comments_stat.cnt, 0)
		`).Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ?", user.ID).Delete(&model.ChallengeMember{}).Error; err != nil {
			return err
		}
		if len(challengeIDs) > 0 {
			if err := tx.Where("challenge_id IN ?", challengeIDs).Delete(&model.ChallengeMember{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("creator_user_id = ?", user.ID).Delete(&model.Challenge{}).Error; err != nil {
			return err
		}

		if len(summary.ChallengeGroupIDs) > 0 {
			if err := tx.Where("im_group_id IN ?", summary.ChallengeGroupIDs).Delete(&model.Group{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", user.ID).Delete(&model.HabitCheckin{}).Error; err != nil {
			return err
		}
		if len(habitIDs) > 0 {
			if err := tx.Where("user_habit_id IN ?", habitIDs).Delete(&model.HabitCheckin{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&model.UserHabit{}).Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ?", user.ID).Delete(&model.OfflineActivityRegistration{}).Error; err != nil {
			return err
		}
		if len(activityIDs) > 0 {
			if err := tx.Exec(`
				UPDATE offline_activities oa
				LEFT JOIN (
					SELECT activity_id, COUNT(*) AS cnt
					FROM offline_activity_registrations
					WHERE status = 'registered'
					GROUP BY activity_id
				) reg_stat ON reg_stat.activity_id = oa.id
				SET oa.registration_count = COALESCE(reg_stat.cnt, 0)
				WHERE oa.id IN ?
			`, activityIDs).Error; err != nil {
				return err
			}
		}

		if tx.Migrator().HasTable(&model.UserRefreshSession{}) {
			if err := tx.Where("user_id = ?", user.ID).Delete(&model.UserRefreshSession{}).Error; err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable(&model.UserSecuritySettings{}) {
			if err := tx.Where("user_id = ?", user.ID).Delete(&model.UserSecuritySettings{}).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&model.User{}, user.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func collectUserPurgeTargets(db *gorm.DB, user *model.User) (*userPurgeTargets, error) {
	targets := &userPurgeTargets{}

	if err := db.Model(&model.UserHabit{}).
		Where("user_id = ?", user.ID).
		Pluck("id", &targets.habitIDs).Error; err != nil {
		return nil, err
	}

	var challenges []model.Challenge
	if err := db.Where("creator_user_id = ?", user.ID).Find(&challenges).Error; err != nil {
		return nil, err
	}
	targets.challengeIDs = make([]uint, 0, len(challenges))
	targets.challengeGroupIDs = make([]string, 0, len(challenges))
	for _, challenge := range challenges {
		targets.challengeIDs = append(targets.challengeIDs, challenge.ID)
		if challenge.IMGroupID != "" {
			targets.challengeGroupIDs = append(targets.challengeGroupIDs, challenge.IMGroupID)
		}
	}
	targets.challengeGroupIDs = uniqueStrings(targets.challengeGroupIDs)

	if err := db.Model(&model.CommunityPost{}).
		Where("user_id = ?", user.ID).
		Pluck("id", &targets.postIDs).Error; err != nil {
		return nil, err
	}

	commentIDs := make([]uint, 0)
	var authoredCommentIDs []uint
	if err := db.Model(&model.CommunityPostComment{}).
		Where("user_id = ?", user.ID).
		Pluck("id", &authoredCommentIDs).Error; err != nil {
		return nil, err
	}
	commentIDs = append(commentIDs, authoredCommentIDs...)
	if len(targets.postIDs) > 0 {
		var postCommentIDs []uint
		if err := db.Model(&model.CommunityPostComment{}).
			Where("post_id IN ?", targets.postIDs).
			Pluck("id", &postCommentIDs).Error; err != nil {
			return nil, err
		}
		commentIDs = append(commentIDs, postCommentIDs...)
	}
	targets.commentIDs = uniqueUints(commentIDs)

	if err := db.Model(&model.OfflineActivityRegistration{}).
		Where("user_id = ?", user.ID).
		Pluck("activity_id", &targets.activityIDs).Error; err != nil {
		return nil, err
	}
	targets.activityIDs = uniqueUints(targets.activityIDs)

	return targets, nil
}

func uniqueUints(values []uint) []uint {
	if len(values) == 0 {
		return values
	}
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func reportTargetIMIDColumn(db *gorm.DB) string {
	if db == nil || db.NamingStrategy == nil {
		return "target_im_id"
	}
	return db.NamingStrategy.ColumnName("", "TargetIMID")
}
