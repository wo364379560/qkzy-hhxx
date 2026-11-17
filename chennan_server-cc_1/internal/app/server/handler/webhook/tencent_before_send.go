package handler_webhook

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/grouprisk"
	"github.com/ixayda/ix-Go/internal/app/server/groupslowmode"
	"github.com/ixayda/ix-Go/internal/app/server/keywordfilter"
	"github.com/ixayda/ix-Go/internal/app/server/migrationfence"
	"github.com/ixayda/ix-Go/internal/app/server/quietlog"
	"github.com/ixayda/ix-Go/internal/app/server/quietpolicy"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/app/server/usercache"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

const (
	cmdC2CBeforeSendMsg   = "C2C.CallbackBeforeSendMsg"
	cmdGroupBeforeSendMsg = "Group.CallbackBeforeSendMsg"
	cmdC2CAfterSendMsg    = "C2C.CallbackAfterSendMsg"
	cmdGroupAfterSendMsg  = "Group.CallbackAfterSendMsg"

	quietHoursRejectCodeSendMessage      = 120001
	groupSlowModeRejectCodeSendMessage   = 120005
	sourceMigrationRejectCodeSendMessage = 120099
)

type beforeSendMsgReq struct {
	CallbackCommand string                     `json:"CallbackCommand"`
	FromAccount     string                     `json:"From_Account"`
	OperatorAccount string                     `json:"Operator_Account"`
	ToAccount       string                     `json:"To_Account"`
	GroupID         string                     `json:"GroupId"`
	CloudCustomData string                     `json:"CloudCustomData"`
	MsgBody         []tencentim.MessageElement `json:"MsgBody"`
}

type beforeSendMsgResp struct {
	ActionStatus string                     `json:"ActionStatus"`
	ErrorCode    int                        `json:"ErrorCode"`
	ErrorInfo    string                     `json:"ErrorInfo"`
	MsgBody      []tencentim.MessageElement `json:"MsgBody,omitempty"`
}

