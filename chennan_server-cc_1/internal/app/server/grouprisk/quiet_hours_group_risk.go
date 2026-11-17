package grouprisk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/quietlog"
	"github.com/ixayda/ix-Go/internal/app/server/quietpolicy"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/idgs"
	"github.com/ixayda/ix-Go/pkg/logger"
)

const (
	StatusPending         = "pending"
	StatusAutoDismissed   = "auto_dismissed"
	StatusManualDismissed = "manual_dismissed"
	StatusResolved        = "resolved"
	StatusProtected       = "protected"
	StatusRiskOnly        = "risk_only"
	StatusAPIFailed       = "api_failed"

	SourceAfterCreate     = "im_after_create_group"
	SourceBeforeSendGroup = "im_before_send_group_msg"

	ActionAutoDismissGroup   = "auto_dismiss_group"
	ActionManualDismissGroup = "manual_dismiss_group"
	ActionRiskOnly           = "risk_only"

	afterCreateDismissWindow = 5 * time.Minute
	beforeSendDismissWindow  = 5 * time.Hour
)

// GroupCreatedInput is the normalized Tencent after-create callback payload.
type GroupCreatedInput struct {
	CallbackCommand string
	GroupID         string
	GroupName       string
	GroupType       string
	OwnerAccount    string
	OperatorAccount string
	CreateTimeUnix  int64
	TriggerSource   string
	DryRun          bool
}

// BeforeSendInput is the normalized Tencent group-message before-send payload.
type BeforeSendInput struct {
	CallbackCommand string
	GroupID         string
	FromAccount     string
	DryRun          bool
}

// BeforeSendDecision describes whether the current group message must be blocked.
type BeforeSendDecision struct {
	Blocked bool
	Reason  string
}

type groupSnapshot struct {
	IMGroupID         string
	GroupName         string
	GroupType         string
	OwnerAccount      string
	TencentCreateTime *time.Time
	LocalGroup        *model.Group
	OwnerUser         *model.User
}

// HandleAfterCreateGroup syncs a newly created Tencent group and dismisses it when it violates quiet-hours create rules.
func HandleAfterCreateGroup(ctx context.Context, input GroupCreatedInput) error {
	input.GroupID = strings.TrimSpace(input.GroupID)
	if input.GroupID == "" {
		return errors.New("group id is empty")
	}
	if input.TriggerSource == "" {
		input.TriggerSource = SourceAfterCreate
	}

	snapshot, err := loadGroupSnapshot(ctx, input.GroupID, input.GroupName, input.GroupType, firstNonEmpty(input.OwnerAccount, input.OperatorAccount), input.CreateTimeUnix)
	if err != nil {
		return err
	}
	if snapshot.LocalGroup != nil && snapshot.LocalGroup.Status == "dismissed" {
		return nil
	}

	evaluation, err := evaluateCreateViolation(ctx, snapshot, time.Now())
	if err != nil {
		return err
	}
	if !evaluation.shouldRecord {
		return nil
	}

	if evaluation.protected {
		_, err = upsertRisk(snapshot, input, evaluation, StatusProtected, ActionRiskOnly, "群在保护名单内，只记录风险不自动解散", "")
		return err
	}
	if !evaluation.canAutoDismiss {
		_, err = upsertRisk(snapshot, input, evaluation, evaluation.status, ActionRiskOnly, evaluation.reason, "")
		return err
	}

	if snapshot.TencentCreateTime == nil {
		_, err = upsertRisk(snapshot, input, evaluation, StatusRiskOnly, ActionRiskOnly, "拿不到腾讯真实 CreateTime，只记录风险", "")
		return err
	}
	if time.Since(*snapshot.TencentCreateTime) > afterCreateDismissWindow {
		_, err = upsertRisk(snapshot, input, evaluation, StatusPending, ActionRiskOnly, "创建后回调超过 5 分钟，转入发送前兜底", "")
		return err
	}

	resultJSON, dismissErr := dismissTencentGroup(input.GroupID, input.DryRun)
	status := StatusAutoDismissed
	reason := evaluation.reason
	if dismissErr != nil {
		status = StatusAPIFailed
		reason = "腾讯解散接口失败，转入后台待处理：" + dismissErr.Error()
	}
	risk, err := upsertRisk(snapshot, input, evaluation, status, ActionAutoDismissGroup, reason, resultJSON)
	if err != nil {
		return err
	}
	if dismissErr == nil {
		if err := markLocalGroupDismissed(snapshot.LocalGroup); err != nil {
			return err
		}
		recordQuietHit(input.CallbackCommand, SourceAfterCreate, snapshot, reason)
		now := time.Now()
		_ = datasource.Datasource.Mysql.GetClient().Model(risk).Updates(map[string]any{
			"dismissed_at": &now,
		}).Error
	}
	return dismissErr
}

