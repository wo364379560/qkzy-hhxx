package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
)

type Event struct {
	Key     string
	Title   string
	Fields  map[string]string
	Message string
}

type localDedupeEntry struct {
	expiresAt time.Time
}

const localDedupeMaxEntries = 10000

var localDedupe = struct {
	sync.Mutex
	entries map[string]localDedupeEntry
}{
	entries: make(map[string]localDedupeEntry),
}

func NotifyAsync(ctx context.Context, event Event) {
	opts := config.Cfg.Get().TelegramAlert
	if !opts.Enabled || strings.TrimSpace(opts.BotToken) == "" || strings.TrimSpace(opts.ChatID) == "" {
		return
	}
	if strings.TrimSpace(event.Key) == "" {
		return
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		if !allowDedupe(bgCtx, opts.DedupeTTL, event.Key) {
			return
		}
		if err := sendTelegram(bgCtx, opts.BotToken, opts.ChatID, buildMessage(opts.PlatformName, event)); err != nil {
			logger.Warnf("telegram alert send failed: key=%s err=%v", event.Key, err)
		}
	}()
	_ = ctx
}

func AdminLogin(ctx context.Context, username, stage, result, reason, ip, userAgent string) {
	key := fmt.Sprintf("admin_login:%s:%s:%s:%s", sanitizeKey(username), sanitizeKey(stage), sanitizeKey(result), sanitizeKey(ip))
	NotifyAsync(ctx, Event{
		Key:   key,
		Title: "管理后台登录通知",
		Fields: map[string]string{
			"用户名": username,
			"阶段":  stage,
			"结果":  result,
			"原因":  reason,
			"IP":  ip,
			"UA":  userAgent,
		},
	})
}

func allowDedupe(ctx context.Context, ttl time.Duration, key string) bool {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	redisKey := "telegram_alert:dedupe:" + key
	if datasource.Datasource != nil && datasource.Datasource.Redis != nil {
		client := datasource.Datasource.Redis.GetClient()
		if client != nil {
			ok, err := client.SetNX(ctx, redisKey, "1", ttl).Result()
			if err == nil {
				return ok
			}
			logger.Warnf("telegram alert redis dedupe failed: key=%s err=%v", key, err)
		}
	}
	return allowLocalDedupe(key, ttl, time.Now())
}

func allowLocalDedupe(key string, ttl time.Duration, now time.Time) bool {
	localDedupe.Lock()
	defer localDedupe.Unlock()

	for entryKey, entry := range localDedupe.entries {
		if !entry.expiresAt.After(now) {
			delete(localDedupe.entries, entryKey)
		}
	}
	if entry, ok := localDedupe.entries[key]; ok && entry.expiresAt.After(now) {
		return false
	}
	if len(localDedupe.entries) >= localDedupeMaxEntries {
		logger.Warnf("telegram alert local dedupe is full; dropping alert key=%s", key)
		return false
	}
	localDedupe.entries[key] = localDedupeEntry{expiresAt: now.Add(ttl)}
	return true
}

func sendTelegram(ctx context.Context, botToken, chatID, text string) error {
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", strings.TrimSpace(botToken))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram api status=%d", resp.StatusCode)
	}
	return nil
}

func buildMessage(platformName string, event Event) string {
	platformName = strings.TrimSpace(platformName)
	if platformName == "" {
		platformName = "本平台"
	}
	lines := []string{fmt.Sprintf("【%s】%s", platformName, strings.TrimSpace(event.Title))}
	for _, key := range []string{"用户名", "阶段", "结果", "原因", "IP", "UA"} {
		value := strings.TrimSpace(event.Fields[key])
		if value != "" {
			lines = append(lines, fmt.Sprintf("%s：%s", key, truncate(value, 220)))
		}
	}
	if msg := strings.TrimSpace(event.Message); msg != "" {
		lines = append(lines, truncate(msg, 500))
	}
	lines = append(lines, "时间："+time.Now().Format("2006-01-02 15:04:05"))
	return strings.Join(lines, "\n")
}

func sanitizeKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(" ", "_", "\n", "_", "\r", "_", "\t", "_", "/", "_", ":", "_")
	return truncate(replacer.Replace(value), 120)
}

func truncate(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