func handleBeforeSendMsg(c *gin.Context) {
	var req beforeSendMsgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, beforeSendMsgResp{ActionStatus: "FAIL", ErrorCode: 1, ErrorInfo: "invalid request"})
		return
	}
	if migrationfence.Active() {
		c.JSON(http.StatusOK, beforeSendMsgResp{
			ActionStatus: "OK",
			ErrorCode:    sourceMigrationRejectCodeSendMessage,
			ErrorInfo:    "系统迁移维护中，请稍后重试",
		})
		return
	}
	if req.CallbackCommand == cmdGroupBeforeSendMsg &&
		isTencentServerOperator(req.OperatorAccount) &&
		req.CloudCustomData == tencentim.ServerSystemNoticeCloudCustomData {
		c.JSON(http.StatusOK, beforeSendMsgResp{ActionStatus: "OK", ErrorCode: 0, ErrorInfo: ""})
		return
	}
	if req.CallbackCommand == cmdGroupBeforeSendMsg && req.GroupID != "" {
		decision, decisionErr := grouprisk.HandleBeforeSendGroupMessage(c.Request.Context(), grouprisk.BeforeSendInput{
			CallbackCommand: req.CallbackCommand,
			GroupID:         req.GroupID,
			FromAccount:     req.FromAccount,
		})
		if decisionErr != nil {
			logger.Warnf("evaluate quiet-hours group risk failed: command=%s from=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.GroupID, decisionErr)
		} else if decision.Blocked {
			c.JSON(http.StatusOK, beforeSendMsgResp{
				ActionStatus: "OK",
				ErrorCode:    quietHoursRejectCodeSendMessage,
				ErrorInfo:    decision.Reason,
			})
			return
		}

		slowDecision, slowErr := groupslowmode.CheckBeforeSend(c.Request.Context(), req.GroupID, req.FromAccount)
		if slowErr != nil {
			logger.Warnf("evaluate group slow mode failed: command=%s from=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.GroupID, slowErr)
		} else if slowDecision.Blocked {
			c.JSON(http.StatusOK, beforeSendMsgResp{
				ActionStatus: "OK",
				ErrorCode:    groupSlowModeRejectCodeSendMessage,
				ErrorInfo:    slowDecision.Reason,
			})
			return
		}
	}

	userRepo := repository.NewUserRepository()
	cacheGeneration := usercache.Generation(c.Request.Context(), req.FromAccount)
	sender, cachedSender := usercache.GetByIMUserID(c.Request.Context(), req.FromAccount)
	var err error
	if !cachedSender {
		sender, err = userRepo.FindByIMUserID(req.FromAccount)
	}
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnf("load sender for quiet hours failed: command=%s from=%s to=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.ToAccount, req.GroupID, err)
		}
	} else {
		if !cachedSender {
			usercache.SetByIMUserID(c.Request.Context(), sender, cacheGeneration)
		}

		resolution, resolveErr := quietpolicy.ResolveChatQuietHoursPolicyForOwner(c.Request.Context(), sender.OwnerID)
		if resolveErr != nil {
			logger.Warnf("resolve quiet hours policy failed: command=%s from=%s to=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.ToAccount, req.GroupID, resolveErr)
		} else if resolution.HasPolicy {
			blocked, blockErr := config.IsChatQuietHoursActionBlockedNow(resolution.Policy, config.ChatQuietHoursActionSendMessage, time.Now())
			if blockErr != nil {
				logger.Warnf("evaluate quiet hours policy failed: command=%s from=%s to=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.ToAccount, req.GroupID, blockErr)
			} else if blocked {
				reason := config.ChatQuietHoursBlockedMessage(config.ChatQuietHoursActionSendMessage, resolution.Policy)
				_ = quietlog.RecordChatQuietHoursHit(quietlog.ChatQuietHoursHitInput{
					Action:          "send_message",
					Source:          "im_before_send",
					CallbackCommand: req.CallbackCommand,
					ActorIMUserID:   req.FromAccount,
					TargetAccount:   req.ToAccount,
					TargetGroupID:   req.GroupID,
					Reason:          reason,
				})
				c.JSON(http.StatusOK, beforeSendMsgResp{
					ActionStatus: "OK",
					ErrorCode:    quietHoursRejectCodeSendMessage,
					ErrorInfo:    reason,
				})
				return
			}
		}
	}

	shouldFilterSender, filterSenderErr := keywordfilter.ShouldFilterSender(req.FromAccount)
	if filterSenderErr != nil {
		logger.Warnf("load keyword filter sender failed: command=%s from=%s to=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.ToAccount, req.GroupID, filterSenderErr)
	} else if !shouldFilterSender {
		c.JSON(http.StatusOK, beforeSendMsgResp{ActionStatus: "OK", ErrorCode: 0, ErrorInfo: ""})
		return
	}

	maskedBody, changed, err := keywordfilter.MaskMessageElements(req.MsgBody)
	if err != nil {
		logger.Warnf("mask message keyword failed: command=%s from=%s to=%s group=%s err=%v", req.CallbackCommand, req.FromAccount, req.ToAccount, req.GroupID, err)
		c.JSON(http.StatusOK, beforeSendMsgResp{ActionStatus: "OK", ErrorCode: 0, ErrorInfo: ""})
		return
	}

	if !changed {
		c.JSON(http.StatusOK, beforeSendMsgResp{ActionStatus: "OK", ErrorCode: 0, ErrorInfo: ""})
		return
	}

	c.JSON(http.StatusOK, beforeSendMsgResp{
		ActionStatus: "OK",
		ErrorCode:    0,
		ErrorInfo:    "",
		MsgBody:      maskedBody,
	})
}

// handleAfterSendMsg acknowledges existing Tencent callback configuration.
// The retired unverified-message quota no longer records any sends.
func handleAfterSendMsg(c *gin.Context) {
	c.JSON(http.StatusOK, beforeSendMsgResp{ActionStatus: "OK", ErrorCode: 0, ErrorInfo: ""})
}