// HandleBeforeSendGroupMessage dismisses a recently created violating group and blocks the current message.
func HandleBeforeSendGroupMessage(ctx context.Context, input BeforeSendInput) (BeforeSendDecision, error) {
	input.GroupID = strings.TrimSpace(input.GroupID)
	if input.GroupID == "" {
		return BeforeSendDecision{}, nil
	}

	db := datasource.Datasource.Mysql.GetClient()
	var existing model.QuietHoursGroupRisk
	existingErr := db.Where("im_group_id = ?", input.GroupID).First(&existing).Error
	if existingErr == nil {
		switch existing.Status {
		case StatusAutoDismissed, StatusManualDismissed:
			return BeforeSendDecision{Blocked: true, Reason: "该群已因静默时段违规建群被解散，消息已阻断"}, nil
		case StatusProtected, StatusResolved, StatusAPIFailed:
			return BeforeSendDecision{}, nil
		}
	} else if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
		return BeforeSendDecision{}, existingErr
	}

	snapshot, err := loadGroupSnapshot(ctx, input.GroupID, "", "", "", 0)
	if err != nil {
		return BeforeSendDecision{}, err
	}
	evaluation, err := evaluateCreateViolation(ctx, snapshot, time.Now())
	if err != nil {
		return BeforeSendDecision{}, err
	}
	if !evaluation.shouldRecord {
		return BeforeSendDecision{}, nil
	}

	groupInput := GroupCreatedInput{
		CallbackCommand: input.CallbackCommand,
		GroupID:         input.GroupID,
		GroupName:       snapshot.GroupName,
		GroupType:       snapshot.GroupType,
		OwnerAccount:    snapshot.OwnerAccount,
		TriggerSource:   SourceBeforeSendGroup,
		DryRun:          input.DryRun,
	}

	if evaluation.protected {
		_, err = upsertRisk(snapshot, groupInput, evaluation, StatusProtected, ActionRiskOnly, "群在保护名单内，只记录风险不自动解散", "")
		return BeforeSendDecision{}, err
	}
	if !evaluation.canAutoDismiss {
		_, err = upsertRisk(snapshot, groupInput, evaluation, evaluation.status, ActionRiskOnly, evaluation.reason, "")
		return BeforeSendDecision{}, err
	}
	if snapshot.TencentCreateTime == nil {
		_, err = upsertRisk(snapshot, groupInput, evaluation, StatusRiskOnly, ActionRiskOnly, "拿不到腾讯真实 CreateTime，只记录风险", "")
		return BeforeSendDecision{}, err
	}
	if time.Since(*snapshot.TencentCreateTime) > beforeSendDismissWindow {
		_, err = upsertRisk(snapshot, groupInput, evaluation, StatusPending, ActionRiskOnly, "违规建群已超过 5 小时，进入后台待处理", "")
		return BeforeSendDecision{}, err
	}

	sendBlocked, err := config.IsChatQuietHoursActionBlockedNow(evaluation.policy, config.ChatQuietHoursActionSendMessage, time.Now())
	if err != nil {
		return BeforeSendDecision{}, err
	}
	if !sendBlocked {
		_, err = upsertRisk(snapshot, groupInput, evaluation, StatusPending, ActionRiskOnly, "当前不在禁止发消息时段，进入后台待处理", "")
		return BeforeSendDecision{}, err
	}

	resultJSON, dismissErr := dismissTencentGroup(input.GroupID, input.DryRun)
	status := StatusAutoDismissed
	reason := evaluation.reason
	if dismissErr != nil {
		status = StatusAPIFailed
		reason = "腾讯解散接口失败，转入后台待处理：" + dismissErr.Error()
	}
	risk, err := upsertRisk(snapshot, groupInput, evaluation, status, ActionAutoDismissGroup, reason, resultJSON)
	if err != nil {
		return BeforeSendDecision{}, err
	}
	if dismissErr != nil {
		return BeforeSendDecision{}, dismissErr
	}
	if err := markLocalGroupDismissed(snapshot.LocalGroup); err != nil {
		return BeforeSendDecision{}, err
	}
	recordQuietHit(input.CallbackCommand, SourceBeforeSendGroup, snapshot, reason)
	now := time.Now()
	_ = db.Model(risk).Updates(map[string]any{"dismissed_at": &now}).Error
	return BeforeSendDecision{Blocked: true, Reason: "该群为静默时段违规新建群，已自动解散，本次消息已阻断"}, nil
}

