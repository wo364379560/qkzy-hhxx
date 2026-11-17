// Package sms 封装短信验证码发送（阿里云）与数据库存储。
// provider=dev 时仅打印日志，不真实发送（开发/测试环境）。
package sms

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	codeLength                  = 6
	codeTTL                     = 5 * time.Minute
	maxCodeVerificationAttempts = 5
)

const defaultSMSSendScene = "general"

type RateLimitErr string

func (e RateLimitErr) Error() string { return string(e) }

type ValidationErr string

func (e ValidationErr) Error() string { return string(e) }

// VerificationErr is a user-facing invalid/expired-code result. Database and
// transaction failures remain ordinary errors so handlers can return a generic 500.
type VerificationErr string

func (e VerificationErr) Error() string { return string(e) }

// VerificationTicket represents a code that was valid when checked. Its
// fields are private so callers can only pass a ticket back to Consume.
type VerificationTicket struct {
	phone  string
	code   string
	bypass bool
}

type AliyunErr struct {
	Code    string
	Message string
}

func (e AliyunErr) Error() string {
	return fmt.Sprintf("sms: aliyun error [%s]: %s", e.Code, e.Message)
}

type SendInput struct {
	Phone    string
	DeviceID string
	Scene    string
	IP       string
}

// smsCode 短信验证码表结构（内存表或临时表）
type smsCode struct {
	Phone          string    `gorm:"primaryKey;size:20"`
	Code           string    `gorm:"size:10;not null"`
	FailedAttempts int       `gorm:"not null;default:0"`
	ExpiresAt      time.Time `gorm:"not null;index"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (smsCode) TableName() string {
	return "sms_codes"
}

// MigrateSchema creates the SMS-owned tables during application startup.
// Schema migration must not run in the request path because it is expensive
// and can amplify abusive traffic to the SMS endpoint.
func MigrateSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("sms: mysql datasource not initialized")
	}
	if err := db.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci").
		AutoMigrate(&smsCode{}); err != nil {
		return err
	}
	var tableInfo struct {
		Engine string `gorm:"column:engine"`
	}
	if err := db.Raw(`
		SELECT ENGINE AS engine
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?
	`, (smsCode{}).TableName()).Scan(&tableInfo).Error; err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(tableInfo.Engine), "InnoDB") {
		return db.Exec("ALTER TABLE sms_codes ENGINE=InnoDB").Error
	}
	return nil
}

func isValidMainlandMobile(phone string) bool {
	if len(phone) != 11 || phone[0] != '1' {
		return false
	}
	for _, ch := range phone {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isAliyunPhoneIllegal(err error) bool {
	var aliyunErr AliyunErr
	return errors.As(err, &aliyunErr) && aliyunErr.Code == "isv.MOBILE_NUMBER_ILLEGAL"
}

func isAliyunFrequencyLimited(err error) bool {
	var aliyunErr AliyunErr
	if !errors.As(err, &aliyunErr) {
		return false
	}
	switch aliyunErr.Code {
	case "isv.BUSINESS_LIMIT_CONTROL", "isv.MOBILE_COUNT_OVER_LIMIT":
		return true
	default:
		return false
	}
}

// Send 生成验证码，按 provider 发送短信，并将验证码存入数据库（5 分钟有效）。
func Send(ctx context.Context, input SendInput) error {
	phone := strings.TrimSpace(input.Phone)
	deviceID := strings.TrimSpace(input.DeviceID)
	scene := strings.TrimSpace(input.Scene)
	ipAddress := strings.TrimSpace(input.IP)
	if scene == "" {
		scene = defaultSMSSendScene
	}
	if !isValidMainlandMobile(phone) {
		return ValidationErr("手机号格式错误")
	}

	code, err := genCode(codeLength)
	if err != nil {
		return fmt.Errorf("sms: gen code: %w", err)
	}

	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)

	if err := enforceRateLimit(ctx, db, phone, ipAddress, deviceID, scene); err != nil {
		recordSMSSendLog(ctx, db, phone, ipAddress, deviceID, scene, config.Cfg.Get().SMS.Provider, "rejected", err.Error())
		return err
	}

	cfg := config.Cfg.Get().SMS
	if cfg.Provider == "aliyun" && cfg.AccessKeyID != "" {
		if err = sendViaAliyun(phone, code, cfg); err != nil {
			recordSMSSendLog(ctx, db, phone, ipAddress, deviceID, scene, cfg.Provider, "failed", err.Error())
			if isAliyunPhoneIllegal(err) {
				return ValidationErr("手机号格式错误")
			}
			if isAliyunFrequencyLimited(err) {
				return RateLimitErr("验证码发送过于频繁，请稍后再试")
			}
			return err
		}
	} else {
		// dev 模式：打印到日志，方便调试
		logger.Infof("[DEV SMS] phone=%s code=%s", phone, code)
	}

	// 单条原子 upsert，避免同手机号的并发发送在 DELETE + CREATE 之间
	// 发生重复主键，导致短信已经发出但验证码没有落库。
	expiresAt := time.Now().Add(codeTTL)
	record := smsCode{
		Phone:          phone,
		Code:           code,
		FailedAttempts: 0,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "phone"}},
		DoUpdates: clause.AssignmentColumns([]string{"code", "failed_attempts", "expires_at", "created_at"}),
	}).Create(&record).Error; err != nil {
		return fmt.Errorf("sms: save code: %w", err)
	}
	recordSMSSendLog(ctx, db, phone, ipAddress, deviceID, scene, cfg.Provider, "success", "")

	return nil
}

// Validate checks a code without consuming a valid one. Wrong attempts are
// still counted. Call Consume after the protected CPU work succeeds.
func Validate(ctx context.Context, phone, code string) (*VerificationTicket, error) {
	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)
	phone = strings.TrimSpace(phone)
	code = strings.TrimSpace(code)
	var verificationErr error
	err := db.Transaction(func(tx *gorm.DB) error {
		var record smsCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("phone = ? AND expires_at > ?", phone, time.Now()).
			First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return VerificationErr("验证码已过期或不存在")
			}
			return err
		}

		if subtle.ConstantTimeCompare([]byte(record.Code), []byte(code)) != 1 {
			verificationErr = VerificationErr("验证码错误")
			if record.FailedAttempts+1 >= maxCodeVerificationAttempts {
				return tx.Delete(&record).Error
			}
			return tx.Model(&smsCode{}).
				Where("phone = ?", record.Phone).
				UpdateColumn("failed_attempts", gorm.Expr("failed_attempts + ?", 1)).Error
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if verificationErr != nil {
		return nil, verificationErr
	}
	return &VerificationTicket{phone: phone, code: code}, nil
}

// Consume atomically consumes a previously validated real code. Concurrent
// requests may validate the same code, but only one can delete it successfully.
func Consume(ctx context.Context, ticket *VerificationTicket) error {
	if ticket == nil {
		return VerificationErr("验证码已过期或不存在")
	}
	if ticket.bypass {
		return nil
	}
	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)
	result := db.Where("phone = ? AND code = ? AND expires_at > ?", ticket.phone, ticket.code, time.Now()).
		Delete(&smsCode{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return VerificationErr("验证码已过期或不存在")
	}
	return nil
}

// Verify validates and consumes a real code in one call.
func Verify(ctx context.Context, phone, code string) error {
	ticket, err := Validate(ctx, phone, code)
	if err != nil {
		return err
	}
	return Consume(ctx, ticket)
}

// ValidateOrBypass validates the process-gated bypass or a real SMS code
// without consuming the real code.
func ValidateOrBypass(ctx context.Context, phone, code string) (*VerificationTicket, error) {
	settings, err := config.GetSMSBypassSettings(ctx, datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}

	if settings.Enabled && settings.Code != "" &&
		subtle.ConstantTimeCompare([]byte(settings.Code), []byte(strings.TrimSpace(code))) == 1 {
		return &VerificationTicket{bypass: true}, nil
	}

	return Validate(ctx, phone, code)
}

// VerifyOrBypass validates and consumes a code in one call.
func VerifyOrBypass(ctx context.Context, phone, code string) error {
	ticket, err := ValidateOrBypass(ctx, phone, code)
	if err != nil {
		return err
	}
	return Consume(ctx, ticket)
}

// CleanExpired 清理过期验证码（可定期调用）
func CleanExpired(ctx context.Context) error {
	db := datasource.Datasource.Mysql.GetClient().WithContext(ctx)
	return db.Where("expires_at <= ?", time.Now()).Delete(&smsCode{}).Error
}

func enforceRateLimit(ctx context.Context, db *gorm.DB, phone, ipAddress, deviceID, scene string) error {
	repo := repository.NewSMSSendLogRepositoryWithDB(db.WithContext(ctx))
	now := time.Now()
	settings, err := config.GetSMSRateLimitSettings(ctx, db)
	if err != nil {
		return fmt.Errorf("sms: load rate limit settings: %w", err)
	}
	if !settings.Enabled {
		return nil
	}

	appliedSettings := settings
	switch settings.StrategyMode {
	case config.SMSRateLimitStrategyAlwaysOn:
		appliedSettings = buildStrictSMSRateLimitSettings(settings)
	case config.SMSRateLimitStrategyRiskBased:
		triggered, triggerErr := isSMSAbuseAnomalyTriggered(repo, now, settings)
		if triggerErr != nil {
			return fmt.Errorf("sms: evaluate anomaly strategy: %w", triggerErr)
		}
		if triggered {
			appliedSettings = buildStrictSMSRateLimitSettings(settings)
		}
	}

	latest, err := repo.FindLatestSuccessByPhone(phone)
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("sms: query phone cooldown: %w", err)
	}
	phoneCooldown := time.Duration(appliedSettings.PhoneCooldownSeconds) * time.Second
	if err == nil && now.Sub(latest.CreatedAt) < phoneCooldown {
		remain := int((phoneCooldown - now.Sub(latest.CreatedAt)).Seconds())
		if remain < 1 {
			remain = 1
		}
		return RateLimitErr(fmt.Sprintf("发送过于频繁，请 %d 秒后再试", remain))
	}

	phoneHourCount, err := repo.CountByPhoneSince(phone, now.Add(-1*time.Hour))
	if err != nil {
		return fmt.Errorf("sms: query phone hour limit: %w", err)
	}
	if phoneHourCount >= int64(appliedSettings.PhoneHourLimit) {
		return RateLimitErr("该手机号发送过于频繁，请1小时后再试")
	}

	phoneDayCount, err := repo.CountByPhoneSince(phone, now.Add(-24*time.Hour))
	if err != nil {
		return fmt.Errorf("sms: query phone day limit: %w", err)
	}
	if phoneDayCount >= int64(appliedSettings.PhoneDayLimit) {
		return RateLimitErr("该手机号今日短信次数已达上限")
	}

	if ipAddress != "" {
		ipHourCount, err := repo.CountByIPSince(ipAddress, now.Add(-1*time.Hour))
		if err != nil {
			return fmt.Errorf("sms: query ip hour limit: %w", err)
		}
		if ipHourCount >= int64(appliedSettings.IPHourLimit) {
			return RateLimitErr("当前网络请求过于频繁，请稍后再试")
		}

		ipDayCount, err := repo.CountByIPSince(ipAddress, now.Add(-24*time.Hour))
		if err != nil {
			return fmt.Errorf("sms: query ip day limit: %w", err)
		}
		if ipDayCount >= int64(appliedSettings.IPDayLimit) {
			return RateLimitErr("当前网络今日短信次数已达上限")
		}
	}

	if deviceID != "" {
		deviceHourCount, err := repo.CountByDeviceSince(deviceID, now.Add(-1*time.Hour))
		if err != nil {
			return fmt.Errorf("sms: query device hour limit: %w", err)
		}
		if deviceHourCount >= int64(appliedSettings.DeviceHourLimit) {
			return RateLimitErr("当前设备请求过于频繁，请稍后再试")
		}

		deviceDayCount, err := repo.CountByDeviceSince(deviceID, now.Add(-24*time.Hour))
		if err != nil {
			return fmt.Errorf("sms: query device day limit: %w", err)
		}
		if deviceDayCount >= int64(appliedSettings.DeviceDayLimit) {
			return RateLimitErr("当前设备今日短信次数已达上限")
		}
	}

	_ = scene
	return nil
}

func isSMSAbuseAnomalyTriggered(repo *repository.SMSSendLogRepository, now time.Time, settings config.SMSRateLimitSettings) (bool, error) {
	windowStart := now.Add(-time.Duration(settings.AnomalyWindowMinutes) * time.Minute)
	total, err := repo.CountAllSince(windowStart)
	if err != nil {
		return false, err
	}
	return total >= int64(settings.AnomalyTotalLimit), nil
}

func buildStrictSMSRateLimitSettings(settings config.SMSRateLimitSettings) config.SMSRateLimitSettings {
	settings.PhoneCooldownSeconds = settings.StrictPhoneCooldownSeconds
	settings.PhoneHourLimit = settings.StrictPhoneHourLimit
	settings.PhoneDayLimit = settings.StrictPhoneDayLimit
	settings.IPHourLimit = settings.StrictIPHourLimit
	settings.IPDayLimit = settings.StrictIPDayLimit
	settings.DeviceHourLimit = settings.StrictDeviceHourLimit
	settings.DeviceDayLimit = settings.StrictDeviceDayLimit
	return settings
}

func recordSMSSendLog(ctx context.Context, db *gorm.DB, phone, ipAddress, deviceID, scene, provider, status, rejectReason string) {
	if db == nil {
		return
	}
	log := &model.SMSSendLog{
		Phone:        phone,
		IPAddress:    ipAddress,
		DeviceID:     deviceID,
		Scene:        scene,
		Status:       status,
		RejectReason: rejectReason,
		Provider:     provider,
	}
	if err := repository.NewSMSSendLogRepositoryWithDB(db.WithContext(ctx)).Create(log); err != nil {
		logger.Warnf("record sms send log failed: phone=%s scene=%s status=%s err=%v", phone, scene, status, err)
	}
}

// genCode 生成 n 位数字验证码。
func genCode(n int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*s", n, num.String()), nil
}

// ──────────────────────────────────────────────
// 阿里云 DYSMS API（RPC 风格，HMAC-SHA1 签名）
// ──────────────────────────────────────────────

const aliyunEndpoint = "https://dysmsapi.aliyuncs.com"

func sendViaAliyun(phone, code string, cfg config.SMS) error {
	nonce, err := genNonce()
	if err != nil {
		return fmt.Errorf("sms: gen nonce: %w", err)
	}

	params := map[string]string{
		"Action":           "SendSms",
		"Version":          "2017-05-25",
		"Format":           "JSON",
		"RegionId":         "cn-hangzhou",
		"AccessKeyId":      cfg.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   nonce,
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"PhoneNumbers":     phone,
		"SignName":         cfg.SignName,
		"TemplateCode":     cfg.TemplateCode,
		"TemplateParam":    fmt.Sprintf(`{"code":"%s"}`, code),
	}

	// 1. 排序并编码参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, aliyunEncode(k)+"="+aliyunEncode(params[k]))
	}
	queryStr := strings.Join(parts, "&")

	// 2. 构造待签名字符串
	stringToSign := "POST&%2F&" + aliyunEncode(queryStr)

	// 3. HMAC-SHA1 签名
	mac := hmac.New(sha1.New, []byte(cfg.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	params["Signature"] = sig

	// 4. 构造 POST 表单
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	// 必须带超时：源站访问不了 dysmsapi 时，默认 http.Post 会一直挂起，
	// 导致 App 端 /sms/send 超时、域名池误切备用。
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(aliyunEndpoint, "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("sms: aliyun http: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err = json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("sms: parse response: %w", err)
	}
	if result.Code != "OK" {
		return AliyunErr{Code: result.Code, Message: result.Message}
	}
	return nil
}

// aliyunEncode 按阿里云规范进行 URL 编码（RFC 3986，空格→%20，*→%2A，~→~）。
func aliyunEncode(s string) string {
	encoded := url.QueryEscape(s)
	// QueryEscape 将空格编码为 +，阿里云要求 %20
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	// QueryEscape 不编码 *，阿里云要求 %2A
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	// QueryEscape 将 ~ 编码为 %7E，阿里云要求保持 ~
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// genNonce 生成随机唯一串（无需外部依赖）。
func genNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
