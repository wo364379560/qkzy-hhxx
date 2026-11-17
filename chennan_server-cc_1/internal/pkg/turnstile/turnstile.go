package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/config"
)

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func Verify(ctx context.Context, token, remoteIP string) error {
	return VerifyWithConfig(ctx, config.Cfg.Get().Turnstile, token, remoteIP)
}

func VerifyWithConfig(ctx context.Context, cfg config.Turnstile, token, remoteIP string) error {
	if !cfg.Enabled {
		return nil
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("请完成人机验证")
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return errors.New("人机验证未配置")
	}

	verifyURL := strings.TrimSpace(cfg.VerifyURL)
	if verifyURL == "" {
		verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}

	form := url.Values{}
	form.Set("secret", cfg.SecretKey)
	form.Set("response", token)
	if strings.TrimSpace(remoteIP) != "" {
		form.Set("remoteip", strings.TrimSpace(remoteIP))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.New("人机验证失败，请稍后重试")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.New("人机验证失败，请稍后重试")
	}

	var payload siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return errors.New("人机验证失败，请稍后重试")
	}
	if !payload.Success {
		return errors.New("人机验证未通过")
	}
	if !hostnameAllowed(payload.Hostname, cfg) {
		return fmt.Errorf("人机验证来源不匹配")
	}
	return nil
}

func hostnameAllowed(hostname string, cfg config.Turnstile) bool {
	hostname = normalizeHostname(hostname)
	if hostname == "" {
		return true
	}

	expected := append([]string{}, cfg.ExpectedHostnames...)
	if cfg.ExpectedHostname != "" {
		expected = append(expected, cfg.ExpectedHostname)
	}
	if len(expected) == 0 {
		return true
	}

	for _, item := range expected {
		if normalizeHostname(item) == hostname {
			return true
		}
	}
	return false
}

func normalizeHostname(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
		raw = parsed.Hostname()
	}
	if host := strings.Split(raw, ":")[0]; host != "" {
		return host
	}
	return raw
}