// SyncGroupFromTencent refreshes the local groups mirror using Tencent group info without dismissing the group.
func SyncGroupFromTencent(ctx context.Context, groupID string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("group id is empty")
	}
	_, err := loadGroupSnapshot(ctx, groupID, "", "", "", 0)
	return err
}

type evaluationResult struct {
	shouldRecord   bool
	canAutoDismiss bool
	protected      bool
	status         string
	reason         string
	policy         config.ChatQuietHoursPolicy
	resolution     quietpolicy.ChatQuietHoursPolicyResolution
}

func evaluateCreateViolation(ctx context.Context, snapshot groupSnapshot, now time.Time) (evaluationResult, error) {
	if snapshot.OwnerUser == nil || snapshot.OwnerUser.OwnerID == 0 {
		return evaluationResult{shouldRecord: true, status: StatusRiskOnly, reason: "群 owner 无法解析归属，只记录风险"}, nil
	}
	if isProtected(snapshot.IMGroupID) {
		return evaluationResult{shouldRecord: true, protected: true, status: StatusProtected}, nil
	}
	resolution, err := quietpolicy.ResolveChatQuietHoursPolicyForOwner(ctx, snapshot.OwnerUser.OwnerID)
	if err != nil {
		return evaluationResult{}, err
	}
	if !resolution.HasPolicy || !resolution.Policy.Enabled {
		return evaluationResult{}, nil
	}
	if !resolution.Policy.BlockCreateGroup {
		return evaluationResult{shouldRecord: true, status: StatusRiskOnly, reason: "策略未启用禁止建群，只记录风险", policy: resolution.Policy, resolution: resolution}, nil
	}
	if snapshot.TencentCreateTime == nil {
		return evaluationResult{shouldRecord: true, status: StatusRiskOnly, reason: "拿不到腾讯真实 CreateTime，只记录风险", policy: resolution.Policy, resolution: resolution}, nil
	}
	blockedAtCreate, err := config.IsChatQuietHoursActionBlockedNow(resolution.Policy, config.ChatQuietHoursActionCreateGroup, *snapshot.TencentCreateTime)
	if err != nil {
		return evaluationResult{}, err
	}
	if !blockedAtCreate {
		return evaluationResult{}, nil
	}
	if now.Sub(*snapshot.TencentCreateTime) < 0 {
		return evaluationResult{shouldRecord: true, status: StatusRiskOnly, reason: "腾讯 CreateTime 晚于当前时间，只记录风险", policy: resolution.Policy, resolution: resolution}, nil
	}
	reason := fmt.Sprintf("静默时段违规建群：真实创建时间 %s 命中禁止建群时段 %s-%s",
		snapshot.TencentCreateTime.Format("2006-01-02 15:04:05"),
		resolution.Policy.Start,
		resolution.Policy.End,
	)
	return evaluationResult{
		shouldRecord:   true,
		canAutoDismiss: true,
		status:         StatusPending,
		reason:         reason,
		policy:         resolution.Policy,
		resolution:     resolution,
	}, nil
}

