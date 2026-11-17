package forwardqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// These are forwarding budgets, not claims that a customer's purchased quota
// was inspected. Keep all processes for one SDKAppID on the same configuration.
// Tencent documents default 200/s per REST command, and 40/s per group:
// https://cloud.tencent.com/document/product/269/2282
// https://cloud.tencent.com/document/product/269/1629
// https://cloud.tencent.com/document/product/269/1616
// https://cloud.tencent.com/document/product/269/112002
// https://cloud.tencent.com/document/product/269/32429
// Other app features also consume these quotas; defaults deliberately leave room.
type PacingSettings struct {
	SDKAppID             int    `json:"sdk_app_id"`
	C2CRPS               int    `json:"c2c_send_rps"`
	GroupRPS             int    `json:"group_send_rps"`
	GroupInfoRPS         int    `json:"group_info_rps"`
	GroupMemberRPS       int    `json:"group_member_rps"`
	PerGroupRPS          int    `json:"per_group_rps"`
	AppRPS               int    `json:"app_rps"`
	SenderRPS            int    `json:"sender_rps"`
	Workers              int    `json:"workers"`
	MaterializeHighWater int    `json:"materialize_high_water"`
	MaterializeBatch     int    `json:"materialize_batch"`
	SDKSendIntervalMS    int    `json:"sdk_send_interval_ms"`
	SDKPrepareIntervalMS int    `json:"sdk_prepare_interval_ms"`
	SDKConcurrency       int    `json:"sdk_concurrency"`
	QuotaSource          string `json:"quota_source"`
	ConfigValid          bool   `json:"config_valid"`
}

func defaultPacing() PacingSettings {
	return PacingSettings{SDKAppID: config.Cfg.Get().TencentIM.SDKAppID, C2CRPS: 100, GroupRPS: 100, GroupInfoRPS: 100, GroupMemberRPS: 100, PerGroupRPS: 20, Workers: 8, MaterializeHighWater: 2000, MaterializeBatch: 100, SDKSendIntervalMS: 1000, SDKPrepareIntervalMS: 1000, SDKConcurrency: 4, QuotaSource: "public_defaults_with_headroom_not_account_verified", ConfigValid: true}
}

func LoadPacingSettings() (PacingSettings, error) {
	s := defaultPacing()
	for _, v := range []struct {
		name     string
		value    *int
		min, max int
	}{
		{"FORWARD_C2C_SEND_RPS", &s.C2CRPS, 1, 10000}, {"FORWARD_GROUP_SEND_RPS", &s.GroupRPS, 1, 10000},
		{"FORWARD_GROUP_INFO_RPS", &s.GroupInfoRPS, 1, 10000}, {"FORWARD_GROUP_MEMBER_RPS", &s.GroupMemberRPS, 1, 10000},
		{"FORWARD_PER_GROUP_RPS", &s.PerGroupRPS, 1, 40}, {"FORWARD_APP_RPS", &s.AppRPS, 0, 10000}, {"FORWARD_SENDER_RPS", &s.SenderRPS, 0, 10000},
		{"FORWARD_WORKERS", &s.Workers, 1, 64}, {"FORWARD_MATERIALIZE_HIGH_WATER", &s.MaterializeHighWater, 100, 100000},
		{"FORWARD_MATERIALIZE_BATCH", &s.MaterializeBatch, 1, 200},
		{"FORWARD_SDK_SEND_INTERVAL_MS", &s.SDKSendIntervalMS, 50, 60000}, {"FORWARD_SDK_PREPARE_INTERVAL_MS", &s.SDKPrepareIntervalMS, 50, 60000},
		{"FORWARD_SDK_CONCURRENCY", &s.SDKConcurrency, 1, 16},
	} {
		if raw, ok := os.LookupEnv(v.name); ok && strings.TrimSpace(raw) != "" {
			n, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil || n < v.min || n > v.max {
				s.ConfigValid = false
				return s, fmt.Errorf("%s must be between %d and %d", v.name, v.min, v.max)
			}
			*v.value = n
			s.QuotaSource = "explicit_environment_not_account_verified"
		}
	}
	return s, nil
}

// Invalid settings are visible to the UI; workers fail closed instead of
// silently ignoring a mistyped quota and dispatching at an unexpected rate.
func CurrentSettings() PacingSettings { s, _ := LoadPacingSettings(); return s }

type pacingGate struct {
	id       string
	interval int64
}

