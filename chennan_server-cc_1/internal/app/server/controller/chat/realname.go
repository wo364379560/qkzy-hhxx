package controller_chat

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/messagelimit"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/cloudauth"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

var idNumberPattern = regexp.MustCompile(`^[1-9]\d{5}(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)

// AntiFraudNoticeResult App 端防诈公告响应。
type AntiFraudNoticeResult struct {
	Enabled      bool   `json:"enabled"`
	Content      string `json:"content"`
	CountdownSec int    `json:"countdown_sec"`
}

// GetAntiFraudNotice 获取当前防诈公告配置。
func GetAntiFraudNotice() (*AntiFraudNoticeResult, error) {
	settings, err := config.GetAntiFraudNoticeSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}

	result := &AntiFraudNoticeResult{
		Enabled:      settings.Enabled,
		CountdownSec: settings.CountdownSec,
	}
	if settings.Enabled {
		result.Content = settings.Content
	}
	return result, nil
}

// RealnameStatusResult 用户实名状态响应。
type RealnameStatusResult struct {
	FeatureEnabled bool   `json:"feature_enabled"`
	Verified       bool   `json:"verified"`
	RealNameMasked string `json:"real_name_masked"`
	IDNumberMasked string `json:"id_number_masked"`
}

// GetRealnameStatus 获取当前用户实名状态。
func GetRealnameStatus(imUserID string) (*RealnameStatusResult, error) {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	featureEnabled, err := config.GetRealnameVerifyEnabled(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}

	result := &RealnameStatusResult{FeatureEnabled: featureEnabled}
	record, err := repository.NewUserRealNameRepository().FindByUserID(user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		return nil, err
	}
	if record.Status == model.UserRealNameStatusVerified {
		result.Verified = true
		result.RealNameMasked = cloudauth.MaskRealName(record.RealName)
		result.IDNumberMasked = record.IDNumberMasked
	}
	return result, nil
}

// PeerRealnameItem 对方实名状态（仅布尔，不含证件信息）。
type PeerRealnameItem struct {
	IMUserID string `json:"im_user_id"`
	Verified bool   `json:"verified"`
}

// GetPeerRealnameStatus 批量查询对方是否已实名（功能关闭时全部视为已实名，避免误提示）。
func GetPeerRealnameStatus(peerIMUserIDs []string) ([]PeerRealnameItem, error) {
	featureEnabled, err := config.GetRealnameVerifyEnabled(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(peerIMUserIDs))
	seen := map[string]struct{}{}
	for _, id := range peerIMUserIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) >= 50 {
			break
		}
	}

	out := make([]PeerRealnameItem, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	if !featureEnabled {
		for _, id := range ids {
			out = append(out, PeerRealnameItem{IMUserID: id, Verified: true})
		}
		return out, nil
	}

	users, err := repository.NewUserRepository().FindByIMUserIDs(ids)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint, 0, len(users))
	imByUserID := make(map[uint]string, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
		imByUserID[u.ID] = u.IMUserID
	}
	verifiedSet := map[string]struct{}{}
	if len(userIDs) > 0 {
		records, err := repository.NewUserRealNameRepository().FindByUserIDs(userIDs)
		if err != nil {
			return nil, err
		}
		for uid, rec := range records {
			if rec != nil && rec.Status == model.UserRealNameStatusVerified {
				if im, ok := imByUserID[uid]; ok {
					verifiedSet[im] = struct{}{}
				}
			}
		}
	}
	for _, id := range ids {
		_, ok := verifiedSet[id]
		out = append(out, PeerRealnameItem{IMUserID: id, Verified: ok})
	}
	return out, nil
}

// VerifyRealnameReq 实名认证请求。
type VerifyRealnameReq struct {
	RealName string
	IDNumber string
}

// VerifyRealname 提交姓名与身份证号进行二要素核验。
func VerifyRealname(imUserID string, req VerifyRealnameReq) error {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return err
	}
	if user.Status == "deleted" {
		return errors.New("账号已注销")
	}

	enabled, err := config.GetRealnameVerifyEnabled(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return err
	}
	if !enabled {
		return errors.New("实名认证功能未开启")
	}

	realName := strings.TrimSpace(req.RealName)
	idNumber := strings.ToUpper(strings.TrimSpace(req.IDNumber))
	if realName == "" {
		return errors.New("姓名不能为空")
	}
	if !idNumberPattern.MatchString(idNumber) {
		return errors.New("身份证号格式无效")
	}

	existing, err := repository.NewUserRealNameRepository().FindByUserID(user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil && existing.Status == model.UserRealNameStatusVerified {
		return errors.New("已完成实名认证")
	}

	idHash := cloudauth.HashIDNumber(idNumber)
	occupied, err := repository.NewUserRealNameRepository().FindVerifiedByIDNumberHash(idHash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if occupied != nil && occupied.UserID != user.ID {
		return errors.New("该身份证号已被其他账号认证")
	}

	creds, err := cloudauth.LoadCredentialsFromEnv()
	if err != nil {
		if errors.Is(err, cloudauth.ErrNotConfigured) {
			return cloudauth.ErrNotConfigured
		}
		return err
	}

	verifyResult, err := cloudauth.Id2MetaVerify(creds, realName, idNumber)
	if err != nil {
		return err
	}

	idMasked := cloudauth.MaskIDNumber(idNumber)
	repo := repository.NewUserRealNameRepository()
	if !verifyResult.Matched {
		if saveErr := repo.SaveFailed(user.ID, idHash, idMasked); saveErr != nil {
			return saveErr
		}
		messagelimit.InvalidateRealNameVerified(context.Background(), user.ID)
		return errors.New("姓名与身份证号不一致")
	}

	now := time.Now()
	if err := repo.UpsertVerified(&model.UserRealName{
		UserID:          user.ID,
		RealName:        realName,
		IDNumberHash:    idHash,
		IDNumberMasked:  idMasked,
		Status:          model.UserRealNameStatusVerified,
		VerifiedAt:      &now,
		AliyunRequestID: verifyResult.RequestID,
	}); err != nil {
		return err
	}
	messagelimit.InvalidateRealNameVerified(context.Background(), user.ID)
	return nil
}