func loadGroupSnapshot(ctx context.Context, groupID, name, groupType, ownerAccount string, createTimeUnix int64) (groupSnapshot, error) {
	snapshot := groupSnapshot{
		IMGroupID:    strings.TrimSpace(groupID),
		GroupName:    strings.TrimSpace(name),
		GroupType:    strings.TrimSpace(groupType),
		OwnerAccount: strings.TrimSpace(ownerAccount),
	}
	if createTimeUnix > 0 {
		t := time.Unix(createTimeUnix, 0)
		snapshot.TencentCreateTime = &t
	}

	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		infoMap, err := imClient.GroupGetInfo([]string{snapshot.IMGroupID})
		if err != nil {
			logger.Warnf("load tencent group info failed: group=%s err=%v", snapshot.IMGroupID, err)
		} else if info, ok := infoMap[snapshot.IMGroupID]; ok {
			if snapshot.GroupName == "" {
				snapshot.GroupName = strings.TrimSpace(info.Name)
			}
			if snapshot.GroupType == "" {
				snapshot.GroupType = strings.TrimSpace(info.Type)
			}
			if snapshot.OwnerAccount == "" {
				snapshot.OwnerAccount = strings.TrimSpace(info.OwnerAccount)
			}
			if snapshot.TencentCreateTime == nil && info.CreateTime > 0 {
				t := time.Unix(info.CreateTime, 0)
				snapshot.TencentCreateTime = &t
			}
		}
	}
	if snapshot.GroupName == "" {
		snapshot.GroupName = snapshot.IMGroupID
	}
	if snapshot.GroupType == "" {
		snapshot.GroupType = "Work"
	}

	if snapshot.OwnerAccount != "" {
		if ownerUser, err := repository.NewUserRepository().FindByIMUserID(snapshot.OwnerAccount); err == nil {
			snapshot.OwnerUser = ownerUser
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return snapshot, err
		}
	}

	localGroup, err := syncLocalGroup(snapshot)
	if err != nil {
		return snapshot, err
	}
	snapshot.LocalGroup = localGroup
	_ = ctx
	return snapshot, nil
}

func syncLocalGroup(snapshot groupSnapshot) (*model.Group, error) {
	db := datasource.Datasource.Mysql.GetClient()
	var existing model.Group
	err := db.Where("im_group_id = ?", snapshot.IMGroupID).First(&existing).Error
	createAt := time.Now()
	if snapshot.TencentCreateTime != nil {
		createAt = *snapshot.TencentCreateTime
	}

	ownerID := uint(0)
	createdBy := uint(0)
	if snapshot.OwnerUser != nil {
		ownerID = snapshot.OwnerUser.OwnerID
		createdBy = snapshot.OwnerUser.ID
	}

	switch {
	case err == nil:
		existing.Name = snapshot.GroupName
		existing.GroupType = snapshot.GroupType
		if createdBy > 0 {
			existing.CreatedBy = createdBy
		}
		if err := repository.NewGroupRepositoryWithDB(db).SaveMirror(context.Background(), &existing); err != nil {
			return nil, err
		}

		_ = db.Where("im_group_id = ?", snapshot.IMGroupID).First(&existing).Error
		return &existing, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 归属未解析时不落 owner_id=0 的脏数据，否则刚建群邀请会误报 10101
		if ownerID == 0 {
			logger.Warnf("skip local group mirror without owner: group=%s name=%s", snapshot.IMGroupID, snapshot.GroupName)
			return nil, nil
		}
		group := model.Group{
			BizID:     idgs.Global.GroupGenerate().Int64(),
			IMGroupID: snapshot.IMGroupID,
			Name:      snapshot.GroupName,
			GroupType: snapshot.GroupType,
			OwnerID:   ownerID,
			CreatedBy: createdBy,
			Status:    "active",
			CreatedAt: createAt,
		}
		if err := repository.NewGroupRepositoryWithDB(db).SaveMirror(context.Background(), &group); err != nil {
			return nil, err
		}
		return &group, nil
	default:
		return nil, err
	}
}