func (s PacingSettings) gateID(scope string) string {
	return fmt.Sprintf("im:%d:%s", s.SDKAppID, scope)
}
func identityKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}
func paceInterval(rps int) int64 {
	if rps <= 0 {
		return 0
	}
	return int64((1000 + rps - 1) / rps)
}
func sendCommand(conversation string) string {
	if strings.HasPrefix(conversation, "GROUP") {
		return "send_group_msg"
	}
	return "sendmsg"
}
func (s PacingSettings) gates(command, sender, conversation string) ([]pacingGate, error) {
	var rps int
	switch command {
	case "sendmsg":
		rps = s.C2CRPS
	case "send_group_msg":
		rps = s.GroupRPS
	case "get_group_info":
		rps = s.GroupInfoRPS
	case "get_specified_group_member_info":
		rps = s.GroupMemberRPS
	default:
		return nil, errors.New("unsupported forwarding command")
	}
	// The app gate remains present at interval zero so an explicit Tencent 60011
	// pause is honored by both permission reads and sends, even without app quota.
	gates := []pacingGate{{s.gateID("app"), paceInterval(s.AppRPS)}, {s.gateID("api:" + command), paceInterval(rps)}}
	if command == "sendmsg" || command == "send_group_msg" {
		if s.SenderRPS > 0 {
			gates = append(gates, pacingGate{s.gateID("sender:" + identityKey(sender)), paceInterval(s.SenderRPS)})
		}
		if command == "send_group_msg" {
			gates = append(gates, pacingGate{s.gateID("group:" + identityKey(conversation)), paceInterval(s.PerGroupRPS)})
		}
	}
	sort.Slice(gates, func(i, j int) bool { return gates[i].id < gates[j].id })
	return gates, nil
}

// Reserve all applicable dimensions atomically in consistent order. Waiting
// never consumes another dimension's capacity. Database time avoids clock skew
// between workers and application instances. No gate is keyed by client IP.
func (q Queue) reservePacing(ctx context.Context, gates []pacingGate) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var wait time.Duration
	err := q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		held := make([]model.ForwardGate, len(gates))
		for i, g := range gates {
			gate := model.ForwardGate{ID: g.id}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&gate).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&gate).Error; err != nil {
				return err
			}
			held[i] = gate
		}
		var now int64
		if err := tx.Raw("SELECT CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS SIGNED)").Scan(&now).Error; err != nil {
			return err
		}
		for _, g := range held {
			if g.NextAt > now {
				wait = max(wait, time.Duration(g.NextAt-now)*time.Millisecond)
			}
		}
		if wait > 0 {
			return nil
		}
		for i, g := range gates {
			if g.interval > 0 {
				if err := tx.Model(&held[i]).Update("next_at", now+g.interval).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	return wait, err
}
func (q Queue) slot(ctx context.Context, item *model.ForwardDelivery) (time.Duration, error) {
	s, err := LoadPacingSettings()
	if err != nil {
		return 0, err
	}
	gates, err := s.gates(sendCommand(item.ConversationID), item.Sender, item.ConversationID)
	if err != nil {
		return 0, err
	}
	return q.reservePacing(ctx, gates)
}
func (q Queue) querySlot(ctx context.Context, command string) (time.Duration, error) {
	if command != "get_group_info" && command != "get_specified_group_member_info" {
		return 0, ErrInvalid
	}
	s, err := LoadPacingSettings()
	if err != nil {
		return 0, err
	}
	gates, err := s.gates(command, "", "")
	if err != nil {
		return 0, err
	}
	return q.reservePacing(ctx, gates)
}
func (q Queue) waitQuery(ctx context.Context, command string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		wait, err := q.querySlot(ctx, command)
		if err != nil {
			return err
		}
		if wait == 0 {
			return nil
		}
		if !pause(ctx, wait) {
			return ctx.Err()
		}
	}
}

func (s PacingSettings) cooldownGate(err error, sent bool, item *model.ForwardDelivery) string {
	if !isRateError(err) {
		return ""
	}
	var e *tencentim.DeliveryError
	_ = errors.As(err, &e)
	if e.Code == 60011 {
		return s.gateID("app")
	}
	if sent && e.Code == 10023 && strings.HasPrefix(item.ConversationID, "GROUP") {
		return s.gateID("group:" + identityKey(item.ConversationID))
	}
	command := e.Command
	if command == "" && sent {
		command = sendCommand(item.ConversationID)
	}
	// Commands have documented quotas. Generic frequency codes are not evidence
	// for merging unrelated commands, users or all queries into a global bucket.
	if _, err := s.gates(command, item.Sender, item.ConversationID); err != nil {
		return ""
	}
	return s.gateID("api:" + command)
}