func upsertRisk(snapshot groupSnapshot, input GroupCreatedInput, evaluation evaluationResult, status, action, reason, tencentResult string) (*model.QuietHoursGroupRisk, error) {
	now := time.Now()
	var localGroupID *uint
	if snapshot.LocalGroup != nil {
		id := snapshot.LocalGroup.ID
		localGroupID = &id
	}
	var ownerUserID *uint
	var ownerID *uint
	if snapshot.OwnerUser != nil {
		uid := snapshot.OwnerUser.ID
		oid := snapshot.OwnerUser.OwnerID
		ownerUserID = &uid
		ownerID = &oid
	}
	policyAdminID := evaluation.resolution.AdminUserID
	row := model.QuietHoursGroupRisk{
		IMGroupID:         snapshot.IMGroupID,
		GroupName:         snapshot.GroupName,
		GroupType:         snapshot.GroupType,
		LocalGroupID:      localGroupID,
		OwnerIMUserID:     snapshot.OwnerAccount,
		OwnerUserID:       ownerUserID,
		OwnerID:           ownerID,
		PolicyScope:       evaluation.resolution.Scope,
		PolicyAdminUserID: policyAdminID,
		PolicyStart:       evaluation.policy.Start,
		PolicyEnd:         evaluation.policy.End,
		TencentCreateTime: snapshot.TencentCreateTime,
		TriggerSource:     input.TriggerSource,
		CallbackCommand:   input.CallbackCommand,
		Action:            action,
		Status:            status,
		Reason:            reason,
		TencentResult:     tencentResult,
		DryRun:            input.DryRun,
		LastCheckedAt:     &now,
	}
	if status == StatusAutoDismissed || status == StatusManualDismissed {
		row.DismissedAt = &now
	}

	db := datasource.Datasource.Mysql.GetClient()
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "im_group_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"group_name", "group_type", "local_group_id", "owner_im_user_id", "owner_user_id", "owner_id",
			"policy_scope", "policy_admin_user_id", "policy_start", "policy_end", "tencent_create_time",
			"trigger_source", "callback_command", "action", "status", "reason", "tencent_result", "dry_run",
			"last_checked_at", "dismissed_at", "updated_at",
		}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	var saved model.QuietHoursGroupRisk
	if err := db.Where("im_group_id = ?", snapshot.IMGroupID).First(&saved).Error; err != nil {
		return nil, err
	}
	return &saved, nil
}

func dismissTencentGroup(groupID string, dryRun bool) (string, error) {
	if dryRun {
		data, _ := json.Marshal(map[string]any{"dry_run": true, "group_id": groupID})
		return string(data), nil
	}
	client := tencentim.NewClient()
	if !client.IsConfigured() {
		return "", errors.New("tencent im is not configured")
	}
	err := client.GroupDestroy(groupID)
	result := map[string]any{"group_id": groupID, "ok": err == nil}
	if err != nil {
		result["error"] = err.Error()
	}
	data, _ := json.Marshal(result)
	return string(data), err
}

func markLocalGroupDismissed(group *model.Group) error {
	if group == nil {
		return nil
	}
	return datasource.Datasource.Mysql.GetClient().
		Model(&model.Group{}).
		Where("id = ?", group.ID).
		Update("status", "dismissed").Error
}

func recordQuietHit(callbackCommand, source string, snapshot groupSnapshot, reason string) {
	actor := snapshot.OwnerAccount
	_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
		Action:          ActionAutoDismissGroup,
		Source:          source,
		CallbackCommand: callbackCommand,
		ActorIMUserID:   actor,
		TargetGroupID:   snapshot.IMGroupID,
		Reason:          reason,
	})
}

func isProtected(groupID string) bool {
	var count int64
	_ = datasource.Datasource.Mysql.GetClient().
		Model(&model.QuietHoursGroupProtection{}).
		Where("im_group_id = ? AND status = 'active'", groupID).
		Count(&count).Error
	return count > 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
